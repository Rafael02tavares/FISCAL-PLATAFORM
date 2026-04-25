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
	SourceCFOP       string `json:"source_cfop"`
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
	IPICST              string `json:"ipi_cst"`
	IPIRate             string `json:"ipi_rate"`
	IPICEnq             string `json:"ipi_cenq"`
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

type CESTReference struct {
	Code        string `json:"code"`
	NCMCode     string `json:"ncm_code"`
	Segment     string `json:"segment"`
	Description string `json:"description"`
	LegalSource string `json:"legal_source"`
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

type TaxDiagnosticItem struct {
	Area   string `json:"area"`
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Action string `json:"action"`
}

type TaxDiagnostics struct {
	Ready     int                 `json:"ready"`
	Attention int                 `json:"attention"`
	Missing   int                 `json:"missing"`
	Items     []TaxDiagnosticItem `json:"items"`
}

type DecisionSummary struct {
	Status               string   `json:"status"`
	Severity             string   `json:"severity"`
	Title                string   `json:"title"`
	Message              string   `json:"message"`
	CanUseOperationally  bool     `json:"can_use_operationally"`
	RequiresManualReview bool     `json:"requires_manual_review"`
	BlockingReasons      []string `json:"blocking_reasons"`
	NextActions          []string `json:"next_actions"`
}

type SuggestResponse struct {
	SelectedOperation SelectedOperation `json:"selected_operation"`
	MatchType         string            `json:"match_type"`
	ConfidenceScore   float64           `json:"confidence_score"`
	Suggestion        Suggestion        `json:"suggestion"`
	CESTReference     *CESTReference    `json:"cest_reference,omitempty"`
	LegalBasis        []LegalBasisItem  `json:"legal_basis"`
	Warnings          []string          `json:"warnings"`
	Diagnostics       TaxDiagnostics    `json:"diagnostics"`
	DecisionSummary   DecisionSummary   `json:"decision_summary"`
}
