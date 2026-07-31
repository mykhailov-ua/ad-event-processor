package notifier

import (
	"espx/internal/notifier/pb"
)

type Handler struct {
	pb.UnimplementedNotifierServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}
