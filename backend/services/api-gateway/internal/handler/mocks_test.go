package handler_test

import (
	"context"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockAuthServiceClient struct {
	mock.Mock
}

func (m *MockAuthServiceClient) Register(ctx context.Context, in *pb.RegisterRequest, opts ...grpc.CallOption) (*pb.RegisterResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.RegisterResponse), args.Error(1)
}

func (m *MockAuthServiceClient) Login(ctx context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.LoginResponse), args.Error(1)
}

func (m *MockAuthServiceClient) RefreshToken(ctx context.Context, in *pb.RefreshRequest, opts ...grpc.CallOption) (*pb.RefreshResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.RefreshResponse), args.Error(1)
}

func (m *MockAuthServiceClient) Logout(ctx context.Context, in *pb.LogoutRequest, opts ...grpc.CallOption) (*pb.LogoutResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.LogoutResponse), args.Error(1)
}

type MockStatsServiceClient struct {
	mock.Mock
}

func (m *MockStatsServiceClient) GetURLStats(ctx context.Context, in *pb.GetURLStatsRequest, opts ...grpc.CallOption) (*pb.GetURLStatsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetURLStatsResponse), args.Error(1)
}

type MockUrlServiceClient struct {
	mock.Mock
}

func (m *MockUrlServiceClient) GetURL(ctx context.Context, in *pb.GetURLRequest, opts ...grpc.CallOption) (*pb.GetURLResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetURLResponse), args.Error(1)
}

func (m *MockUrlServiceClient) CreateShortURL(ctx context.Context, in *pb.CreateShortUrlRequest, opts ...grpc.CallOption) (*pb.CreateShortUrlResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.CreateShortUrlResponse), args.Error(1)
}

func (m *MockUrlServiceClient) ResolveSlug(ctx context.Context, in *pb.ResolveSlugRequest, opts ...grpc.CallOption) (*pb.ResolveSlugResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.ResolveSlugResponse), args.Error(1)
}
