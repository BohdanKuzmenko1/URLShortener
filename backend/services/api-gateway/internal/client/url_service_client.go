package client

import (
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewURLServiceClient establishes a gRPC connection to the URL Service at the given address
// and returns a client instance ready to make RPC calls.
func NewURLServiceClient(addr string) (pb.URLServiceClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return pb.NewURLServiceClient(conn), nil
}
