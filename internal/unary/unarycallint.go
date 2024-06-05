package unary

import (
	"context"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"google.golang.org/protobuf/proto"
)

type UnaryCallIntProxy struct{}

func (u UnaryCallIntProxy) Path() string {
	return "/unarycallint"
}

func (u UnaryCallIntProxy) UnaryCall(
	ctx context.Context,
	client tgsbpb.TylerSandboxServiceClient,
	request *tgsbpb.UnaryCallIntRequest,
) (proto.Message, error) {
	return client.UnaryCallInt(ctx, request)
}

func (u UnaryCallIntProxy) NewGrpcMessage() *tgsbpb.UnaryCallIntRequest {
	return &tgsbpb.UnaryCallIntRequest{}
}
