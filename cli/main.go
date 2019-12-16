// Package main implements a client for Greeter service.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

var (
	address = flag.String("address", "localhost:2917",
		"Au10 service access point address")
	command = flag.String("cmd", "log", "command: log")
	login   = flag.String("login", "", "login")
)

func main() {
	flag.Parse()

	grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name))

	client := DealOrDie(*address)
	defer client.Close()

	if *login != "" {
		client.Auth(*login)
	}

	if *command == "log" {
		client.ReadLog()
	} else {
		log.Fatalf(`Unknown command: "%s".`, *command)
	}

	log.Printf("To exit press CTRL+C")
	signalsChan := make(chan os.Signal)
	defer close(signalsChan)
	signal.Notify(signalsChan, os.Interrupt, syscall.SIGTERM)
	<-signalsChan
}
