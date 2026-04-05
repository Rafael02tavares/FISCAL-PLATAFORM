package tax

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/fiscaloperations"
	"github.com/rafa/fiscal-platform/backend/internal/legalbasis"
)

type Service struct {
	repo              *Repository
	fiscalOpService   *fiscaloperations.Service
	legalBasisService *legalbasis.Service
}

type legalRulePayload struct {
	CFOP              string `json:"cfop"`
	NCM               string `json:"ncm"`
	CEST              string `json:"cest"`
	CClasTrib         string `json:"cclas_trib"`
	PISCST            string `json:"pis_cst"`
	COFINSCST         string `json:"cofins_cst"`
	IBSRate           string `json:"ibs_rate"`
	CBSRate           string `json:"cbs_rate"`
	PISRevenueCode    string `json:"pis_revenue_code"`
	COFINSRevenueCode string `json:"cofins_revenue_code"`
	PISValue          string `json:"pis_value"`
	COFINSValue       string `json:"cofins_value"`
	ICMSValue         string `json:"icms_value"`
	IPIValue          string `json:"ipi_value"`
}

func NewService(
	repo *Repository,
	fiscalOpService *fiscaloperations.Service,
	legalBasisService *legalbasis.Service,
) *Service {
	return &Service{
		repo:              repo,
		fiscalOpService:   fiscalOpService,
		legalBasisService: legalBasisService,
	}
}

func (s *Service) Suggest(ctx context.Context, req SuggestRequest) (*SuggestResponse, error) {
	op, err := s.fiscalOpService.ResolveOperation(ctx, req.OperationCode)
	if err != nil {
		return nil, err
	}

	item, err := s.repo.FindBestMatch(ctx, req.GTIN, req.Description)
	if err != nil {
		return nil, err
	}

	resp := s.buildBaseResponse(op, item)

	rules, err := s.legalBasisService.FindApplicableRules(ctx, legalbasis.FindApplicableRulesParams{
		OperationCode: op.Code,
		TaxRegime:     req.TaxRegime,
		NCMCode:       resp.Suggestion.NCM,
		EmitterUF:     req.EmitterUF,
		RecipientUF:   req.RecipientUF,
	})
	if err == nil {
		resp.LegalBasis = buildLegalBasisItems(rules)
		applyLegalRules(resp, rules)
	}

	return resp, nil
}

