package tax

import (
	"context"
	"encoding/json"

	"github.com/rafa/fiscal-platform/backend/internal/fiscaloperations"
	"github.com/rafa/fiscal-platform/backend/internal/legalbasis"
)

type Service struct {
	repo              *Repository
	fiscalOpService   *fiscaloperations.Service
	legalBasisService *legalbasis.Service
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

	resp := &SuggestResponse{
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
		if item.LegalSourceID == "" {
			continue
		}

		_ = s.repo.CreateSuggestionLegalBasis(ctx, CreateSuggestionLegalBasisParams{
			SuggestionLogID: suggestionLogID,
			LegalSourceID:   item.LegalSourceID,
			TaxType:         item.TaxType,
			AppliedReason:   item.AppliedReason,
			Weight:          item.Weight,
		})
	}

	return nil
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
	for _, rule := range rules {
		var payload map[string]string
		if err := json.Unmarshal([]byte(rule.ValueContent), &payload); err != nil {
			continue
		}

		switch rule.ValueType {
		case "cfop_rule":
			if v := payload["cfop"]; v != "" {
				resp.Suggestion.CFOP = v
			}

		case "classification_rule":
			if v := payload["ncm"]; v != "" {
				resp.Suggestion.NCM = v
			}
			if v := payload["cest"]; v != "" {
				resp.Suggestion.CEST = v
			}
			if v := payload["cclas_trib"]; v != "" {
				resp.Suggestion.CClasTrib = v
			}

		case "cst_rule":
			if v := payload["pis_cst"]; v != "" {
				resp.Suggestion.PISCST = v
			}
			if v := payload["cofins_cst"]; v != "" {
				resp.Suggestion.COFINSCST = v
			}

		case "rate_rule":
			if v := payload["ibs_rate"]; v != "" {
				resp.Suggestion.IBSRate = v
			}
			if v := payload["cbs_rate"]; v != "" {
				resp.Suggestion.CBSRate = v
			}
			if v := payload["pis_revenue_code"]; v != "" {
				resp.Suggestion.PISRevenueCode = v
			}
			if v := payload["cofins_revenue_code"]; v != "" {
				resp.Suggestion.COFINSRevenueCode = v
			}
			if v := payload["pis_value"]; v != "" {
				resp.Suggestion.PISValue = v
			}
			if v := payload["cofins_value"]; v != "" {
				resp.Suggestion.COFINSValue = v
			}
			if v := payload["icms_value"]; v != "" {
				resp.Suggestion.ICMSValue = v
			}
			if v := payload["ipi_value"]; v != "" {
				resp.Suggestion.IPIValue = v
			}
		}
	}
}
