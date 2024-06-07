package testutils

import (
	"context"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type TestMessage struct {
	Value *wrapperspb.StringValue
	Err   error
}

// GrpcClientStreamMock is a mock implementation of the GrpcClientStreamFacade interface
// it provides a way to simulate a server stream from the grpc server via the
// SimulateServerSideMessage method
type GrpcClientStreamMock struct {
	serverStream chan TestMessage
}

func (s GrpcClientStreamMock) RecvMsg(m any) error {
	testMessage := <-s.serverStream
	if testMessage.Err != nil {
		return testMessage.Err
	}
	payload, _ := protojson.Marshal(testMessage.Value)
	return protojson.Unmarshal(payload, m.(*wrapperspb.StringValue))
}

func (s GrpcClientStreamMock) SimulateServerSideMessage(tm TestMessage) {
	s.serverStream <- tm
}

type OpenStreamFuncMock struct {
	GrpcClientStreamMock
	errorWhenStreamOpened error
	ReceivedRequest       *wrapperspb.StringValue
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
	m.ReceivedRequest = req
	if m.errorWhenStreamOpened != nil {
		return nil, m.errorWhenStreamOpened
	}
	return &m.GrpcClientStreamMock, nil
}

type WebsocketEvent struct {
	Payload []byte
	Err     error
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
				Payload: msg,
				Err:     err,
			}
			ch <- event
			if err != nil {
				return
			}
		}
	}()
	return
}
