package server_test

import (
	"context"
	"errors"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/auth-service/internal/server"
	"github.com/BohdanKuzmenko1/URLShortener/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (*shared.TokenPair, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*shared.TokenPair), args.Error(1)
}

func (m *MockAuthService) Register(ctx context.Context, email, password string) (*shared.TokenPair, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*shared.TokenPair), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*shared.TokenPair, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*shared.TokenPair), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

func TestAuthServer_Register(t *testing.T) {
	tests := []struct {
		name           string
		request        *pb.RegisterRequest
		mockResponse   *shared.TokenPair
		mockError      error
		serverResponse *pb.RegisterResponse
		serverError    bool
	}{
		{
			name: "Success",
			request: &pb.RegisterRequest{
				Email:    "test@example.com",
				Password: "secret123",
			},
			mockResponse: &shared.TokenPair{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    28800,
				TokenType:    "Bearer",
			},
			serverResponse: &pb.RegisterResponse{
				Token: &pb.TokenPair{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresIn:    28800,
					TokenType:    "Bearer",
				},
			},
		},
		{
			name: "Service returns error",
			request: &pb.RegisterRequest{
				Email:    "existing@example.com",
				Password: "secret123",
			},
			mockError:   errors.New("something went wrong"),
			serverError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockAuthService)
			mockService.
				On("Register", mock.Anything, tt.request.Email, tt.request.Password).
				Return(tt.mockResponse, tt.mockError)

			srv := server.NewAuthServer(mockService)

			// Act
			resp, err := srv.Register(context.Background(), tt.request)

			// Assert
			if tt.serverError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.serverResponse.Token.AccessToken, resp.Token.AccessToken)
				assert.Equal(t, tt.serverResponse.Token.RefreshToken, resp.Token.RefreshToken)
				assert.Equal(t, tt.serverResponse.Token.ExpiresIn, resp.Token.ExpiresIn)
				assert.Equal(t, tt.serverResponse.Token.TokenType, resp.Token.TokenType)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthServer_Login(t *testing.T) {
	tests := []struct {
		name           string
		request        *pb.LoginRequest
		mockResponse   *shared.TokenPair
		mockError      error
		serverResponse *pb.LoginResponse
		serverError    bool
	}{
		{
			name: "Success",
			request: &pb.LoginRequest{
				Email:    "test@example.com",
				Password: "secret123",
			},
			mockResponse: &shared.TokenPair{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    28800,
				TokenType:    "Bearer",
			},
			serverResponse: &pb.LoginResponse{
				Token: &pb.TokenPair{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresIn:    28800,
					TokenType:    "Bearer",
				},
			},
		},
		{
			name: "Service returns error",
			request: &pb.LoginRequest{
				Email:    "existing@example.com",
				Password: "secret123",
			},
			mockError:   errors.New("something went wrong"),
			serverError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockAuthService)
			mockService.
				On("Login", mock.Anything, tt.request.Email, tt.request.Password).
				Return(tt.mockResponse, tt.mockError)

			srv := server.NewAuthServer(mockService)

			// Act
			resp, err := srv.Login(context.Background(), tt.request)

			// Assert
			if tt.serverError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.serverResponse.Token.AccessToken, resp.Token.AccessToken)
				assert.Equal(t, tt.serverResponse.Token.RefreshToken, resp.Token.RefreshToken)
				assert.Equal(t, tt.serverResponse.Token.ExpiresIn, resp.Token.ExpiresIn)
				assert.Equal(t, tt.serverResponse.Token.TokenType, resp.Token.TokenType)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthServer_RefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		request        *pb.RefreshRequest
		mockResponse   *shared.TokenPair
		mockError      error
		serverResponse *pb.RefreshResponse
		serverError    bool
	}{
		{
			name: "Success",
			request: &pb.RefreshRequest{
				RefreshToken: "refresh-token",
			},
			mockResponse: &shared.TokenPair{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    28800,
				TokenType:    "Bearer",
			},
			serverResponse: &pb.RefreshResponse{
				Token: &pb.TokenPair{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresIn:    28800,
					TokenType:    "Bearer",
				},
			},
		},
		{
			name: "Service returns error",
			request: &pb.RefreshRequest{
				RefreshToken: "refresh-token",
			},
			mockError:   errors.New("something went wrong"),
			serverError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockAuthService)
			mockService.
				On("RefreshToken", mock.Anything, tt.request.RefreshToken).
				Return(tt.mockResponse, tt.mockError)

			srv := server.NewAuthServer(mockService)

			// Act
			resp, err := srv.RefreshToken(context.Background(), tt.request)

			// Assert
			if tt.serverError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.serverResponse.Token.AccessToken, resp.Token.AccessToken)
				assert.Equal(t, tt.serverResponse.Token.RefreshToken, resp.Token.RefreshToken)
				assert.Equal(t, tt.serverResponse.Token.ExpiresIn, resp.Token.ExpiresIn)
				assert.Equal(t, tt.serverResponse.Token.TokenType, resp.Token.TokenType)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAuthServer_Logout(t *testing.T) {
	tests := []struct {
		name           string
		request        *pb.LogoutRequest
		mockError      error
		serverResponse *pb.LogoutResponse
		serverError    bool
	}{
		{
			name: "Success",
			request: &pb.LogoutRequest{
				RefreshToken: "refresh-token",
			},
			serverResponse: &pb.LogoutResponse{
				Success: true,
			},
		},
		{
			name: "Service returns error",
			request: &pb.LogoutRequest{
				RefreshToken: "refresh-token",
			},
			mockError:   errors.New("something went wrong"),
			serverError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockService := new(MockAuthService)
			mockService.
				On("Logout", mock.Anything, tt.request.RefreshToken).
				Return(tt.mockError)

			srv := server.NewAuthServer(mockService)

			// Act
			resp, err := srv.Logout(context.Background(), tt.request)

			// Assert
			if tt.serverError {
				assert.Error(t, err)
				assert.Equal(t, resp.Success, false)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.serverResponse.Success, resp.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}
