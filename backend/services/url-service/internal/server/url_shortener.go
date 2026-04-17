package server

import (
	"context"
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal"
	"github.com/BohdanKuzmenko1/URLShortener/shared"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"os"
)

func (s *URLServer) GetURL(ctx context.Context, req *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "no authorization header")
	}

	claims, err := shared.ValidateToken(authHeaders[0], os.Getenv("JWT_SIGNING_KEY"))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	userId := claims.UserId
	if userId == 0 {
		return nil, status.Error(codes.Unauthenticated, "invalid user id")
	}

	url, err := s.urlShortenerService.GetURL(ctx, userId, int(req.UrlId))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetURLResponse{ShortUrl: &pb.ShortURL{
		UrlId:     url.UrlId,
		TargetUrl: url.TargetUrl,
		Slug:      url.Slug,
		CreatedAt: url.CreatedAt,
	}}, nil
}

func (s *URLServer) CreateShortURL(ctx context.Context, req *pb.CreateShortUrlRequest) (*pb.CreateShortUrlResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "no authorization header")
	}

	claims, err := shared.ValidateToken(authHeaders[0], os.Getenv("JWT_SIGNING_KEY"))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	userId := claims.UserId
	if userId == 0 {
		return nil, status.Error(codes.Unauthenticated, "invalid user id")
	}

	if req.TargetUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "empty target url")
	}

	logrus.Printf("Creating shorturl for user %d", userId)
	shortURL, err := s.urlShortenerService.GenerateShortURL(ctx, userId, req.TargetUrl, req.Slug)
	if err != nil {
		return nil, err
	}
	return &pb.CreateShortUrlResponse{ShortUrl: shortURL}, nil
}

func (s *URLServer) ResolveSlug(ctx context.Context, req *pb.ResolveSlugRequest) (*pb.ResolveSlugResponse, error) {
	redirect := internal.Redirect{
		Slug:      req.Redirect.Slug,
		ClientIP:  req.Redirect.ClientIp,
		Language:  req.Redirect.Language,
		UserAgent: req.Redirect.UserAgent,
		Country:   req.Redirect.Country,
		Referer:   req.Redirect.Referer,
	}

	targetURL, err := s.urlShortenerService.ResolveSlug(ctx, redirect)

	if err != nil {
		return nil, err
	}

	return &pb.ResolveSlugResponse{TargetUrl: targetURL}, nil
}
