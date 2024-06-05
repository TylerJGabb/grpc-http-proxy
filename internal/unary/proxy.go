package unary

import (
	"context"
	"fmt"
	"io"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type UnaryProxy[T proto.Message] interface {
	UnaryCall(context.Context, tgsbpb.TylerSandboxServiceClient, T) (proto.Message, error)
	NewGrpcMessage() T
	Path() string
}

func ApplyUnaryProxy[T proto.Message](r *gin.Engine, unaryProxy UnaryProxy[T]) {
	r.POST(unaryProxy.Path(), ProxyRequestHandler(unaryProxy))
}

func ProxyRequestHandler[T proto.Message](unaryProxy UnaryProxy[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		client := c.MustGet("client").(tgsbpb.TylerSandboxServiceClient)
		requestBody := unaryProxy.NewGrpcMessage()
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			fmt.Printf("error reading request body: %v", err)
			return
		}

		if err := protojson.Unmarshal(body, requestBody); err != nil {
			fmt.Printf("error unmarshalling request body: %v", err)
			c.String(400, "error unmarshalling request body: %v", err)
		}

		response, err := unaryProxy.UnaryCall(c, client, requestBody)
		if err != nil {
			fmt.Printf("error proxying request: %v", err)
			c.String(500, "error calling proxying: %v", err)
		}

		responseBody, err := protojson.Marshal(response)
		if err != nil {
			fmt.Printf("error marshalling response: %v", err)
			c.String(500, "error marshalling response: %v", err)
		}

		c.Data(200, "application/json", responseBody)
	}
}
