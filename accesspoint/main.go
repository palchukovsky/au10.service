package main

import (
	"flag"
	"fmt"
	"net"
	"strings"

	lib "bitbucket.org/au10/service/accesspoint/lib"
	proto "bitbucket.org/au10/service/accesspoint/proto"
	"bitbucket.org/au10/service/au10"
	"bitbucket.org/au10/service/postdb"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

var (
	nodeName      = flag.String("name", "", "node instance name")
	port          = flag.Uint("port", 2917, "access point port")
	streamBrokers = flag.String("stream_brokers",
		"localhost:9092", "stream brokers, as a comma-separated list")
	dbHost     = flag.String("db_host", "localhost", "database host")
	dbName     = flag.String("db_name", "postdb", "database name")
	dbLogin    = flag.String("db_login", "postdb", "database user login name")
	dbPassword = flag.String("db_password", "", "database user login password")
)

var props = &proto.Props{
	MaxChunkSize: 10}

func main() {
	flag.Parse()

	service := au10.DialOrPanic("accesspoint", *nodeName,
		strings.Split(*streamBrokers, ","), au10.NewFactory())
	defer service.Close()
	defer service.Log().Info("Stopped.")

	users, err := au10.NewUsers(service)
	if err != nil {
		service.Log().Fatal(`Failed to start users service: "%s".`, err)
		return
	}
	defer users.Close()

	service.Log().Debug(`Connecting to the database "%s@%s/%s"...`,
		*dbLogin, *dbHost, *dbName)
	db, err := postdb.Dial(*dbHost, *dbName, *dbLogin, *dbPassword,
		func(err error) { service.Log().Fatal(`DB error: "%s".`, err) })
	if err != nil {
		service.Log().Fatal(`Failed to connect to the database "%s@%s/%s": "%s".`,
			*dbLogin, *dbHost, *dbName, err)
		return
	}
	defer db.Close()

	posts := au10.NewPosts(service, db)
	if err != nil {
		service.Log().Fatal(`Failed to start posts service: "%s".`,
			*dbLogin, *dbHost, *dbName, err)
		return
	}
	defer posts.Close()

	publisher, err := au10.NewPublisher(service)
	if err != nil || publisher == nil {
		service.Log().Fatal(`Failed to create publisher: "%s".`, err)
		return
	}
	defer publisher.Close()

	var defaultUser au10.User
	defaultUser, err = service.GetFactory().NewUser(
		0, "", au10.NewMembership("", ""), []au10.Rights{}, service)
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
	service.Log().Debug(`Started server on port "%d".`, *port)

	err = server.Serve(listen)
	if err != nil {
		service.Log().Fatal(`Failed to start server: "%s".`, err)
		return
	}

	service.Log().Debug("Server stopped. Closing...")
}
