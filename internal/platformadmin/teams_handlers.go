package platformadmin

import (
	"encoding/json"
	"net/http"

	"ad-event-processor/internal/teamscope"
	"ad-event-processor/pkg/httpresponse"
)

type CreateTeamRequest struct {
	Name string `json:"name"`
}

type TeamsListResponse struct {
	Items []teamscope.TeamDTO `json:"items"`
}

func (h *TeamHTTPHandlers) registerTeamsRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.Pool == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/team/teams", limit(perm([]string{"team:read"}, h.listTeams)))
	mux.HandleFunc("POST /api/v1/team/teams", limit(perm([]string{"team:write"}, h.createTeam)))
}

func (h *TeamHTTPHandlers) listTeams(w http.ResponseWriter, r *http.Request) {
	customerID, err := h.ResolveCustomerID(r, nil)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	items, err := teamscope.ListTeams(r.Context(), h.Pool, customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, TeamsListResponse{Items: items})
}

func (h *TeamHTTPHandlers) createTeam(w http.ResponseWriter, r *http.Request) {
	customerID, err := h.ResolveCustomerID(r, nil)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	var req CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	team, err := teamscope.CreateTeam(r.Context(), h.Pool, customerID, req.Name)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, team)
}
