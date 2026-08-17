package controlplane

import (
	"context"
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"

	"github.com/google/uuid"
)

type publisherAdminAdapter struct {
	svc *Service
}

func (a publisherAdminAdapter) ResolvePublisherBind(ctx context.Context, userID uuid.UUID) (adminapi.PublisherBind, error) {
	bind, err := a.svc.ResolvePublisherBind(ctx, userID)
	if err != nil {
		return adminapi.PublisherBind{}, err
	}
	return adminapi.PublisherBind{
		SellerID:           bind.SellerID,
		PublisherAccountID: bind.PublisherAccountID,
		CustomerID:         bind.CustomerID,
	}, nil
}

func (a publisherAdminAdapter) GetPublisherDashboard(ctx context.Context, bind adminapi.PublisherBind, from, to time.Time) (adminapi.PublisherDashboardDTO, error) {
	return a.svc.GetPublisherDashboard(ctx, PublisherBind{
		SellerID:           bind.SellerID,
		PublisherAccountID: bind.PublisherAccountID,
		CustomerID:         bind.CustomerID,
	}, from, to)
}

func (a publisherAdminAdapter) ListPublisherStatements(ctx context.Context, bind adminapi.PublisherBind, from, to time.Time, limit, offset int32) ([]adminapi.PublisherStatementDTO, int64, error) {
	return a.svc.ListPublisherStatements(ctx, PublisherBind{
		SellerID:           bind.SellerID,
		PublisherAccountID: bind.PublisherAccountID,
		CustomerID:         bind.CustomerID,
	}, from, to, limit, offset)
}
