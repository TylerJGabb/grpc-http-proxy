package unary

import (
	"context"
	"fmt"
	"io"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ProxyMessage interface {
	tgsbpb.UnaryCallIntRequest | tgsbpb.UnaryCallStringRequest
}

func ProxyRequest[T, U proto.Message](
	c *gin.Context,
	emptyRequest T,
	callFunc func(context.Context, T, ...grpc.CallOption) (U, error),
) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		fmt.Printf("error reading request body: %v", err)
		return
	}

	if err := protojson.Unmarshal(body, emptyRequest); err != nil {
		fmt.Printf("error unmarshalling request body: %v", err)
		c.String(400, "error unmarshalling request body: %v", err)
	}

	response, err := callFunc(c, emptyRequest)
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
