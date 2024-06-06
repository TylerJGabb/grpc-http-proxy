package unary_test

import (
	"bytes"
	"context"
	"errors"
	"github.com/TylerJGabb/grpc-http-proxy/internal/unary"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockCallFunc struct {
	receivedValue string
	returnError   bool
	valueToReturn string
}

func (m *mockCallFunc) callFunc(
	_ context.Context,
	v *wrapperspb.StringValue,
	_ ...grpc.CallOption,
) (*wrapperspb.StringValue, error) {
	m.receivedValue = v.Value
	if m.returnError {
		return nil, errors.New("intentionally returned an error")
	}
	return &wrapperspb.StringValue{Value: m.valueToReturn}, nil
}

func Test_ProxyRequest(t *testing.T) {

	t.Run("happy path", func(t *testing.T) {
		app := gin.New()
		expectedValue := "1234"
		mockedCallFunc := &mockCallFunc{valueToReturn: expectedValue}
		app.POST("/test", func(c *gin.Context) {
			unary.ProxyRequest(c, &wrapperspb.StringValue{}, mockedCallFunc.callFunc)
		})
		requestBodyValue := "abcd"
		payload, err := protojson.Marshal(&wrapperspb.StringValue{
			Value: requestBodyValue,
		})
		if err != nil {
			t.Fatalf("protojson failed to marshall: %v\n", err)
		}

		req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(payload))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected status code %d, got %d\n", http.StatusOK, w.Code)
		}

		received := &wrapperspb.StringValue{}
		err = protojson.Unmarshal(w.Body.Bytes(), received)
		if err != nil {
			t.Fatalf("protojson failed to unmarshall received message: %v\n", err)
		}
		if received.Value != expectedValue {
			t.Fatalf("Expected response body value %s, got %s\n", expectedValue, received.Value)
		}
		if mockedCallFunc.receivedValue != requestBodyValue {
			t.Fatalf("Expected callFunc to receive value %s, got %s\n", requestBodyValue, mockedCallFunc.receivedValue)
		}
	})

	t.Run("failure to unmarshall request body returns 400", func(t *testing.T) {
		app := gin.New()
		mockedCallFunc := &mockCallFunc{}
		app.POST("/test", func(c *gin.Context) {
			unary.ProxyRequest(c, &wrapperspb.StringValue{}, mockedCallFunc.callFunc)
		})
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString("garbage"))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status code %d, got %d\n", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("if call func returns error return 500", func(t *testing.T) {
		app := gin.New()
		mockedCallFunc := &mockCallFunc{returnError: true}
		app.POST("/test", func(c *gin.Context) {
			unary.ProxyRequest(c, &wrapperspb.StringValue{}, mockedCallFunc.callFunc)
		})
		requestValue := "5432"
		payload, err := protojson.Marshal(&wrapperspb.StringValue{
			Value: requestValue,
		})
		if err != nil {
			t.Fatalf("protojson failed to marshall: %v\n", err)
		}
		req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(payload))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("Expected status code %d, got %d\n", http.StatusInternalServerError, w.Code)
		}
		if mockedCallFunc.receivedValue != requestValue {
			t.Fatalf("Expected callFunc to receive value %s, got %s\n", requestValue, mockedCallFunc.receivedValue)
		}
	})

}
