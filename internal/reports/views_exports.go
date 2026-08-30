package reports

import (
	"ad-event-processor/internal/reports/views"

	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	ViewsHTTPHandlers = views.ViewsHTTPHandlers
	SavedViewDTO      = views.SavedViewDTO
	ViewsStore        = views.ViewsStore
	CreateViewRequest = views.CreateViewRequest
	UpdateViewRequest = views.UpdateViewRequest
)

func NewViewsStore(pool *pgxpool.Pool) *ViewsStore {
	return views.NewViewsStore(pool)
}

var ErrViewNotFound = views.ErrViewNotFound
