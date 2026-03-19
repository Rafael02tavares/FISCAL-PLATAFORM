package tax

type SuggestRequest struct {
	GTIN          string `json:"gtin"`
	Description   string `json:"description"`
	OperationCode string `json:"operation_code"`
	EmitterUF     string `json:"emitter_uf"`
	RecipientUF   string `json:"recipient_uf"`
	TaxRegime     string `json:"tax_regime"`
}

type SelectedOperation struct {
	Code string `json:"code"`
	Name string `json:"name"`
	CFOP string `json:"cfop"`
}

type Suggestion struct {
	NCM               string `json:"ncm"`
	CEST              string `json:"cest"`
	CClasTrib         string `json:"cclas_trib"`
	CFOP              string `json:"cfop"`

	PISCST            string `json:"pis_cst"`
	COFINSCST         string `json:"cofins_cst"`
	PISRevenueCode    string `json:"pis_revenue_code"`
	COFINSRevenueCode string `json:"cofins_revenue_code"`

	ICMSValue         string `json:"icms_value"`
	IPIValue          string `json:"ipi_value"`
	PISValue          string `json:"pis_value"`
	COFINSValue       string `json:"cofins_value"`

	IBSRate           string `json:"ibs_rate"`
	CBSRate           string `json:"cbs_rate"`
}

type LegalBasisItem struct {
	LegalSourceID string `json:"legal_source_id"`
	TaxType       string `json:"tax_type"`
	Title         string `json:"title"`
	ReferenceCode string `json:"reference_code"`
	Jurisdiction  string `json:"jurisdiction"`
	UF            string `json:"uf"`
	AppliedReason string `json:"applied_reason"`
	Weight        string `json:"weight"`
}

type SuggestResponse struct {
	SelectedOperation SelectedOperation `json:"selected_operation"`
	MatchType         string            `json:"match_type"`
	ConfidenceScore   float64           `json:"confidence_score"`
	Suggestion        Suggestion        `json:"suggestion"`
	LegalBasis        []LegalBasisItem  `json:"legal_basis"`
}
