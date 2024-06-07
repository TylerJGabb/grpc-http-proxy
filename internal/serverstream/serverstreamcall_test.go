package serverstream_test

import (
	"context"
	"errors"
	"fmt"
	"io"
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

type TestMessage struct {
	value *wrapperspb.StringValue
	err   error
}

// GrpcClientStreamMock is a mock implementation of the GrpcClientStreamFacade interface
// it provides a way to simulate a server stream from the grpc server via the
// SimulateServerSideMessage method
type GrpcClientStreamMock struct {
	serverStream chan TestMessage
}

func NewGrpcClientStreamMock() GrpcClientStreamMock {
	return GrpcClientStreamMock{
		// should we buffer this
		serverStream: make(chan TestMessage),
	}
}

func (s GrpcClientStreamMock) RecvMsg(m any) error {
	testMessage := <-s.serverStream
	if testMessage.err != nil {
		return testMessage.err
	}
	payload, _ := protojson.Marshal(testMessage.value)
	return protojson.Unmarshal(payload, m.(*wrapperspb.StringValue))
}

func (s GrpcClientStreamMock) SimulateServerSideMessage(tm TestMessage) {
	s.serverStream <- tm
}

type OpenStreamFuncMock struct {
	GrpcClientStreamMock
	errorWhenStreamOpened error
	_receivedRequest      *wrapperspb.StringValue
}

type OpenStreamFuncMockOptFunc func(*OpenStreamFuncMock)

func WithErrorWhenStreamOpened(err error) OpenStreamFuncMockOptFunc {
	return func(m *OpenStreamFuncMock) {
		m.errorWhenStreamOpened = err
	}
}

func NewOpenStreamFuncMock(opts ...OpenStreamFuncMockOptFunc) OpenStreamFuncMock {
	m := OpenStreamFuncMock{
		GrpcClientStreamMock: GrpcClientStreamMock{
			serverStream: make(chan TestMessage),
		},
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func (m *OpenStreamFuncMock) Func(
	_ context.Context,
	req *wrapperspb.StringValue,
	_ ...grpc.CallOption,
) (*GrpcClientStreamMock, error) {
	m._receivedRequest = req
	if m.errorWhenStreamOpened != nil {
		return nil, m.errorWhenStreamOpened
	}
	return &m.GrpcClientStreamMock, nil
}

func (m *OpenStreamFuncMock) GetReceivedRequest() *wrapperspb.StringValue {
	return m._receivedRequest
}

type WebsocketEvent struct {
	payload []byte
	err     error
}

func OpenWebsocket(handler func(c *gin.Context)) (
	events <-chan WebsocketEvent,
	closeFunc func(),
	err error,
) {
	app := gin.New()
	app.GET("/test", handler)
	s := httptest.NewServer(app)
	closeFunc = s.Close

	url := "ws" + strings.TrimPrefix(s.URL, "http") + "/test"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return
	}

	ch := make(chan WebsocketEvent)
	events = ch
	closeFunc = func() {
		conn.Close()
		s.Close()
	}

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			event := WebsocketEvent{
				payload: msg,
				err:     err,
			}
			ch <- event
			if err != nil {
				return
			}
		}
	}()
	return
}

