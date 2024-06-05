package main

import (
	"flag"

	server "github.com/TylerJGabb/grpc-http-proxy/examples/server/impl"
)

func main() {
	port := flag.Int("port", 50054, "the port number")
	flag.Parse()
	s := server.ExampleServer{}
	s.Start(*port)
	select {}
}
