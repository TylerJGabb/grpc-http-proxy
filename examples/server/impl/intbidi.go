package server

import (
	"github.com/TylerJGabb/grpc-http-proxy/internal/logging"
	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ExampleServer) BidirectionalStreamInt(
	stream tgsbpb.TylerSandboxService_BidirectionalStreamIntServer,
) error {
	logger := logging.NewLogger(logging.WithOperationId("server.BidirectionalStreamInt"))
	logger.Info("stream starting")
	in := make(chan *tgsbpb.BidirectionalStreamIntRequest)
	go func() {
		defer logger.Info("server receive goroutine ended")
		for {
			// this is a blocking call, will unblock when stream context is canceled
			msg, err := stream.Recv()
			if err != nil {
				logger.Info("error while receiving from stream", "err", err)
				return
			}
			select {
			case in <- msg:
			case <-stream.Context().Done():
				return
			}
		}
	}()
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case req := <-in:
			if req.Close {
				logger.Info("server is closing the stream intentionally")
				return status.Error(codes.Canceled, "stream closed intentionally")
			}
			if req.RespondWithError {
				logger.Info("server is intentionally returning error")
				return status.Error(codes.Internal, "error returned intentionally")
			}
			send := &tgsbpb.BidirectionalStreamIntResponse{Value: req.Value}
			logger.Info("sending response", "value", send.Value)
			if err := stream.Send(send); err != nil {
				logger.Info("unexpected error while sending on stream", "err", err)
			}
		}
	}
}
