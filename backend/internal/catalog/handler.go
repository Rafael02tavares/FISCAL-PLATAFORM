package catalog

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/auth"
	"github.com/rafa/fiscal-platform/backend/internal/organizations"
)

type Handler struct {
	service    *Service
	orgService *organizations.Service
}

func NewHandler(service *Service, orgService *organizations.Service) *Handler {
	return &Handler{
		service:    service,
		orgService: orgService,
	}
}

type saveProductRequest struct {
	ProductID string `json:"product_id"`

	ProductCode string `json:"product_code"`
	GTIN        string `json:"gtin"`
	Description string `json:"description"`

	NCM               string `json:"ncm"`
	NCMEx             string `json:"ncm_ex"`
	CEST              string `json:"cest"`
	CFOP              string `json:"cfop"`
	CClasTrib         string `json:"cclas_trib"`
	PISCST            string `json:"pis_cst"`
	COFINSCST         string `json:"cofins_cst"`
	PISRevenueCode    string `json:"pis_revenue_code"`
	COFINSRevenueCode string `json:"cofins_revenue_code"`
	ICMSCST           string `json:"icms_cst"`
	CSOSN             string `json:"csosn"`
	CBenef            string `json:"cbenef"`

	ICMSValue         string `json:"icms_value"`
	IPIValue          string `json:"ipi_value"`
	PISValue          string `json:"pis_value"`
	COFINSValue       string `json:"cofins_value"`
	PISRate           string `json:"pis_rate"`
	COFINSRate        string `json:"cofins_rate"`
	ICMSRate          string `json:"icms_rate"`
	ICMSBaseReduction string `json:"icms_base_reduction"`
	FCPRate           string `json:"fcp_rate"`
	ICMSSTRate        string `json:"icms_st_rate"`
	IBSRate           string `json:"ibs_rate"`
	CBSRate           string `json:"cbs_rate"`
	SelectiveTaxCode  string `json:"selective_tax_code"`
	SelectiveTaxRate  string `json:"selective_tax_rate"`

	OperationCode     string `json:"operation_code"`
	EmitterUF         string `json:"emitter_uf"`
	RecipientUF       string `json:"recipient_uf"`
	OperationNature   string `json:"operation_nature"`
	TargetTaxRegime   string `json:"target_tax_regime"`
	ObservedTaxRegime string `json:"observed_tax_regime"`
	TargetCRT         string `json:"target_crt"`
	ObservedCRT       string `json:"observed_crt"`
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	items, err := h.service.ListCatalogProducts(r.Context(), organizationID, query)
	if err != nil {
		writeCatalogError(w, http.StatusInternalServerError, "LIST_PRODUCTS_FAILED", "nao foi possivel listar os produtos do catalogo")
		return
	}

	writeCatalogJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (h *Handler) SaveProduct(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req saveProductRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeCatalogError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisicao invalido")
		return
	}

	if err := h.service.SaveManualProduct(r.Context(), SaveManualProductParams{
		OrganizationID: organizationID,
		ProductID:      strings.TrimSpace(req.ProductID),
		ProductCode:    strings.TrimSpace(req.ProductCode),
		GTIN:           strings.TrimSpace(req.GTIN),
		Description:    strings.TrimSpace(req.Description),

		NCM:               strings.TrimSpace(req.NCM),
		NCMEx:             strings.TrimSpace(req.NCMEx),
		CEST:              strings.TrimSpace(req.CEST),
		CFOP:              strings.TrimSpace(req.CFOP),
		CClasTrib:         strings.TrimSpace(req.CClasTrib),
		PISCST:            strings.TrimSpace(req.PISCST),
		COFINSCST:         strings.TrimSpace(req.COFINSCST),
		PISRevenueCode:    strings.TrimSpace(req.PISRevenueCode),
		COFINSRevenueCode: strings.TrimSpace(req.COFINSRevenueCode),
		ICMSCST:           strings.TrimSpace(req.ICMSCST),
		CSOSN:             strings.TrimSpace(req.CSOSN),
		CBenef:            strings.TrimSpace(req.CBenef),

		ICMSValue:         strings.TrimSpace(req.ICMSValue),
		IPIValue:          strings.TrimSpace(req.IPIValue),
		PISValue:          strings.TrimSpace(req.PISValue),
		COFINSValue:       strings.TrimSpace(req.COFINSValue),
		PISRate:           strings.TrimSpace(req.PISRate),
		COFINSRate:        strings.TrimSpace(req.COFINSRate),
		ICMSRate:          strings.TrimSpace(req.ICMSRate),
		ICMSBaseReduction: strings.TrimSpace(req.ICMSBaseReduction),
		FCPRate:           strings.TrimSpace(req.FCPRate),
		ICMSSTRate:        strings.TrimSpace(req.ICMSSTRate),
		IBSRate:           strings.TrimSpace(req.IBSRate),
		CBSRate:           strings.TrimSpace(req.CBSRate),
		SelectiveTaxCode:  strings.TrimSpace(req.SelectiveTaxCode),
		SelectiveTaxRate:  strings.TrimSpace(req.SelectiveTaxRate),

		OperationCode:     strings.TrimSpace(req.OperationCode),
		EmitterUF:         strings.ToUpper(strings.TrimSpace(req.EmitterUF)),
		RecipientUF:       strings.ToUpper(strings.TrimSpace(req.RecipientUF)),
		OperationNature:   strings.TrimSpace(req.OperationNature),
		TargetTaxRegime:   strings.TrimSpace(req.TargetTaxRegime),
		ObservedTaxRegime: strings.TrimSpace(req.ObservedTaxRegime),
		TargetCRT:         strings.TrimSpace(req.TargetCRT),
		ObservedCRT:       strings.TrimSpace(req.ObservedCRT),
	}); err != nil {
		writeCatalogError(w, http.StatusBadRequest, "SAVE_PRODUCT_FAILED", err.Error())
		return
	}

	items, _ := h.service.ListCatalogProducts(r.Context(), organizationID, "")
	writeCatalogJSON(w, http.StatusCreated, map[string]any{
		"message": "produto fiscal salvo com sucesso",
		"items":   items,
	})
}

func (h *Handler) authorizeRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeCatalogError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuario nao autenticado")
		return "", false
	}

	organizationID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if organizationID == "" {
		writeCatalogError(w, http.StatusBadRequest, "MISSING_ORGANIZATION_ID", "X-Organization-ID e obrigatorio")
		return "", false
	}

	allowed, err := h.orgService.UserBelongsToOrganization(r.Context(), userID, organizationID)
	if err != nil {
		writeCatalogError(w, http.StatusInternalServerError, "ORGANIZATION_VALIDATION_FAILED", "nao foi possivel validar acesso a organizacao")
		return "", false
	}

	if !allowed {
		writeCatalogError(w, http.StatusForbidden, "FORBIDDEN", "usuario sem acesso a esta organizacao")
		return "", false
	}

	return organizationID, true
}

func writeCatalogJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeCatalogError(w http.ResponseWriter, status int, code, message string) {
	writeCatalogJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