func (s *Service) PersistSuggestion(
	ctx context.Context,
	organizationID string,
	req SuggestRequest,
	resp *SuggestResponse,
) error {
	if strings.TrimSpace(organizationID) == "" {
		return errors.New("organizationID is required")
	}

	if resp == nil {
		return errors.New("suggest response is required")
	}

	suggestionLogID, err := s.repo.CreateSuggestionLog(ctx, CreateSuggestionLogParams{
		OrganizationID: organizationID,

		GTIN:          req.GTIN,
		Description:   req.Description,
		OperationCode: resp.SelectedOperation.Code,
		CClasTrib:     resp.Suggestion.CClasTrib,

		SuggestedNCM:  resp.Suggestion.NCM,
		SuggestedCEST: resp.Suggestion.CEST,
		SuggestedCFOP: resp.Suggestion.CFOP,

		SuggestedPISCST:        resp.Suggestion.PISCST,
		SuggestedCOFINSCST:     resp.Suggestion.COFINSCST,
		SuggestedPISRevCode:    resp.Suggestion.PISRevenueCode,
		SuggestedCOFINSRevCode: resp.Suggestion.COFINSRevenueCode,

		SuggestedICMS:   resp.Suggestion.ICMSValue,
		SuggestedIPI:    resp.Suggestion.IPIValue,
		SuggestedPIS:    resp.Suggestion.PISValue,
		SuggestedCOFINS: resp.Suggestion.COFINSValue,

		SuggestedIBSRate: resp.Suggestion.IBSRate,
		SuggestedCBSRate: resp.Suggestion.CBSRate,

		MatchType:       resp.MatchType,
		ConfidenceScore: resp.ConfidenceScore,
	})
	if err != nil {
		return err
	}

	for _, item := range resp.LegalBasis {
		if strings.TrimSpace(item.LegalSourceID) == "" {
			continue
		}

		if err := s.repo.CreateSuggestionLegalBasis(ctx, CreateSuggestionLegalBasisParams{
			SuggestionLogID: suggestionLogID,
			LegalSourceID:   item.LegalSourceID,
			TaxType:         item.TaxType,
			AppliedReason:   item.AppliedReason,
			Weight:          item.Weight,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) buildBaseResponse(
	op *fiscaloperations.Operation,
	item *BestMatchResult,
) *SuggestResponse {
	return &SuggestResponse{
		SelectedOperation: SelectedOperation{
			Code: op.Code,
			Name: op.Name,
			CFOP: op.DefaultCFOP,
		},
		MatchType:       item.MatchType,
		ConfidenceScore: item.ConfidenceScore,
		Suggestion: Suggestion{
			NCM:               item.NCM,
			CEST:              item.CEST,
			CClasTrib:         item.CClasTrib,
			CFOP:              op.DefaultCFOP,
			PISCST:            item.PISCST,
			COFINSCST:         item.COFINSCST,
			PISRevenueCode:    item.PISRevenueCode,
			COFINSRevenueCode: item.COFINSRevenueCode,
			ICMSValue:         item.ICMSValue,
			IPIValue:          item.IPIValue,
			PISValue:          item.PISValue,
			COFINSValue:       item.COFINSValue,
			IBSRate:           item.IBSRate,
			CBSRate:           item.CBSRate,
		},
		LegalBasis: []LegalBasisItem{},
	}
}

func buildLegalBasisItems(rules []legalbasis.ApplicableLegalRule) []LegalBasisItem {
	items := make([]LegalBasisItem, 0, len(rules))

	for _, rule := range rules {
		items = append(items, LegalBasisItem{
			LegalSourceID: rule.LegalSourceID,
			TaxType:       rule.TaxType,
			Title:         rule.Title,
			ReferenceCode: rule.ReferenceCode,
			Jurisdiction:  rule.Jurisdiction,
			UF:            rule.UF,
			AppliedReason: "regra legal aplicável ao contexto",
			Weight:        rule.ConfidenceBase,
		})
	}

	return items
}

func applyLegalRules(resp *SuggestResponse, rules []legalbasis.ApplicableLegalRule) {
	if resp == nil {
		return
	}

	for _, rule := range rules {
		payload, ok := parseLegalRulePayload(rule.ValueContent)
		if !ok {
			continue
		}

		applyLegalRuleByType(resp, rule.ValueType, payload)
	}
}

func parseLegalRulePayload(raw string) (legalRulePayload, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return legalRulePayload{}, false
	}

	var payload legalRulePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return legalRulePayload{}, false
	}

	return payload, true
}

func applyLegalRuleByType(resp *SuggestResponse, valueType string, payload legalRulePayload) {
	switch valueType {
	case "cfop_rule":
		applyCFOPRule(resp, payload)

	case "classification_rule":
		applyClassificationRule(resp, payload)

	case "cst_rule":
		applyCSTRule(resp, payload)

	case "rate_rule":
		applyRateRule(resp, payload)
	}
}

func applyCFOPRule(resp *SuggestResponse, payload legalRulePayload) {
	if payload.CFOP != "" {
		resp.Suggestion.CFOP = payload.CFOP
	}
}

func applyClassificationRule(resp *SuggestResponse, payload legalRulePayload) {
	if payload.NCM != "" {
		resp.Suggestion.NCM = payload.NCM
	}
	if payload.CEST != "" {
		resp.Suggestion.CEST = payload.CEST
	}
	if payload.CClasTrib != "" {
		resp.Suggestion.CClasTrib = payload.CClasTrib
	}
}

func applyCSTRule(resp *SuggestResponse, payload legalRulePayload) {
	if payload.PISCST != "" {
		resp.Suggestion.PISCST = payload.PISCST
	}
	if payload.COFINSCST != "" {
		resp.Suggestion.COFINSCST = payload.COFINSCST
	}
}

func applyRateRule(resp *SuggestResponse, payload legalRulePayload) {
	if payload.IBSRate != "" {
		resp.Suggestion.IBSRate = payload.IBSRate
	}
	if payload.CBSRate != "" {
		resp.Suggestion.CBSRate = payload.CBSRate
	}
	if payload.PISRevenueCode != "" {
		resp.Suggestion.PISRevenueCode = payload.PISRevenueCode
	}
	if payload.COFINSRevenueCode != "" {
		resp.Suggestion.COFINSRevenueCode = payload.COFINSRevenueCode
	}
	if payload.PISValue != "" {
		resp.Suggestion.PISValue = payload.PISValue
	}
	if payload.COFINSValue != "" {
		resp.Suggestion.COFINSValue = payload.COFINSValue
	}
	if payload.ICMSValue != "" {
		resp.Suggestion.ICMSValue = payload.ICMSValue
	}
	if payload.IPIValue != "" {
		resp.Suggestion.IPIValue = payload.IPIValue
	}
}