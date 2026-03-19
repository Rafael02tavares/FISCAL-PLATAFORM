package legalbasis

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type createLegalSourceRequest struct {
	TaxType       string `json:"tax_type"`
	SourceType    string `json:"source_type"`
	Jurisdiction  string `json:"jurisdiction"`
	UF            string `json:"uf"`
	Title         string `json:"title"`
	ReferenceCode string `json:"reference_code"`
	Description   string `json:"description"`
	OfficialURL   string `json:"official_url"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to"`
	Notes         string `json:"notes"`
}

func (h *Handler) CreateLegalSource(w http.ResponseWriter, r *http.Request) {
	var req createLegalSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.TaxType == "" || req.SourceType == "" || req.Jurisdiction == "" || req.Title == "" {
		http.Error(w, "tax_type, source_type, jurisdiction and title are required", http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateLegalSource(r.Context(), CreateLegalSourceParams{
		TaxType:       req.TaxType,
		SourceType:    req.SourceType,
		Jurisdiction:  req.Jurisdiction,
		UF:            req.UF,
		Title:         req.Title,
		ReferenceCode: req.ReferenceCode,
		Description:   req.Description,
		OfficialURL:   req.OfficialURL,
		EffectiveFrom: req.EffectiveFrom,
		EffectiveTo:   req.EffectiveTo,
		Notes:         req.Notes,
	})
	if err != nil {
		http.Error(w, "cannot create legal source: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id": id,
	})
}

func (h *Handler) ListLegalSources(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	items, err := h.service.ListLegalSources(r.Context(), limit)
	if err != nil {
		http.Error(w, "cannot list legal sources: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if items == nil {
		items = []LegalSource{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": items,
	})
}

type createLegalRuleMappingRequest struct {
	LegalSourceID  string `json:"legal_source_id"`
	TaxType        string `json:"tax_type"`
	OperationCode  string `json:"operation_code"`
	TaxRegime      string `json:"tax_regime"`
	NCMCode        string `json:"ncm_code"`
	CEST           string `json:"cest"`
	CClasTrib      string `json:"cclas_trib"`
	CFOP           string `json:"cfop"`
	PISCST         string `json:"pis_cst"`
	COFINSCST      string `json:"cofins_cst"`
	ICMSCST        string `json:"icms_cst"`
	CSOSN          string `json:"csosn"`
	CBenef         string `json:"cbenef"`
	EmitterUF      string `json:"emitter_uf"`
	RecipientUF    string `json:"recipient_uf"`
	ValueType      string `json:"value_type"`
	ValueContent   string `json:"value_content"`
	Priority       int    `json:"priority"`
	ConfidenceBase string `json:"confidence_base"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to"`
}

func (h *Handler) CreateLegalRuleMapping(w http.ResponseWriter, r *http.Request) {
	var req createLegalRuleMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.LegalSourceID == "" || req.TaxType == "" || req.ValueType == "" {
		http.Error(w, "legal_source_id, tax_type and value_type are required", http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateLegalRuleMapping(r.Context(), CreateLegalRuleMappingParams{
		LegalSourceID:  req.LegalSourceID,
		TaxType:        req.TaxType,
		OperationCode:  req.OperationCode,
		TaxRegime:      req.TaxRegime,
		NCMCode:        req.NCMCode,
		CEST:           req.CEST,
		CClasTrib:      req.CClasTrib,
		CFOP:           req.CFOP,
		PISCST:         req.PISCST,
		COFINSCST:      req.COFINSCST,
		ICMSCST:        req.ICMSCST,
		CSOSN:          req.CSOSN,
		CBenef:         req.CBenef,
		EmitterUF:      req.EmitterUF,
		RecipientUF:    req.RecipientUF,
		ValueType:      req.ValueType,
		ValueContent:   req.ValueContent,
		Priority:       req.Priority,
		ConfidenceBase: req.ConfidenceBase,
		EffectiveFrom:  req.EffectiveFrom,
		EffectiveTo:    req.EffectiveTo,
	})
	if err != nil {
		http.Error(w, "cannot create legal rule mapping: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id": id,
	})
}

func (h *Handler) ListLegalRuleMappings(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	items, err := h.service.ListLegalRuleMappings(r.Context(), limit)
	if err != nil {
		http.Error(w, "cannot list legal rule mappings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if items == nil {
		items = []LegalRuleMapping{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": items,
	})
}