func Test_ServerStreamProxy(t *testing.T) {

	t.Run("proxies all server sent messages over the socket connection", func(t *testing.T) {

		mockedOpenStreamFunc := NewOpenStreamFuncMock()

		expectedValueFromRequest := "parsed-value-from-request"
		parseRequest := func(c *gin.Context) (*wrapperspb.StringValue, error) {
			return &wrapperspb.StringValue{
				Value: expectedValueFromRequest,
			}, nil
		}

		handler := func(c *gin.Context) {
			serverstream.ServerStreamProxy(
				c,
				mockedOpenStreamFunc.Func,
				parseRequest,
				&wrapperspb.StringValue{},
			)
		}

		events, closeFunc, err := OpenWebsocket(handler)
		defer closeFunc()
		if err != nil {
			t.Fatalf("failed to open websocket: %v\n", err)
		}

		receivedRequest := mockedOpenStreamFunc.GetReceivedRequest()
		if receivedRequest.Value != expectedValueFromRequest {
			t.Fatalf(
				"expected openStreamFunc to receive request with value %s, got %s\n",
				expectedValueFromRequest,
				receivedRequest.Value,
			)
		}

		for i := 0; i < 3; i++ {
			expectedValue := &wrapperspb.StringValue{Value: fmt.Sprintf("value-%d", i)}
			// this will block until the proxy loop in the proxy function is ready to receive
			mockedOpenStreamFunc.SimulateServerSideMessage(TestMessage{value: expectedValue})
			var websocketEvent WebsocketEvent
			select {
			case websocketEvent = <-events:
			case <-time.After(1 * time.Second):
				t.Fatalf("timed out waiting for message from websocket\n")
			}

			if websocketEvent.err != nil {
				t.Fatalf("did not expect error from websocket: %v\n", websocketEvent.err)
			}

			expectedPayload, _ := protojson.Marshal(expectedValue)
			if string(websocketEvent.payload) != string(expectedPayload) {
				t.Fatalf(
					"expected message to be %s, got %s\n",
					string(expectedPayload),
					string(websocketEvent.payload),
				)
			}
		}
	})

	t.Run("propagates server error to client with CloseInternalServerErr close frame", func(t *testing.T) {

		mockedOpenStreamFunc := NewOpenStreamFuncMock()

		expectedValueFromRequest := "parsed-value-from-request"
		parseRequest := func(c *gin.Context) (*wrapperspb.StringValue, error) {
			return &wrapperspb.StringValue{
				Value: expectedValueFromRequest,
			}, nil
		}

		handler := func(c *gin.Context) {
			serverstream.ServerStreamProxy(
				c,
				mockedOpenStreamFunc.Func,
				parseRequest,
				&wrapperspb.StringValue{},
			)
		}

		events, closeFunc, err := OpenWebsocket(handler)
		defer closeFunc()
		if err != nil {
			t.Fatalf("failed to open websocket: %v\n", err)
		}

		receivedRequest := mockedOpenStreamFunc.GetReceivedRequest()
		if receivedRequest.Value != expectedValueFromRequest {
			t.Fatalf(
				"expected openStreamFunc to receive request with value %s, got %s\n",
				expectedValueFromRequest,
				receivedRequest.Value,
			)
		}

		expectedErrorFromServer := fmt.Errorf("server error")
		mockedOpenStreamFunc.SimulateServerSideMessage(TestMessage{err: expectedErrorFromServer})
		var websocketEvent WebsocketEvent
		select {
		case websocketEvent = <-events:
		case <-time.After(1 * time.Second):
			t.Fatalf("timed out waiting for event from websocket\n")
		}

		if websocketEvent.err == nil {
			t.Fatalf("expected error from websocket, got nil\n")
		}

		if !websocket.IsCloseError(websocketEvent.err, websocket.CloseInternalServerErr) {
			t.Fatalf("expected close frame to be CloseInternalServerErr, got %v\n", websocketEvent.err)
		}

		if !strings.Contains(websocketEvent.err.Error(), expectedErrorFromServer.Error()) {
			t.Fatalf(
				"expected error message to contain '%s', got '%s'\n",
				expectedErrorFromServer.Error(),
				websocketEvent.err.Error(),
			)
		}
	})

	t.Run("if client closes connection, handler dies", func(t *testing.T) {
		mockedOpenStreamFunc := NewOpenStreamFuncMock()

		expectedValueFromRequest := "parsed-value-from-request"
		parseRequest := func(c *gin.Context) (*wrapperspb.StringValue, error) {
			return &wrapperspb.StringValue{
				Value: expectedValueFromRequest,
			}, nil
		}

		dead := false
		handler := func(c *gin.Context) {
			serverstream.ServerStreamProxy(
				c,
				mockedOpenStreamFunc.Func,
				parseRequest,
				&wrapperspb.StringValue{},
			)
			dead = true
		}

		events, closeFunc, err := OpenWebsocket(handler)
		defer closeFunc()
		if err != nil {
			t.Fatalf("failed to open websocket: %v\n", err)
		}

		receivedRequest := mockedOpenStreamFunc.GetReceivedRequest()
		if receivedRequest.Value != expectedValueFromRequest {
			t.Fatalf(
				"expected openStreamFunc to receive request with value %s, got %s\n",
				expectedValueFromRequest,
				receivedRequest.Value,
			)
		}

		closeFunc()
		var websocketEvent WebsocketEvent
		select {
		case websocketEvent = <-events:
		case <-time.After(1 * time.Second):
			t.Fatalf("timed out waiting for event from websocket\n")
		}

		if websocketEvent.err == nil {
			t.Fatalf("expected error from websocket, got nil\n")
		}

		time.Sleep(1 * time.Second)
		if !dead {
			t.Fatalf("expected handler to be dead after client closes connection\n")
		}
	})

	t.Run("if the server ends the stream, normal close frame is sent", func(t *testing.T) {
		mockedOpenStreamFunc := NewOpenStreamFuncMock()

		expectedValueFromRequest := "parsed-value-from-request"
		parseRequest := func(c *gin.Context) (*wrapperspb.StringValue, error) {
			return &wrapperspb.StringValue{
				Value: expectedValueFromRequest,
			}, nil
		}

		handler := func(c *gin.Context) {
			serverstream.ServerStreamProxy(
				c,
				mockedOpenStreamFunc.Func,
				parseRequest,
				&wrapperspb.StringValue{},
			)
		}

		events, closeFunc, err := OpenWebsocket(handler)
		defer closeFunc()
		if err != nil {
			t.Fatalf("failed to open websocket: %v\n", err)
		}

		receivedRequest := mockedOpenStreamFunc.GetReceivedRequest()
		if receivedRequest.Value != expectedValueFromRequest {
			t.Fatalf(
				"expected openStreamFunc to receive request with value %s, got %s\n",
				expectedValueFromRequest,
				receivedRequest.Value,
			)
		}

		mockedOpenStreamFunc.SimulateServerSideMessage(TestMessage{err: io.EOF})
		var websocketEvent WebsocketEvent
		select {
		case websocketEvent = <-events:
		case <-time.After(1 * time.Second):
			t.Fatalf("timed out waiting for event from websocket\n")
		}

		if websocketEvent.err == nil {
			t.Fatalf("expected error from websocket, got nil\n")
		}

		if !websocket.IsCloseError(websocketEvent.err, websocket.CloseNormalClosure) {
			t.Fatalf("expected close frame to be CloseNormalClosure, got %v\n", websocketEvent.err)
		}
	})

	t.Run("if open stream fails, ws handshake fails with 500", func(t *testing.T) {
		mockedOpenStreamFunc := NewOpenStreamFuncMock(
			WithErrorWhenStreamOpened(errors.New("open stream failed")),
		)

		parseRequest := func(c *gin.Context) (*wrapperspb.StringValue, error) {
			return &wrapperspb.StringValue{}, nil
		}

		handler := func(c *gin.Context) {
			serverstream.ServerStreamProxy(
				c,
				mockedOpenStreamFunc.Func,
				parseRequest,
				&wrapperspb.StringValue{},
			)
		}

		_, closeFunc, err := OpenWebsocket(handler)
		defer closeFunc()

		if !errors.Is(err, websocket.ErrBadHandshake) {
			t.Fatalf("expected websocket handshake to fail with ErrBadHandshake, got %v\n", err)
		}
	})

	t.Run("if request parser fails, ws handshake fails with 400", func(t *testing.T) {
		mockedOpenStreamFunc := NewOpenStreamFuncMock()

		parseRequest := func(c *gin.Context) (*wrapperspb.StringValue, error) {
			return nil, errors.New("parse request failed")
		}

		handler := func(c *gin.Context) {
			serverstream.ServerStreamProxy(
				c,
				mockedOpenStreamFunc.Func,
				parseRequest,
				&wrapperspb.StringValue{},
			)
		}

		_, closeFunc, err := OpenWebsocket(handler)
		defer closeFunc()

		if !errors.Is(err, websocket.ErrBadHandshake) {
			t.Fatalf("expected websocket handshake to fail with ErrBadHandshake, got %v\n", err)
		}
	})
}
