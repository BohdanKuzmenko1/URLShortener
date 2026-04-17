package server

import (
	pb "github.com/BohdanKuzmenko1/URLShortener/proto"
	"github.com/BohdanKuzmenko1/URLShortener/services/url-service/internal/service"
)

type URLServer struct {
	pb.UnimplementedURLServiceServer
	urlShortenerService service.URLShortenerService
}

func NewURLServer(urlShortenerService service.URLShortenerService) *URLServer {
	return &URLServer{
		urlShortenerService: urlShortenerService,
	}
}
