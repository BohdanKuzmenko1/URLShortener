package server

import (
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/service"
)

// AuthServer implements the gRPC AuthServiceServer interface.
type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	authService service.Auth
}

// NewAuthServer returns a new AuthServer instance with the provided Auth service.
func NewAuthServer(authService service.Auth) *AuthServer {
	return &AuthServer{
		authService: authService,
	}
}
