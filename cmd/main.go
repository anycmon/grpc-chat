package main

import (
	"flag"
	"fmt"
	pb "grpc-chat/chat"
	"grpc-chat/server"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	port = flag.Int("port", 10000, "The server port")
)

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterChatServer(grpcServer, server.New())
	grpcServer.Serve(lis)
}
