package server

import (
	"fmt"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ExampleServer) ClientStreamInt(
	stream tgsbpb.TylerSandboxService_ClientStreamIntServer,
) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		if msg.Close {
			if err := stream.SendAndClose(&tgsbpb.ClientStreamIntResponse{Value: msg.Value}); err != nil {
				fmt.Println(err)
				return err
			}
			return nil
		}
		if msg.RespondWithError {
			return status.Error(codes.Internal, "intentionally returned error")
		}
	}
}

func (s *ExampleServer) ClientStreamString(
	stream tgsbpb.TylerSandboxService_ClientStreamStringServer,
) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		if msg.Close {
			if err := stream.SendAndClose(&tgsbpb.ClientStreamStringResponse{Value: msg.Value}); err != nil {
				fmt.Println(err)
				return err
			}
			return nil
		}
		if msg.RespondWithError {
			return status.Error(codes.Internal, "intentionally returned error")
		}
	}
}
