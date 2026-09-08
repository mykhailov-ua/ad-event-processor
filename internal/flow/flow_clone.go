package flow

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type CloneFlowRequest struct {
	Name string `json:"name,omitempty"`
}

func (st *Store) CloneFlow(ctx context.Context, flowID uuid.UUID, req CloneFlowRequest) (DTO, error) {
	if st.poolOrNil() == nil {
		return DTO{}, fmt.Errorf("service unavailable")
	}
	if flowID == uuid.Nil {
		return DTO{}, fmt.Errorf("flow id is required")
	}
	src, err := st.GetFlow(ctx, flowID)
	if err != nil {
		return DTO{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(src.Name) + " (copy)"
	}
	paths, err := ParsePaths(src.Paths)
	if err != nil {
		return DTO{}, fmt.Errorf("invalid stored paths")
	}
	return st.CreateFlow(ctx, CreateFlowRequest{Name: name, Paths: paths})
}
