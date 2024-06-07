package serverstream_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TylerJGabb/grpc-http-proxy/internal/serverstream"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type mockGrpcClientStream struct {
	toReceive string
}

func (s mockGrpcClientStream) RecvMsg(m any) error {
	value := &wrapperspb.StringValue{Value: s.toReceive}
	payload, err := protojson.Marshal(value)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(payload, m.(*wrapperspb.StringValue))
}

type mockOpenStreamFunc struct {
	toReceive       string
	receivedRequest *wrapperspb.StringValue
}

func (m *mockOpenStreamFunc) openStreamFunc(
	_ context.Context,
	req *wrapperspb.StringValue,
	_ ...grpc.CallOption,
) (mockGrpcClientStream, error) {
	m.receivedRequest = req
	return mockGrpcClientStream{
		toReceive: m.toReceive,
	}, nil
}

func Test_ServerStreamProxy(t *testing.T) {
	// the proxy will proxy server stream messages over the socket connection
	t.Run("proxies server sent messages over the socket connection", func(t *testing.T) {
		expectedValueFromServer := "received-from-server"
		expectedValueFromRequest := "parsed-value-from-request"

		mockedOpenStreamFunc := &mockOpenStreamFunc{
			toReceive: expectedValueFromServer,
		}

		parseRequest := func(c *gin.Context) (*wrapperspb.StringValue, error) {
			return &wrapperspb.StringValue{
				Value: expectedValueFromRequest,
			}, nil
		}

		app := gin.New()
		app.GET("/test", func(c *gin.Context) {
			serverstream.ServerStreamProxy(
				c,
				mockedOpenStreamFunc.openStreamFunc,
				parseRequest,
				&wrapperspb.StringValue{},
			)
		})
		s := httptest.NewServer(app)
		defer s.Close()

		url := "ws" + strings.TrimPrefix(s.URL, "http") + "/test"
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v\n", err)
		}
		defer conn.Close()

		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("failed to read message from websocket: %v\n", err)
		}

		if mockedOpenStreamFunc.receivedRequest.Value != expectedValueFromRequest {
			t.Fatalf(
				"expected openStreamFunc to receive request with value %s, got %s\n",
				expectedValueFromRequest,
				mockedOpenStreamFunc.receivedRequest.Value,
			)
		}

		expectedValue := &wrapperspb.StringValue{Value: expectedValueFromServer}
		expectedPayload, _ := protojson.Marshal(expectedValue)
		if string(msg) != string(expectedPayload) {
			t.Fatalf(
				"expected message to be %s, got %s\n",
				string(expectedPayload),
				string(msg),
			)
		}
	})
}
