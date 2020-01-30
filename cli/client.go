package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	proto "bitbucket.org/au10/service/accesspoint/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// DialOrDie creates service connection or stops process at error.
func DialOrDie(address string) *Client {
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
	result.client = proto.NewAu10Client(result.conn)
	return &result
}

// Client is a service client implementation.
type Client struct {
	conn   *grpc.ClientConn
	ctx    context.Context
	cancel context.CancelFunc
	client proto.Au10Client
	header metadata.MD
}

// Close closes client.
func (client *Client) Close() {
	client.conn.Close()
}

// Auth authes connection.
func (client *Client) Auth(login string) {
	response, err := client.client.Auth(client.getCtx(),
		&proto.AuthRequest{Login: login}, grpc.Header(&client.header))
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
		&proto.LogReadRequest{}, grpc.Header(&client.header))
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
		&proto.PostsReadRequest{}, grpc.Header(&client.header))
	if err != nil {
		log.Fatalf(`Failed to read posts: "%s".`, err)
	}
	for {
		message, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf(`Failed to read post: "%s".`, err)
		}
		var typeName string
		var post *proto.Post
		switch typedPost := message.Post.(type) {
		case *proto.PostUpdate_Vocal:
			typeName = "Vocal"
			post = typedPost.Vocal.Post
		}
		if post == nil {
			fmt.Printf(`Post has unknown type "%v".`+"\n", message)
			continue
		}
		fmt.Printf("%s %s (%s):\n", typeName, post.Id,
			time.Unix(0, post.Time).Format(time.StampMicro))
		if len(post.Messages) == 0 {
			fmt.Println("\tno messages")
		} else {
			for _, m := range post.Messages {
				fmt.Printf("\t%s (%d bytes, kind %d)\n", m.Id, m.Size, m.Kind)
			}
		}
	}
	log.Println("Posts ended.")
}

// PostVocal posts new vocal.
func (client *Client) PostVocal() {
	log.Println("Positing vocal...")

	messageText := `Test message from cli: "ABC", "1234567890" and "!@#$%^&*()".`
	vocal, err := client.client.AddVocal(
		client.getCtx(),
		&proto.VocalAddRequest{
			Post: &proto.PostAddRequest{
				Location: &proto.GeoPoint{
					Latitude:  34.692946,
					Longitude: 33.031114},
				Messages: []*proto.PostAddRequest_MessageDeclaration{
					&proto.PostAddRequest_MessageDeclaration{
						Kind: proto.Message_TEXT,
						Size: uint32(len(messageText))}}}})
	if err != nil {
		log.Fatalf(`Failed to post vocal: "%s".`, err)
	}

	log.Printf(`Added post with ID "%s" and time %s. Writing images...`+"\n",
		vocal.Post.Id, time.Unix(0, vocal.Post.Time))

	chuckSize := 3
	for i := 0; i < len(messageText); i += chuckSize {
		_, err = client.client.WriteMessageChunk(
			client.getCtx(),
			&proto.MessageChunkWriteRequest{
				PostID:    vocal.Post.Id,
				MessageID: vocal.Post.Messages[0].Id,
				Chunk:     []byte(messageText[i:chuckSize])})
		if err != nil {
			log.Fatalf(`Failed to post vocal: "%s".`, err)
		}
	}

	log.Printf(`Vocal posted with ID "%s" and time %s.`+"\n",
		vocal.Post.Id, time.Unix(0, vocal.Post.Time))

}

func (client *Client) getCtx() context.Context {
	if token, ok := client.header["auth"]; ok && len(token) == 1 {
		return metadata.AppendToOutgoingContext(client.ctx, "auth", token[0])
	}
	return client.ctx
}
