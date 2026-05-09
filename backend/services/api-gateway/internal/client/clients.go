package client

import (
	"errors"
	"fmt"

	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Clients holds all gRPC client connections and their underlying connections.
type Clients struct {
	URL   pb.URLServiceClient
	Auth  pb.AuthServiceClient
	Stats pb.StatsServiceClient

	conns []*grpc.ClientConn
}

// Close gracefully closes all underlying gRPC connections.
func (c *Clients) Close() {
	for _, conn := range c.conns {
		conn.Close()
	}
}

// NewClients establishes gRPC connections to all services and returns a Clients instance.
func NewClients(urlAddr, authAddr, statsAddr string) (*Clients, error) {
	urlConn, err := newConn(urlAddr)
	if err != nil {
		return nil, fmt.Errorf("url service: %w", err)
	}

	authConn, err := newConn(authAddr)
	if err != nil {
		urlConn.Close()
		return nil, fmt.Errorf("auth service: %w", err)
	}

	statsConn, err := newConn(statsAddr)
	if err != nil {
		urlConn.Close()
		authConn.Close()
		return nil, fmt.Errorf("stats service: %w", err)
	}

	return &Clients{
		URL:   pb.NewURLServiceClient(urlConn),
		Auth:  pb.NewAuthServiceClient(authConn),
		Stats: pb.NewStatsServiceClient(statsConn),
		conns: []*grpc.ClientConn{urlConn, authConn, statsConn},
	}, nil
}

func newConn(addr string) (*grpc.ClientConn, error) {
	if addr == "" {
		return nil, errors.New("address must not be empty")
	}
	return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}
