package serverstream

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func awaitClosedConnection(conn *websocket.Conn) {
	for {
		_, _, err := conn.NextReader()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				fmt.Printf("client closed connection\n")
			} else {
				fmt.Printf("error reading from client: %v\n", err)
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

func beginProxyLoopAsync(
	conn *websocket.Conn,
	stream GrpcClientStreamFacade,
	streamResponse proto.Message,
) {
	// this go function will return out and die when the stream's context is done
	// we passed the gin context to the openStreamFunc which means that the stream
	// will close when the request is done
	go func() {
		defer fmt.Printf("proxy loop is done\n")
		defer closeConnection(conn, websocket.CloseNormalClosure, "server stream ended")
		for {
			err := stream.RecvMsg(streamResponse)
			if err != nil {
				fmt.Printf("error receiving response from stream: %v\n", err)
				return
			}
			responsePayload, err := protojson.Marshal(streamResponse)
			if err != nil {
				fmt.Printf("error marshalling response: %v\n", err)
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
	go func() {
		<-c.Request.Context().Done()
		fmt.Printf("gin context is done\n")
	}()
	incomingRequest, err := parseRequest(c)
	if err != nil {
		c.String(400, err.Error())
		return
	}
	stream, err := openStreamFunc(c, incomingRequest)
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
	beginProxyLoopAsync(conn, stream, streamResponse)
	// we await a closed connection to know when to stop reading from the stream
	// two actors can close the stream
	// 1. the client can close their connection, by ending the websocket connection
	// 2. the server can close the stream, inside of the proxy loop
	awaitClosedConnection(conn)
}
