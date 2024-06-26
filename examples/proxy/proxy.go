package main

import (
	"flag"
	"google.golang.org/grpc/credentials/insecure"

	server "github.com/TylerJGabb/grpc-http-proxy/examples/server/impl"
	"github.com/TylerJGabb/grpc-http-proxy/pkg/proxy"
)

func main() {
	exampleServer := server.ExampleServer{}
	addr := exampleServer.Start(9091)

	port := flag.Int("port", 8080, "the port number")
	flag.Parse()
	hps := proxy.NewHttpProxyServer(
		addr,
		proxy.WithPort(*port),
		proxy.WithGrpcTransportCredentials(insecure.NewCredentials()),
	)
	hps.RunBlocking()
}
