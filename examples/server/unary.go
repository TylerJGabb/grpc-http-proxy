package main

import (
	"context"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
)

func (s *exampleServer) UnaryCallInt(
	ctx context.Context,
	req *tgsbpb.UnaryCallIntRequest,
) (*tgsbpb.UnaryCallIntResponse, error) {
	return &tgsbpb.UnaryCallIntResponse{Value: req.Value}, nil
}

func (s *exampleServer) UnaryCallString(
	ctx context.Context,
	req *tgsbpb.UnaryCallStringRequest,
) (*tgsbpb.UnaryCallStringResponse, error) {
	return &tgsbpb.UnaryCallStringResponse{Value: req.Value}, nil
}
