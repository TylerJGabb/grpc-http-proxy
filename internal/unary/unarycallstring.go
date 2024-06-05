package unary

import (
	"context"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"google.golang.org/protobuf/proto"
)

type UnaryCallStringProxy struct{}

func (u UnaryCallStringProxy) Path() string {
	return "/unarycallstring"
}

func (u UnaryCallStringProxy) UnaryCall(
	ctx context.Context,
	client tgsbpb.TylerSandboxServiceClient,
	request *tgsbpb.UnaryCallStringRequest,
) (proto.Message, error) {
	return client.UnaryCallString(ctx, request)
}

func (u UnaryCallStringProxy) NewGrpcMessage() *tgsbpb.UnaryCallStringRequest {
	return &tgsbpb.UnaryCallStringRequest{}
}
