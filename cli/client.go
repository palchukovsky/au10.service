package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	accesspoint "bitbucket.org/au10/service/accesspoint/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// DealOrDie creates service connection or stops process at error.
func DealOrDie(address string) *Client {
	var err error
	var result Client
	log.Printf(`Connecting to "%s"...`, address)
	if err != nil {
		log.Fatalf("failed to load credentials: %v", err)
	}
	config := &tls.Config{InsecureSkipVerify: true}
	creds := grpc.WithTransportCredentials(credentials.NewTLS(config))
	result.conn, err = grpc.Dial(address, creds, grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to connect to access point: %v", err)
	}
	log.Printf("Connected.")
	result.ctx, result.cancel = context.WithCancel(context.Background())
	result.client = accesspoint.NewAu10Client(result.conn)
	return &result
}

// Client is a service client implementation.
type Client struct {
	conn   *grpc.ClientConn
	ctx    context.Context
	cancel context.CancelFunc
	client accesspoint.Au10Client
	header metadata.MD
}

// Close closes client.
func (client *Client) Close() {
	client.conn.Close()
}

// Auth authes connection.
func (client *Client) Auth(login string) {
	response, err := client.client.Auth(client.getCtx(),
		&accesspoint.AuthRequest{Login: login}, grpc.Header(&client.header))
	if err != nil {
		log.Fatalf("Failed to auth: %v", err)
	}
	if response.IsSuccess {
		log.Printf(`Successfully authed with login "%s".`, login)
	} else {
		log.Printf(`Failed to auth with login "%s".`, login)
	}
	log.Printf(`Available methods: %v.`, response.AllowedMethods)
}

// ReadLog reads service log.
func (client *Client) ReadLog() {
	log.Println("Reading log...")
	stream, err := client.client.ReadLog(client.getCtx(),
		&accesspoint.LogReadRequest{}, grpc.Header(&client.header))
	if err != nil {
		log.Fatalf(`Failed to read log: "%s".`, err)
	}
	for {
		record, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf(`Failed to read log record: "%s".`, err)
		}
		if record.Severity != "debug" && record.Severity != "info" {
			record.Severity = strings.ToUpper(record.Severity)
		}
		fmt.Printf("%d. %-5s %s [%s/%s]: %s\n", record.SeqNum, record.Severity,
			time.Unix(0, record.Time).Format(time.StampMicro), record.NodeType,
			record.NodeName, record.Text)
	}
	log.Println("Log ended.")
}

// ReadPosts reads posts.
func (client *Client) ReadPosts() {
	log.Println("Reading posts...")
	stream, err := client.client.ReadPosts(client.getCtx(),
		&accesspoint.PostsReadRequest{}, grpc.Header(&client.header))
	if err != nil {
		log.Fatalf(`Failed to read posts: "%s".`, err)
	}
	for {
		post, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf(`Failed to read post: "%s".`, err)
		}
		fmt.Printf("Post %d (%s):\n", post.Id,
			time.Unix(0, post.Time).Format(time.StampMicro))
		if len(post.Messages) == 0 {
			fmt.Println("\tno messages\t")
		} else {
			for _, m := range post.Messages {
				fmt.Printf("\t%d. (%d bytes)\n", m.Id, m.Size)
			}
		}
	}
	log.Println("Posts ended.")
}

func (client *Client) getCtx() context.Context {
	if token, ok := client.header["auth"]; ok && len(token) == 1 {
		return metadata.AppendToOutgoingContext(client.ctx, "auth", token[0])
	}
	return client.ctx
}
