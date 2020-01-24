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
		strings.Split(*streamBrokers, ","), au10.NewFactory())
	defer service.Close()

	users, err := au10.NewUsers(service.GetFactory())
	if err != nil {
		service.Log().Fatal(`Failed to start users service: "%s".`, err)
		return
	}
	defer users.Close()

	posts := au10.NewPosts(service)
	defer posts.Close()

	var publisher au10.Publisher
	publisher, err = au10.NewPublisher(service)
	if err != nil || publisher == nil {
		service.Log().Fatal(`Failed to create publisher: "%s".`, err)
		return
	}
	defer publisher.Close()

	var defaultUser au10.User
	defaultUser, err = service.GetFactory().NewUser(
		0, "", au10.NewMembership("", ""), []au10.Rights{})
	if err != nil {
		service.Log().Fatal(`Failed to created default user: "%s".`, err)
		return
	}

	logReader := au10.NewLogReader(service)
	defer logReader.Close()

	accesspoint := lib.NewAu10Server(props, service.Log(), logReader, posts,
		publisher, users, defaultUser, lib.NewClient, lib.NewGrpc())
	server := grpc.NewServer()
	defer server.Stop()
	proto.RegisterAu10Server(server, accesspoint)

	service.Log().Debug(`Starting server on port "%d"...`, *port)
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		service.Log().Fatal(`Failed to open server port: "%s".`, err)
		return
	}
	service.Log().Info(`Started server on port "%d".`, *port)

	err = server.Serve(listen)
	if err != nil {
		service.Log().Fatal(`Failed to start server: "%s".`, err)
		return
	}

	service.Log().Info("Server stopped. Closing...")
}
