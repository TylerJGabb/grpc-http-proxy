package server

import (
	"fmt"
	"math"
	"time"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ExampleServer) ServerStreamInt(
	req *tgsbpb.ServerStreamIntRequest,
	stream tgsbpb.TylerSandboxService_ServerStreamIntServer,
) error {
	req.SendPeriodSeconds = uint32(math.Max(float64(req.SendPeriodSeconds), 1))
	if req.SendErrorAtNthResponse == 0 {
		req.SendErrorAtNthResponse = math.MaxUint32
	}
	if req.CloseAtNthResponse == 0 {
		req.CloseAtNthResponse = math.MaxUint32
	}
	var responseCount uint32 = 1
	for {
		time.Sleep(time.Second * time.Duration(req.SendPeriodSeconds))
		if responseCount >= req.CloseAtNthResponse {
			return nil
		}
		if responseCount >= req.SendErrorAtNthResponse {
			return status.Error(codes.Internal, "intentionally returned error")
		}
		if err := stream.Send(&tgsbpb.ServerStreamIntResponse{Value: req.Value}); err != nil {
			fmt.Println(err.Error())
			return status.Error(codes.Internal, err.Error())
		}
		responseCount++
	}
}

func (s *ExampleServer) ServerStreamString(
	req *tgsbpb.ServerStreamStringRequest,
	stream tgsbpb.TylerSandboxService_ServerStreamStringServer,
) error {
	req.SendPeriodSeconds = uint32(math.Max(float64(req.SendPeriodSeconds), 1))
	if req.SendErrorAtNthResponse == 0 {
		req.SendErrorAtNthResponse = math.MaxUint32
	}
	if req.CloseAtNthResponse == 0 {
		req.CloseAtNthResponse = math.MaxUint32
	}
	var responseCount uint32 = 1
	for {
		time.Sleep(time.Second * time.Duration(req.SendPeriodSeconds))
		if responseCount >= req.CloseAtNthResponse {
			return nil
		}
		if responseCount >= req.SendErrorAtNthResponse {
			return status.Error(codes.Internal, "intentionally returned error")
		}
		value := fmt.Sprintf("response-%d value=%s", responseCount, req.Value)
		if err := stream.Send(&tgsbpb.ServerStreamStringResponse{Value: value}); err != nil {
			fmt.Println(err.Error())
			return status.Error(codes.Internal, err.Error())
		}
		responseCount++
	}
}
