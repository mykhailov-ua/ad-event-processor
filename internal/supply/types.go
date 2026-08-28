package supply

type SellerDTO struct {
	ID             int64  `json:"id"`
	SellerID       string `json:"seller_id"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	Name           string `json:"name"`
	IsConfidential bool   `json:"is_confidential"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type AdsTxtEntryDTO struct {
	ID                 int64  `json:"id"`
	Domain             string `json:"domain"`
	PublisherAccountID string `json:"publisher_account_id"`
	Relationship       string `json:"relationship"`
	CertAuthorityID    string `json:"cert_authority_id,omitempty"`
	SortOrder          int32  `json:"sort_order"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type SellerWriteRequest struct {
	SellerID       string `json:"seller_id"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	Name           string `json:"name"`
	IsConfidential bool   `json:"is_confidential"`
}

type AdsTxtWriteRequest struct {
	Domain             string `json:"domain"`
	PublisherAccountID string `json:"publisher_account_id"`
	Relationship       string `json:"relationship"`
	CertAuthorityID    string `json:"cert_authority_id"`
	SortOrder          int32  `json:"sort_order"`
}

type ExportPathDTO struct {
	Path string `json:"path"`
}

type ValidationDTO struct {
	SellersJSONValid      bool     `json:"sellers_json_valid"`
	SellersChecksumSHA256 string   `json:"sellers_checksum_sha256"`
	SellersCount          int      `json:"sellers_count"`
	AdsTxtValid           bool     `json:"ads_txt_valid"`
	AdsTxtChecksumSHA256  string   `json:"ads_txt_checksum_sha256"`
	AdsTxtLineCount       int      `json:"ads_txt_line_count"`
	Issues                []string `json:"issues,omitempty"`
}

type ChainNode struct {
	ASI string `json:"asi"`
	SID string `json:"sid"`
	RID string `json:"rid,omitempty"`
	HP  int    `json:"hp"`
}

type CampaignChainDTO struct {
	CampaignID string      `json:"campaign_id"`
	Nodes      []ChainNode `json:"nodes"`
}

type SellerCreateSpec struct {
	SellerID       string `json:"seller_id"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	Name           string `json:"name"`
	IsConfidential bool   `json:"is_confidential"`
}

type SellerUpdateSpec struct {
	SellerID       string `json:"seller_id"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	Name           string `json:"name"`
	IsConfidential bool   `json:"is_confidential"`
}

type AdsTxtEntryCreateSpec struct {
	Domain             string `json:"domain"`
	PublisherAccountID string `json:"publisher_account_id"`
	Relationship       string `json:"relationship"`
	CertAuthorityID    string `json:"cert_authority_id"`
	SortOrder          int32  `json:"sort_order"`
}

type AdsTxtEntryUpdateSpec struct {
	Domain             string `json:"domain"`
	PublisherAccountID string `json:"publisher_account_id"`
	Relationship       string `json:"relationship"`
	CertAuthorityID    string `json:"cert_authority_id"`
	SortOrder          int32  `json:"sort_order"`
}

type FilesPayload struct {
	Trigger string `json:"trigger"`
}
