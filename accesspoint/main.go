package main

import (
	"flag"
	"fmt"
	"net"
	"strings"

	lib "bitbucket.org/au10/service/accesspoint/lib"
	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

var (
	nodeName      = flag.String("name", "", "node instance name")
	port          = flag.Uint("port", 2917, "access point port")
	streamBrokers = flag.String("stream_brokers",
		"localhost:9092", "stream brokers, as a comma-separated list")
)

const nodeType = "accesspoint"

var props = &proto.Props{
	MaxChunkSize: 10}

func main() {
	flag.Parse()

	service := au10.DialOrPanic(nodeType, *nodeName,
		strings.Split(*streamBrokers, ","), au10.CreateFactory())
	defer service.Close()

	if err := service.InitUsers(); err != nil {
		service.Log().Fatal(`Failed to start users service: "%s".`, err)
		return
	}
	if err := service.InitPosts(); err != nil {
		service.Log().Fatal(`Failed to start users service: "%s".`, err)
		return
	}
	if err := service.GetPosts().InitSubscriptionService(); err != nil {
		service.Log().Fatal(`Failed to start posts subscription service: "%s".`,
			err)
		return
	}
	if err := service.Log().InitSubscriptionService(); err != nil {
		service.Log().Fatal(`Failed to start log subscription service: "%s".`, err)
		return
	}

	defaultUser, err := service.GetFactory().CreateUser(
		"", au10.CreateMembership("", ""), []au10.Rights{})
	if err != nil {
		service.Log().Fatal(`Failed to created default user: "%s".`, err)
		return
	}

	accesspoint := lib.CreateAu10Server(
		props, defaultUser, lib.CreateClient, lib.CreateGrpc(), service)
	server := grpc.NewServer()
	defer server.Stop()

	proto.RegisterAu10Server(server, accesspoint)

	service.Log().Info(`Starting server on port "%d"...`, *port)
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		service.Log().Fatal(`Failed to open server port: "%s".`, err)
		return
	}
	err = server.Serve(listen)
	if err != nil {
		service.Log().Fatal(`Failed to start server: "%s".`, err)
		return
	}
	service.Log().Info("Server stopped.")
}
