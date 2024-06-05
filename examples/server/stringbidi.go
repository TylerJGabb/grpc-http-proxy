package main

import (
	"context"
	"fmt"
	"io"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *exampleServer) BidirectionalStreamString(
	stream tgsbpb.TylerSandboxService_BidirectionalStreamStringServer,
) error {
	// when we return out of this method, the context of the stream will be canceled
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	in := make(chan *tgsbpb.BidirectionalStreamStringRequest)
	errs := make(chan error, 1)
	go func() {
		for {
			// this is a blocking call, will unblock when stream context is canceled
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					cancel()
				} else {
					fmt.Printf("receive error: %s\n", err.Error())
					errs <- err
				}
				return
			}
			select {
			case in <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()
	for {
		select {
		case err := <-errs:
			return err
		case <-ctx.Done():
			return nil
		case req := <-in:
			if req.Close {
				cancel()
				return status.Error(codes.Canceled, "stream closed intentionally")
			}
			if req.RespondWithError {
				return status.Error(codes.Internal, "error returned intentionally")
			}
			send := &tgsbpb.BidirectionalStreamStringResponse{Value: req.Value}
			// this is a blocking call, will unblock when stream context is canceled
			if err := stream.Send(send); err != nil {
				fmt.Printf("send error: %s\n", err.Error())
				return err
			}
		}
	}
}
