package server

import (
	"fmt"
	"net"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type ExampleServer struct {
	tgsbpb.UnimplementedTylerSandboxServiceServer
}

func (s *ExampleServer) Start(port int) string {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	tgsbpb.RegisterTylerSandboxServiceServer(grpcServer, s)
	go func() {
		fmt.Printf("Server listening on %s\n", listener.Addr().String())
		if err := grpcServer.Serve(listener); err != nil {
			panic(err)
		}
	}()
	return listener.Addr().String()
}
