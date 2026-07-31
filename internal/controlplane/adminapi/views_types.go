package adminapi

type SavedViewDTO struct {
	ID         string         `json:"id"`
	OwnerID    string         `json:"owner_id"`
	CustomerID string         `json:"customer_id"`
	Name       string         `json:"name"`
	ReportKey  string         `json:"report_key"`
	Spec       map[string]any `json:"spec"`
	IsShared   bool           `json:"is_shared"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

type CreateViewRequest struct {
	CustomerID string         `json:"customer_id"`
	Name       string         `json:"name"`
	ReportKey  string         `json:"report_key"`
	Spec       map[string]any `json:"spec"`
	IsShared   bool           `json:"is_shared"`
}

type UpdateViewRequest struct {
	Name      string         `json:"name"`
	ReportKey string         `json:"report_key"`
	Spec      map[string]any `json:"spec"`
	IsShared  bool           `json:"is_shared"`
}
