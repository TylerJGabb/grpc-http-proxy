package proxy

import (
	"fmt"
	"strconv"

	"github.com/TylerJGabb/grpc-http-proxy/internal/serverstream"
	"github.com/TylerJGabb/grpc-http-proxy/internal/unary"
	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type OptFunc func(*HttpProxyServer)

func WithGrpcTransportCredentials(
	creds credentials.TransportCredentials,
) OptFunc {
	return func(h *HttpProxyServer) {
		h.transportCredentials = creds
	}
}

func WithPort(
	port int,
) OptFunc {
	return func(h *HttpProxyServer) {
		h.port = port
	}
}

type HttpProxyServer struct {
	port                 int
	grpcServerHost       string
	transportCredentials credentials.TransportCredentials
}

func NewHttpProxyServer(grpcServerHost string, opts ...OptFunc) *HttpProxyServer {
	hps := &HttpProxyServer{
		grpcServerHost:       grpcServerHost,
		port:                 8080,
		transportCredentials: insecure.NewCredentials(),
	}
	for _, optFunc := range opts {
		optFunc(hps)
	}
	return hps
}

func (hps *HttpProxyServer) RunBlocking() error {
	conn, err := grpc.NewClient(hps.grpcServerHost,
		grpc.WithTransportCredentials(hps.transportCredentials),
	)
	if err != nil {
		return err
	}
	client := tgsbpb.NewTylerSandboxServiceClient(conn)
	app := gin.New()

	app.POST("/unarycallint", func(c *gin.Context) {
		unary.ProxyRequest(c, &tgsbpb.UnaryCallIntRequest{}, client.UnaryCallInt)
	})
	app.POST("/unarycallstring", func(c *gin.Context) {
		unary.ProxyRequest(c, &tgsbpb.UnaryCallStringRequest{}, client.UnaryCallString)
	})

	app.GET("/serverstreamstring", func(c *gin.Context) {
		serverstream.ServerStreamProxy(
			c,
			client.ServerStreamString,
			parseStringStreamRequest,
			&tgsbpb.ServerStreamStringRequest{},
		)
	})
	app.GET("/serverstreamint", func(c *gin.Context) {
		serverstream.ServerStreamProxy(
			c,
			client.ServerStreamInt,
			parseIntStreamRequest,
			&tgsbpb.ServerStreamIntRequest{},
		)
	})

	return app.Run(fmt.Sprintf(":%d", hps.port))
}

func parseIntStreamRequest(c *gin.Context) (*tgsbpb.ServerStreamIntRequest, error) {
	incomingRequest := &tgsbpb.ServerStreamIntRequest{}
	value, _ := strconv.ParseInt(c.Query("value"), 10, 32)
	incomingRequest.Value = int32(value)

	sendPeriodSeconds, _ := strconv.ParseUint(c.Query("sendPeriodSeconds"), 10, 32)
	incomingRequest.SendPeriodSeconds = uint32(sendPeriodSeconds)

	sendErrorAtNthResponse, _ := strconv.ParseUint(c.Query("sendErrorAtNthResponse"), 10, 32)
	incomingRequest.SendErrorAtNthResponse = uint32(sendErrorAtNthResponse)

	closeAtNthResponse, _ := strconv.ParseUint(c.Query("closeAtNthResponse"), 10, 32)
	incomingRequest.CloseAtNthResponse = uint32(closeAtNthResponse)

	return incomingRequest, nil
}

func parseStringStreamRequest(c *gin.Context) (*tgsbpb.ServerStreamStringRequest, error) {
	incomingRequest := &tgsbpb.ServerStreamStringRequest{}
	incomingRequest.Value = c.Query("value")

	sendPeriodSeconds, _ := strconv.ParseUint(c.Query("sendPeriodSeconds"), 10, 32)
	incomingRequest.SendPeriodSeconds = uint32(sendPeriodSeconds)

	sendErrorAtNthResponse, _ := strconv.ParseUint(c.Query("sendErrorAtNthResponse"), 10, 32)
	incomingRequest.SendErrorAtNthResponse = uint32(sendErrorAtNthResponse)

	closeAtNthResponse, _ := strconv.ParseUint(c.Query("closeAtNthResponse"), 10, 32)
	incomingRequest.CloseAtNthResponse = uint32(closeAtNthResponse)

	return incomingRequest, nil
}
