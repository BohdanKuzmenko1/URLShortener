package server

import (
	"context"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
)

// Register handles a gRPC registration request.
// Creates a new user account and returns a token pair on success,
// or an error if the email already exists or the request failed.
func (s *AuthServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	tokenPair, err := s.authService.Register(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	pbTokenPair := &pb.TokenPair{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int32(tokenPair.ExpiresIn),
		TokenType:    tokenPair.TokenType,
	}

	return &pb.RegisterResponse{Token: pbTokenPair}, nil
}

// Login handles a gRPC login request.
// Verifies user credentials and returns a token pair on success,
// or an error if the credentials are invalid or the request failed.
func (s *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	tokenPair, err := s.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	pbTokenPair := &pb.TokenPair{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int32(tokenPair.ExpiresIn),
		TokenType:    tokenPair.TokenType,
	}

	return &pb.LoginResponse{Token: pbTokenPair}, nil
}

// RefreshToken handles a gRPC token refresh request.
// Issues a new token pair using the provided refresh token,
// or returns an error if the token is invalid or expired.
func (s *AuthServer) RefreshToken(ctx context.Context, req *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	tokenPair, err := s.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	pbTokenPair := &pb.TokenPair{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int32(tokenPair.ExpiresIn),
		TokenType:    tokenPair.TokenType,
	}

	return &pb.RefreshResponse{Token: pbTokenPair}, nil
}

// Logout handles a gRPC logout request.
// Invalidates the provided refresh token,
// or returns an error if the token is invalid or the request failed.
func (s *AuthServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	err := s.authService.Logout(ctx, req.RefreshToken)
	if err != nil {
		return &pb.LogoutResponse{Success: false}, err
	}

	return &pb.LogoutResponse{Success: true}, nil
}
