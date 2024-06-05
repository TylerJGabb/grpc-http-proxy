package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type exampleServer struct {
	tgsbpb.UnimplementedTylerSandboxServiceServer
}

func (s *exampleServer) Start(port int) string {
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

func main() {
	port := flag.Int("port", 50054, "the port number")
	flag.Parse()
	s := exampleServer{}
	s.Start(*port)
	select {}
}
