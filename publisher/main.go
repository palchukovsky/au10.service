package main

import (
	"flag"
	"strings"

	"bitbucket.org/au10/service/au10"
)

var (
	nodeName      = flag.String("name", "", "node instance name")
	streamBrokers = flag.String("stream_brokers",
		"localhost:9092", "stream brokers, as a comma-separated list")
)

const nodeType = "pub"

func main() {
	flag.Parse()

	service := au10.DialOrPanic(nodeType, *nodeName,
		strings.Split(*streamBrokers, ","), au10.NewFactory())
	defer service.Close()

}
