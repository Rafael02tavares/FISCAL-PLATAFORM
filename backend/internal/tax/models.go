package tax

type SuggestRequest struct {
	OrganizationID   string `json:"-"`
	GTIN             string `json:"gtin"`
	Description      string `json:"description"`
	NCMCode          string `json:"ncm_code"`
	OperationCode    string `json:"operation_code"`
	EmitterUF        string `json:"emitter_uf"`
	RecipientUF      string `json:"recipient_uf"`
	TaxRegime        string `json:"tax_regime"`
	TargetCRT        string `json:"target_crt"`
	SourceICMSCST    string `json:"source_icms_cst"`
	SourceICMSRate   string `json:"source_icms_rate"`
	SourcePISCST     string `json:"source_pis_cst"`
	SourcePISRate    string `json:"source_pis_rate"`
	SourceCOFINSCST  string `json:"source_cofins_cst"`
	SourceCOFINSRate string `json:"source_cofins_rate"`
}

type SelectedOperation struct {
	Code string `json:"code"`
	Name string `json:"name"`
	CFOP string `json:"cfop"`
}

type Suggestion struct {
	NCM       string `json:"ncm"`
	NCMEx     string `json:"ncm_ex"`
	CEST      string `json:"cest"`
	CClasTrib string `json:"cclas_trib"`
	CFOP      string `json:"cfop"`
	CSOSN     string `json:"csosn"`

	PISCST            string `json:"pis_cst"`
	COFINSCST         string `json:"cofins_cst"`
	PISRevenueCode    string `json:"pis_revenue_code"`
	COFINSRevenueCode string `json:"cofins_revenue_code"`

	ICMSCST             string `json:"icms_cst"`
	ICMSValue           string `json:"icms_value"`
	IPIValue            string `json:"ipi_value"`
	PISValue            string `json:"pis_value"`
	COFINSValue         string `json:"cofins_value"`
	PISRate             string `json:"pis_rate"`
	COFINSRate          string `json:"cofins_rate"`
	ICMSRate            string `json:"icms_rate"`
	ICMSBaseReduction   string `json:"icms_base_reduction"`
	FCPRate             string `json:"fcp_rate"`
	ICMSSTRate          string `json:"icms_st_rate"`
	DIFALInternalRate   string `json:"difal_internal_rate"`
	DIFALInterstateRate string `json:"difal_interstate_rate"`
	DIFALDifferenceRate string `json:"difal_difference_rate"`
	DIFALMode           string `json:"difal_mode"`
	CBenef              string `json:"cbenef"`

	IBSRate          string `json:"ibs_rate"`
	CBSRate          string `json:"cbs_rate"`
	SelectiveTaxCode string `json:"selective_tax_code"`
	SelectiveTaxRate string `json:"selective_tax_rate"`
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
	Warnings          []string          `json:"warnings"`
}
