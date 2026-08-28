package supply

type auditIDChange struct {
	ID int64 `json:"id"`
}

type auditSellerCreateChange struct {
	SellerID string `json:"seller_id"`
	Domain   string `json:"domain"`
}

type auditSellerUpdateChange struct {
	ID       int64  `json:"id"`
	SellerID string `json:"seller_id"`
}

type auditAdsTxtDomainChange struct {
	Domain string `json:"domain"`
}
