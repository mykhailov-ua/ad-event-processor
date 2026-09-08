package flow

import "context"

func (st *Store) ValidateFlowPaths(ctx context.Context, paths []PathDTO) (ValidateResponseDTO, error) {
	if st == nil || st.host == nil {
		return ValidateResponseDTO{}, nil
	}
	resp := BuildValidateResponse(paths)
	if !resp.Valid {
		return resp, nil
	}
	if err := st.host.ValidateFlowPaths(ctx, paths); err != nil {
		resp.Valid = false
		resp.PathErrors = append(resp.PathErrors, PathErrorDTO{
			PathIndex: -1,
			Code:      "flow_reference",
			Message:   err.Error(),
		})
	}
	return resp, nil
}
