package accesspoint

import (
	"context"

	grpc_lib "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Grpc wraps grpc-package.
type Grpc interface {
	SendHeader(context.Context, metadata.MD) error
}

// CreateGrpc creates new instance of Grpc.
func CreateGrpc() Grpc {
	return &grpc{}
}

type grpc struct{}

func (*grpc) SendHeader(ctx context.Context, md metadata.MD) error {
	return grpc_lib.SendHeader(ctx, md)
}
