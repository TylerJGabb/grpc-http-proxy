package serverstream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func awaitClosedConnection(conn *websocket.Conn) {
	for {
		_, _, err := conn.NextReader()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				fmt.Printf("client closed connection\n")
			} else if strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Printf("connection already closed\n")
			} else {
				fmt.Printf("unexpected error from websocket client: %v\n", err)
			}
			return
		}
	}
}

func closeConnection(conn *websocket.Conn, closeCode int, msg string) {
	err := conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(closeCode, msg),
	)
	if err != nil {
		fmt.Printf("error writing close message to websocket connection: %v\n", err)
	}
	conn.Close()
}

type GrpcClientStreamFacade interface {
	RecvMsg(m any) error
}

func handleStreamError(err error, conn *websocket.Conn) {
	if errors.Is(err, context.Canceled) {
		fmt.Printf("proxy loop context cancelled\n")
	} else if errors.Is(err, io.EOF) {
		fmt.Printf("grpc serverstream ended\n")
		closeConnection(conn, websocket.CloseNormalClosure, "server stream ended")
	} else if grpcStatus, ok := status.FromError(err); ok {
		if grpcStatus.Code() == codes.Canceled {
			fmt.Printf("grpc serverstream cancelled\n")
		} else {
			fmt.Printf("grpc error: %v\n", grpcStatus.Message())
			closeConnection(conn, websocket.CloseInternalServerErr, grpcStatus.Message())
		}
	} else {
		fmt.Printf("error receiving response from stream: %v\n", err)
		closeConnection(conn, websocket.CloseInternalServerErr, err.Error())
	}
}

func beginProxyLoopAsync(
	conn *websocket.Conn,
	stream GrpcClientStreamFacade,
	streamResponse proto.Message,
) {
	// this go function will return out and die when the stream's context is done
	// we passed the gin's request context to the openStreamFunc which means that the stream
	// will close when the request is done
	go func() {
		defer fmt.Printf("proxy loop is done\n")
		for {
			// blocks until a message is received, context is done, or an error occurs
			if err := stream.RecvMsg(streamResponse); err != nil {
				handleStreamError(err, conn)
				return
			}
			responsePayload, err := protojson.Marshal(streamResponse)
			if err != nil {
				fmt.Printf("error marshalling response: %v\n", err)
				closeConnection(conn, websocket.CloseInternalServerErr, err.Error())
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, responsePayload)
			if err != nil {
				fmt.Printf("error writing response to websocket connection: %v\n", err)
				return
			}
		}
	}()
}

func ServerStreamProxy[T, S proto.Message, U GrpcClientStreamFacade](
	c *gin.Context,
	openStreamFunc func(context.Context, T, ...grpc.CallOption) (U, error),
	parseRequest func(c *gin.Context) (T, error),
	streamResponse S,
) {
	fmt.Printf("beginning server stream proxy %p\n", c.Request.Context())
	incomingRequest, err := parseRequest(c)
	if err != nil {
		c.String(400, err.Error())
		return
	}
	stream, err := openStreamFunc(c.Request.Context(), incomingRequest)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	defer conn.Close()
	beginProxyLoopAsync(conn, stream, streamResponse)

	// we await a closed connection before returning
	// two actors can close the stream
	// 1. the client can close their connection, by ending the websocket connection
	// 2. the server can close the stream, inside of the proxy loop
	// once returned, the parent context (inside of gin.Context) will be done

	// killing the goroutine that is running the proxy loop.

	// if the proxy loop was the one that closed the connection,
	// it is a given that the goroutine is already dead
	awaitClosedConnection(conn)
}
