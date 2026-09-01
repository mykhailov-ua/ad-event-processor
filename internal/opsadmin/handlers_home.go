package opsadmin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"ad-event-processor/internal/doctor"
	"ad-event-processor/pkg/httpresponse"
)

type DoctorSnapshotBuilder func(ctx context.Context) (doctor.DoctorResponseDTO, error)

type opsHomeSnapshotDTO struct {
	Doctor           doctor.DoctorResponseDTO `json:"doctor"`
	StackHealth      StackHealthSnapshot      `json:"stackHealth"`
	DashboardSummary DashboardSummaryDTO      `json:"dashboardSummary"`
}

func (h *HTTPHandlers) registerHomeRoutes(mux *http.ServeMux) {
	if h == nil || h.OpsReader == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/ops/home", limit(perm("shards:read", h.getOpsHome)))
}

func (h *HTTPHandlers) GetOpsHome(w http.ResponseWriter, r *http.Request) {
	h.getOpsHome(w, r)
}

func (h *HTTPHandlers) getOpsHome(w http.ResponseWriter, r *http.Request) {
	if h.DoctorSnapshot == nil {
		h.writeServiceError(w, fmt.Errorf("doctor snapshot not configured"))
		return
	}

	ctx := r.Context()
	var snap opsHomeSnapshotDTO
	var doctorErr, healthErr, dashErr error
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		snap.Doctor, doctorErr = h.DoctorSnapshot(ctx)
	}()
	go func() {
		defer wg.Done()
		snap.StackHealth, healthErr = h.OpsReader.GetStackHealthSnapshot(ctx)
	}()
	go func() {
		defer wg.Done()
		snap.DashboardSummary, dashErr = h.OpsReader.GetDashboardSummary(ctx)
	}()
	wg.Wait()

	if err := errors.Join(doctorErr, healthErr, dashErr); err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpresponse.JSON(w, http.StatusOK, snap)
}
