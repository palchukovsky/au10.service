// Package main implements a client for Greeter service.
package main

import (
	"flag"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

var (
	address = flag.String("address", "localhost:443",
		"Au10 service access point address")
	command = flag.String("cmd", "", "command: log, posts, post_vocal")
	login   = flag.String("login", "", "login")
)

func main() {
	flag.Parse()

	grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name))

	client := DialOrDie(*address)
	defer client.Close()

	if *login != "" {
		client.Auth(*login)
	}

	switch *command {
	case "log":
		client.ReadLog()
	case "posts":
		client.ReadPosts()
	case "post_vocal":
		client.PostVocal()
	default:
		log.Fatalf(`Unknown command: "%s".`, *command)
	}
}
