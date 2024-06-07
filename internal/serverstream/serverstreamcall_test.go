package serverstream_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TylerJGabb/grpc-http-proxy/internal/serverstream"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type mockGrpcClientStream struct {
	trigger   chan bool
	toReceive string
}

func (s mockGrpcClientStream) RecvMsg(m any) error {
	value := &wrapperspb.StringValue{Value: s.toReceive}
	<-s.trigger
	payload, err := protojson.Marshal(value)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(payload, m.(*wrapperspb.StringValue))
}

func (s mockGrpcClientStream) TriggerReceive() {
	s.trigger <- true
}

type mockOpenStreamFunc struct {
	toReceive            string
	receivedRequest      *wrapperspb.StringValue
	mockGrpcClientStream mockGrpcClientStream
}

func (m *mockOpenStreamFunc) openStreamFunc(
	_ context.Context,
	req *wrapperspb.StringValue,
	_ ...grpc.CallOption,
) (mockGrpcClientStream, error) {
	m.receivedRequest = req
	m.mockGrpcClientStream = mockGrpcClientStream{
		trigger:   make(chan bool),
		toReceive: m.toReceive,
	}
	return m.mockGrpcClientStream, nil
}

func Test_ServerStreamProxy(t *testing.T) {
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

		received := make(chan []byte)
		errs := make(chan error, 1)
		go func() {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errs <- err
			}
			received <- msg
		}()
		mockedOpenStreamFunc.mockGrpcClientStream.TriggerReceive()
		var msg []byte
		select {
		case err := <-errs:
			t.Fatalf("error reading from websocket: %v\n", err)
		case msg = <-received:
		case <-time.After(1 * time.Second):
			t.Fatalf("timed out waiting for message from websocket\n")
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

	t.Run("propagates server error to client with CloseInternalServerErr close frame", func(t *testing.T) {

	})

}
