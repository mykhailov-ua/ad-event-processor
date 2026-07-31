package identity

import (
	"espx/internal/config"
	"espx/internal/identity/pb"
)

type Handler struct {
	pb.UnimplementedAuthServiceServer
	service *Service
	cfg     *config.Config
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}
