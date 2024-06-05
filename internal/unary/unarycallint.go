package unary

import (
	"fmt"
	"io"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
)

func ProxyIntRequest(c *gin.Context) {
	client := c.MustGet("client").(tgsbpb.TylerSandboxServiceClient)
	var requestBody tgsbpb.UnaryCallIntRequest
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		fmt.Printf("error reading request body: %v", err)
		return
	}

	if err := protojson.Unmarshal(body, &requestBody); err != nil {
		fmt.Printf("error unmarshalling request body: %v", err)
		c.String(400, "error unmarshalling request body: %v", err)
	}

	response, err := client.UnaryCallInt(c, &requestBody)
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
