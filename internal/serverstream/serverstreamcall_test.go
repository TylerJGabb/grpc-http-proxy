package serverstream_test

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/TylerJGabb/grpc-http-proxy/internal/serverstream/testutils"

	"github.com/TylerJGabb/grpc-http-proxy/internal/serverstream"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func Test_ServerStreamProxy(t *testing.T) {

	t.Run("proxies all server sent messages over the socket connection", func(t *testing.T) {

		mockedOpenStreamFunc := testutils.NewOpenStreamFuncMock()

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

		events, closeFunc, err := testutils.OpenWebsocket(handler)
		defer closeFunc()
		if err != nil {
			t.Fatalf("failed to open websocket: %v\n", err)
		}

		receivedRequest := mockedOpenStreamFunc.ReceivedRequest
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
			mockedOpenStreamFunc.SimulateServerSideMessage(testutils.TestMessage{Value: expectedValue})
			var websocketEvent testutils.WebsocketEvent
			select {
			case websocketEvent = <-events:
			case <-time.After(1 * time.Second):
				t.Fatalf("timed out waiting for message from websocket\n")
			}

			if websocketEvent.Err != nil {
				t.Fatalf("did not expect error from websocket: %v\n", websocketEvent.Err)
			}

			expectedPayload, _ := protojson.Marshal(expectedValue)
			if string(websocketEvent.Payload) != string(expectedPayload) {
				t.Fatalf(
					"expected message to be %s, got %s\n",
					string(expectedPayload),
					string(websocketEvent.Payload),
				)
			}
		}
	})

	t.Run("propagates server error to client with CloseInternalServerErr close frame", func(t *testing.T) {

		mockedOpenStreamFunc := testutils.NewOpenStreamFuncMock()

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

		events, closeFunc, err := testutils.OpenWebsocket(handler)
		defer closeFunc()
		if err != nil {
			t.Fatalf("failed to open websocket: %v\n", err)
		}

		receivedRequest := mockedOpenStreamFunc.ReceivedRequest
		if receivedRequest.Value != expectedValueFromRequest {
			t.Fatalf(
				"expected openStreamFunc to receive request with value %s, got %s\n",
				expectedValueFromRequest,
				receivedRequest.Value,
			)
		}

		expectedErrorFromServer := fmt.Errorf("server error")
		mockedOpenStreamFunc.SimulateServerSideMessage(testutils.TestMessage{Err: expectedErrorFromServer})
		var websocketEvent testutils.WebsocketEvent
		select {
		case websocketEvent = <-events:
		case <-time.After(1 * time.Second):
			t.Fatalf("timed out waiting for event from websocket\n")
		}

		if websocketEvent.Err == nil {
			t.Fatalf("expected error from websocket, got nil\n")
		}

		if !websocket.IsCloseError(websocketEvent.Err, websocket.CloseInternalServerErr) {
			t.Fatalf("expected close frame to be CloseInternalServerErr, got %v\n", websocketEvent.Err)
		}

		if !strings.Contains(websocketEvent.Err.Error(), expectedErrorFromServer.Error()) {
			t.Fatalf(
				"expected error message to contain '%s', got '%s'\n",
				expectedErrorFromServer.Error(),
				websocketEvent.Err.Error(),
			)
		}
	})

	t.Run("if client closes connection, handler dies", func(t *testing.T) {
		mockedOpenStreamFunc := testutils.NewOpenStreamFuncMock()

		expectedValueFromRequest := "parsed-value-from-request"
		parseRequest := func(c *gin.Context) (*wrapperspb.StringValue, error) {
			return &wrapperspb.StringValue{
				Value: expectedValueFromRequest,
			}, nil
		}

		dead := make(chan bool, 1)
		handler := func(c *gin.Context) {
			serverstream.ServerStreamProxy(
				c,
				mockedOpenStreamFunc.Func,
				parseRequest,
				&wrapperspb.StringValue{},
			)
			dead <- true
		}

		events, closeFunc, err := testutils.OpenWebsocket(handler)
		defer closeFunc()
		if err != nil {
			t.Fatalf("failed to open websocket: %v\n", err)
		}

		receivedRequest := mockedOpenStreamFunc.ReceivedRequest
		if receivedRequest.Value != expectedValueFromRequest {
			t.Fatalf(
				"expected openStreamFunc to receive request with value %s, got %s\n",
				expectedValueFromRequest,
				receivedRequest.Value,
			)
		}

		closeFunc()
		var websocketEvent testutils.WebsocketEvent
		select {
		case websocketEvent = <-events:
		case <-time.After(1 * time.Second):
			t.Fatalf("timed out waiting for event from websocket\n")
		}

		if websocketEvent.Err == nil {
			t.Fatalf("expected error from websocket, got nil\n")
		}

		select {
		case <-dead:
		case <-time.After(2 * time.Second):
			t.Fatalf("expected handler to be dead after client closes connection\n")
		}
	})

	t.Run("if the server ends the stream, normal close frame is sent", func(t *testing.T) {
		mockedOpenStreamFunc := testutils.NewOpenStreamFuncMock()

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

		events, closeFunc, err := testutils.OpenWebsocket(handler)
		defer closeFunc()
		if err != nil {
			t.Fatalf("failed to open websocket: %v\n", err)
		}

		receivedRequest := mockedOpenStreamFunc.ReceivedRequest
		if receivedRequest.Value != expectedValueFromRequest {
			t.Fatalf(
				"expected openStreamFunc to receive request with value %s, got %s\n",
				expectedValueFromRequest,
				receivedRequest.Value,
			)
		}

		mockedOpenStreamFunc.SimulateServerSideMessage(testutils.TestMessage{Err: io.EOF})
		var websocketEvent testutils.WebsocketEvent
		select {
		case websocketEvent = <-events:
		case <-time.After(1 * time.Second):
			t.Fatalf("timed out waiting for event from websocket\n")
		}

		if websocketEvent.Err == nil {
			t.Fatalf("expected error from websocket, got nil\n")
		}

		if !websocket.IsCloseError(websocketEvent.Err, websocket.CloseNormalClosure) {
			t.Fatalf("expected close frame to be CloseNormalClosure, got %v\n", websocketEvent.Err)
		}
	})

	t.Run("if open stream fails, ws handshake fails with 500", func(t *testing.T) {
		mockedOpenStreamFunc := testutils.NewOpenStreamFuncMock(
			testutils.WithErrorWhenStreamOpened(errors.New("open stream failed")),
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

		_, closeFunc, err := testutils.OpenWebsocket(handler)
		defer closeFunc()

		if !errors.Is(err, websocket.ErrBadHandshake) {
			t.Fatalf("expected websocket handshake to fail with ErrBadHandshake, got %v\n", err)
		}
	})

	t.Run("if request parser fails, ws handshake fails with 400", func(t *testing.T) {
		mockedOpenStreamFunc := testutils.NewOpenStreamFuncMock()

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

		_, closeFunc, err := testutils.OpenWebsocket(handler)
		defer closeFunc()

		if !errors.Is(err, websocket.ErrBadHandshake) {
			t.Fatalf("expected websocket handshake to fail with ErrBadHandshake, got %v\n", err)
		}
	})
}
