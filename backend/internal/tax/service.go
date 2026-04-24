package tax

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/fiscaloperations"
	"github.com/rafa/fiscal-platform/backend/internal/icmsrates"
	"github.com/rafa/fiscal-platform/backend/internal/legalbasis"
)

type Service struct {
	repo              *Repository
	fiscalOpService   *fiscaloperations.Service
	legalBasisService *legalbasis.Service
	icmsRateService   *icmsrates.Service
}

type legalRulePayload struct {
	CFOP              string `json:"cfop"`
	NCM               string `json:"ncm"`
	NCMEx             string `json:"ncm_ex"`
	CEST              string `json:"cest"`
	CClasTrib         string `json:"cclas_trib"`
	CSOSN             string `json:"csosn"`
	PISCST            string `json:"pis_cst"`
	COFINSCST         string `json:"cofins_cst"`
	ICMSCST           string `json:"icms_cst"`
	CBenef            string `json:"cbenef"`
	IBSRate           string `json:"ibs_rate"`
	CBSRate           string `json:"cbs_rate"`
	PISRate           string `json:"pis_rate"`
	COFINSRate        string `json:"cofins_rate"`
	ICMSRate          string `json:"icms_rate"`
	ICMSBaseReduction string `json:"icms_base_reduction"`
	FCPRate           string `json:"fcp_rate"`
	ICMSSTRate        string `json:"icms_st_rate"`
	SelectiveTaxCode  string `json:"selective_tax_code"`
	SelectiveTaxRate  string `json:"selective_tax_rate"`
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
	icmsRateService *icmsrates.Service,
) *Service {
	return &Service{
		repo:              repo,
		fiscalOpService:   fiscalOpService,
		legalBasisService: legalBasisService,
		icmsRateService:   icmsRateService,
	}
}

func (s *Service) Suggest(ctx context.Context, req SuggestRequest) (*SuggestResponse, error) {
	op, err := s.fiscalOpService.ResolveOperation(ctx, req.OperationCode)
	if err != nil {
		return nil, err
	}

	item, err := s.repo.FindBestMatch(
		ctx,
		req.OrganizationID,
		req.GTIN,
		req.Description,
		req.NCMCode,
		req.TaxRegime,
		req.TargetCRT,
		req.OperationCode,
		req.EmitterUF,
		req.RecipientUF,
	)
	if err != nil {
		item = buildFallbackTaxMatch(req)
	}

	resp := s.buildBaseResponse(op, item)
	applyTaxRegimeRules(req, item, resp)
	applyEntryProfileRules(req, op, item, resp)
	s.enrichCFOPSuggestion(ctx, req, op, resp)

	rules, err := s.legalBasisService.FindApplicableRules(ctx, legalbasis.FindApplicableRulesParams{
		OperationCode: op.Code,
		TaxRegime:     req.TaxRegime,
		NCMCode:       resp.Suggestion.NCM,
		CEST:          resp.Suggestion.CEST,
		CClasTrib:     resp.Suggestion.CClasTrib,
		CFOP:          resp.Suggestion.CFOP,
		EmitterUF:     req.EmitterUF,
		RecipientUF:   req.RecipientUF,
	})
	if err == nil {
		resp.LegalBasis = buildLegalBasisItems(rules)
		applyLegalRules(resp, rules)
	}

	s.enrichInterstateReference(ctx, req, resp)
	resp.Warnings = collectSuggestWarnings(req, item, resp)

	return resp, nil
}

func (s *Service) enrichCFOPSuggestion(
	ctx context.Context,
	req SuggestRequest,
	op *fiscaloperations.FiscalOperation,
	resp *SuggestResponse,
) {
	if resp == nil || op == nil {
		return
	}

	match, err := s.repo.FindSuggestedCFOP(
		ctx,
		op.DefaultCFOP,
		op.Name,
		op.Direction,
		req.EmitterUF,
		req.RecipientUF,
	)
	if err != nil || match == nil || strings.TrimSpace(match.Code) == "" {
		return
	}

	resp.Suggestion.CFOP = match.Code
	if strings.TrimSpace(resp.SelectedOperation.CFOP) == "" {
		resp.SelectedOperation.CFOP = match.Code
	}

	if match.ConfidenceScore > resp.ConfidenceScore && resp.MatchType == "ncm_catalog" {
		resp.ConfidenceScore = match.ConfidenceScore
	}
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

		SuggestedNCM:   resp.Suggestion.NCM,
		SuggestedNCMEx: resp.Suggestion.NCMEx,
		SuggestedCEST:  resp.Suggestion.CEST,
		SuggestedCFOP:  resp.Suggestion.CFOP,

		SuggestedPISCST:        resp.Suggestion.PISCST,
		SuggestedCOFINSCST:     resp.Suggestion.COFINSCST,
		SuggestedPISRevCode:    resp.Suggestion.PISRevenueCode,
		SuggestedCOFINSRevCode: resp.Suggestion.COFINSRevenueCode,

		SuggestedICMSCST:           resp.Suggestion.ICMSCST,
		SuggestedCSOSN:             resp.Suggestion.CSOSN,
		SuggestedCBenef:            resp.Suggestion.CBenef,
		SuggestedICMS:              resp.Suggestion.ICMSValue,
		SuggestedIPI:               resp.Suggestion.IPIValue,
		SuggestedPIS:               resp.Suggestion.PISValue,
		SuggestedCOFINS:            resp.Suggestion.COFINSValue,
		SuggestedPISRate:           resp.Suggestion.PISRate,
		SuggestedCOFINSRate:        resp.Suggestion.COFINSRate,
		SuggestedICMSRate:          resp.Suggestion.ICMSRate,
		SuggestedICMSBaseReduction: resp.Suggestion.ICMSBaseReduction,
		SuggestedFCPRate:           resp.Suggestion.FCPRate,
		SuggestedICMSSTRate:        resp.Suggestion.ICMSSTRate,

		SuggestedIBSRate:          resp.Suggestion.IBSRate,
		SuggestedCBSRate:          resp.Suggestion.CBSRate,
		SuggestedSelectiveTaxCode: resp.Suggestion.SelectiveTaxCode,
		SuggestedSelectiveTaxRate: resp.Suggestion.SelectiveTaxRate,

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
	op *fiscaloperations.FiscalOperation,
	item *TaxMatch,
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
			NCM:                 item.NCM,
			NCMEx:               item.NCMEx,
			CEST:                item.CEST,
			CClasTrib:           item.CClasTrib,
			CFOP:                firstNonEmpty(item.CFOP, op.DefaultCFOP),
			CSOSN:               item.CSOSN,
			PISCST:              item.PISCST,
			COFINSCST:           item.COFINSCST,
			PISRevenueCode:      item.PISRevenueCode,
			COFINSRevenueCode:   item.COFINSRevenueCode,
			ICMSCST:             item.ICMSCST,
			ICMSValue:           item.ICMSValue,
			IPIValue:            item.IPIValue,
			PISValue:            item.PISValue,
			COFINSValue:         item.COFINSValue,
			PISRate:             item.PISRate,
			COFINSRate:          item.COFINSRate,
			ICMSRate:            item.ICMSRate,
			ICMSBaseReduction:   item.ICMSBaseReduction,
			FCPRate:             item.FCPRate,
			ICMSSTRate:          item.ICMSSTRate,
			DIFALInternalRate:   "",
			DIFALInterstateRate: "",
			DIFALDifferenceRate: "",
			DIFALMode:           "",
			CBenef:              item.CBenef,
			IBSRate:             item.IBSRate,
			CBSRate:             item.CBSRate,
			SelectiveTaxCode:    item.SelectiveTaxCode,
			SelectiveTaxRate:    item.SelectiveTaxRate,
		},
		LegalBasis: []LegalBasisItem{},
		Warnings:   []string{},
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
	if payload.NCMEx != "" {
		resp.Suggestion.NCMEx = payload.NCMEx
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
	if payload.ICMSCST != "" {
		resp.Suggestion.ICMSCST = payload.ICMSCST
	}
	if payload.CSOSN != "" {
		resp.Suggestion.CSOSN = payload.CSOSN
	}
	if payload.CBenef != "" {
		resp.Suggestion.CBenef = payload.CBenef
	}
}

func applyRateRule(resp *SuggestResponse, payload legalRulePayload) {
	if payload.IBSRate != "" {
		resp.Suggestion.IBSRate = payload.IBSRate
	}
	if payload.CBSRate != "" {
		resp.Suggestion.CBSRate = payload.CBSRate
	}
	if payload.PISRate != "" {
		resp.Suggestion.PISRate = payload.PISRate
	}
	if payload.COFINSRate != "" {
		resp.Suggestion.COFINSRate = payload.COFINSRate
	}
	if payload.ICMSRate != "" {
		resp.Suggestion.ICMSRate = payload.ICMSRate
	}
	if payload.ICMSBaseReduction != "" {
		resp.Suggestion.ICMSBaseReduction = payload.ICMSBaseReduction
	}
	if payload.FCPRate != "" {
		resp.Suggestion.FCPRate = payload.FCPRate
	}
	if payload.ICMSSTRate != "" {
		resp.Suggestion.ICMSSTRate = payload.ICMSSTRate
	}
	if payload.SelectiveTaxCode != "" {
		resp.Suggestion.SelectiveTaxCode = payload.SelectiveTaxCode
	}
	if payload.SelectiveTaxRate != "" {
		resp.Suggestion.SelectiveTaxRate = payload.SelectiveTaxRate
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func applyEntryProfileRules(
	req SuggestRequest,
	op *fiscaloperations.FiscalOperation,
	item *TaxMatch,
	resp *SuggestResponse,
) {
	if op == nil || item == nil || resp == nil {
		return
	}

	if !strings.EqualFold(strings.TrimSpace(op.Direction), "saida") {
		return
	}

	if !isEntryMemoryProfile(item) {
		return
	}

	resp.MatchType = "entry_memory"
	if resp.ConfidenceScore > 0.58 {
		resp.ConfidenceScore = 0.58
	}

	// Classificacao fiscal da entrada costuma ser reutilizavel,
	// mas a tributacao de saida precisa de mais cautela.
	resp.Suggestion.ICMSCST = ""
	resp.Suggestion.CSOSN = ""
	resp.Suggestion.ICMSValue = ""
	resp.Suggestion.ICMSRate = ""
	resp.Suggestion.ICMSBaseReduction = ""
	resp.Suggestion.FCPRate = ""
	resp.Suggestion.ICMSSTRate = ""
	resp.Suggestion.CBenef = ""

	reusePIS := isReusableOutputPISCST(resp.Suggestion.PISCST) && isReusableSourceContributionCST(req.SourcePISCST, isReusableOutputPISCST)
	reuseCOFINS := isReusableOutputCOFINSCST(resp.Suggestion.COFINSCST) && isReusableSourceContributionCST(req.SourceCOFINSCST, isReusableOutputCOFINSCST)

	if hasDistinctContext(item.TargetTaxRegime, req.TaxRegime) || hasDistinctContext(item.TargetCRT, req.TargetCRT) {
		reusePIS = false
		reuseCOFINS = false
	}

	if strings.TrimSpace(req.SourcePISCST) != "" && strings.TrimSpace(item.PISCST) != "" && !strings.EqualFold(strings.TrimSpace(req.SourcePISCST), strings.TrimSpace(item.PISCST)) {
		reusePIS = false
		if resp.ConfidenceScore > 0.52 {
			resp.ConfidenceScore = 0.52
		}
	}

	if strings.TrimSpace(req.SourceCOFINSCST) != "" && strings.TrimSpace(item.COFINSCST) != "" && !strings.EqualFold(strings.TrimSpace(req.SourceCOFINSCST), strings.TrimSpace(item.COFINSCST)) {
		reuseCOFINS = false
		if resp.ConfidenceScore > 0.52 {
			resp.ConfidenceScore = 0.52
		}
	}

	if !reusePIS {
		resp.Suggestion.PISCST = ""
		resp.Suggestion.PISRate = ""
		resp.Suggestion.PISValue = ""
		resp.Suggestion.PISRevenueCode = ""
	}

	if !reuseCOFINS {
		resp.Suggestion.COFINSCST = ""
		resp.Suggestion.COFINSRate = ""
		resp.Suggestion.COFINSValue = ""
		resp.Suggestion.COFINSRevenueCode = ""
	}
}

func isEntryMemoryProfile(item *TaxMatch) bool {
	sourceType := strings.ToLower(strings.TrimSpace(item.SourceType))
	if sourceType == "invoice_import_entry" {
		return true
	}

	cfop := strings.TrimSpace(item.CFOP)
	if cfop == "" {
		return false
	}

	switch cfop[0] {
	case '1', '2', '3':
		return true
	default:
		return false
	}
}

func isReusableOutputPISCST(value string) bool {
	switch strings.TrimSpace(value) {
	case "01", "02", "03", "04", "05", "06", "07", "08", "09", "49", "98", "99":
		return true
	default:
		return false
	}
}

func isReusableOutputCOFINSCST(value string) bool {
	switch strings.TrimSpace(value) {
	case "01", "02", "03", "04", "05", "06", "07", "08", "09", "49", "98", "99":
		return true
	default:
		return false
	}
}

func isReusableSourceContributionCST(value string, validator func(string) bool) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	return validator(value)
}

func buildFallbackTaxMatch(req SuggestRequest) *TaxMatch {
	return &TaxMatch{
		MatchType:       "context_fallback",
		ConfidenceScore: 0.35,
		NCM:             strings.TrimSpace(req.NCMCode),
	}
}

func applyTaxRegimeRules(req SuggestRequest, item *TaxMatch, resp *SuggestResponse) {
	if resp == nil || item == nil {
		return
	}

	regime := strings.ToLower(strings.TrimSpace(req.TaxRegime))
	switch {
	case strings.Contains(regime, "simples"):
		resp.Suggestion.ICMSCST = ""
	case regime != "":
		resp.Suggestion.CSOSN = ""
	}

	if hasDistinctContext(item.TargetTaxRegime, req.TaxRegime) {
		resp.Suggestion.PISCST = ""
		resp.Suggestion.COFINSCST = ""
		resp.Suggestion.PISRate = ""
		resp.Suggestion.COFINSRate = ""
		resp.Suggestion.PISValue = ""
		resp.Suggestion.COFINSValue = ""
		resp.Suggestion.PISRevenueCode = ""
		resp.Suggestion.COFINSRevenueCode = ""
	}

	if hasDistinctContext(item.TargetCRT, req.TargetCRT) {
		resp.Suggestion.ICMSCST = ""
		resp.Suggestion.CSOSN = ""
		resp.Suggestion.PISCST = ""
		resp.Suggestion.COFINSCST = ""
		resp.Suggestion.PISRate = ""
		resp.Suggestion.COFINSRate = ""
		resp.Suggestion.PISValue = ""
		resp.Suggestion.COFINSValue = ""
		resp.Suggestion.PISRevenueCode = ""
		resp.Suggestion.COFINSRevenueCode = ""
	}
}

func collectSuggestWarnings(req SuggestRequest, item *TaxMatch, resp *SuggestResponse) []string {
	warnings := make([]string, 0, 4)

	if item == nil || resp == nil {
		return warnings
	}

	switch strings.TrimSpace(resp.MatchType) {
	case "context_fallback":
		warnings = append(warnings, "A sugestao foi montada sem historico de produto; use NCM, operacao e base legal como apoio inicial.")
	case "ncm_catalog":
		warnings = append(warnings, "A sugestao foi baseada principalmente no catalogo NCM. Revise CSTs e aliquotas antes de usar em producao.")
	case "entry_memory":
		warnings = append(warnings, "A memoria encontrada veio de nota de entrada. A classificacao pode apoiar a saida, mas a tributacao de venda precisa de validacao.")
	}

	if strings.TrimSpace(req.TaxRegime) == "" {
		warnings = append(warnings, "O regime tributario nao foi informado. Algumas regras legais podem nao ter sido aplicadas com a melhor precisao.")
	} else if strings.TrimSpace(item.TargetTaxRegime) == "" {
		warnings = append(warnings, "A memoria fiscal encontrada nao traz regime tributario explicito. Revise PIS e COFINS antes de reutilizar a sugestao.")
	} else if !strings.EqualFold(strings.TrimSpace(item.TargetTaxRegime), strings.TrimSpace(req.TaxRegime)) {
		warnings = append(warnings, "A memoria fiscal encontrada foi registrada para outro regime tributario. Revise contribuicoes antes de confirmar a sugestao.")
	} else if strings.Contains(strings.ToLower(strings.TrimSpace(req.TaxRegime)), "simples") {
		if strings.TrimSpace(resp.Suggestion.CSOSN) == "" {
			warnings = append(warnings, "Para Simples Nacional, revise o CSOSN da operacao antes de concluir a tributacao.")
		}
	} else if strings.TrimSpace(resp.Suggestion.ICMSCST) == "" {
		warnings = append(warnings, "Para regimes fora do Simples, revise o CST de ICMS da saida antes de concluir a tributacao.")
	}

	if strings.TrimSpace(resp.Suggestion.ICMSCST) == "" && strings.TrimSpace(resp.Suggestion.CSOSN) == "" {
		warnings = append(warnings, "ICMS de saida ainda depende de regra especifica por operacao, UF e regime.")
	}

	if hasDistinctContext(item.TargetCRT, req.TargetCRT) {
		warnings = append(warnings, "A memoria fiscal encontrada foi registrada para outro CRT. Revise CST de ICMS, CSOSN e contribuicoes antes de usar.")
	}

	if strings.TrimSpace(req.SourcePISCST) != "" && strings.TrimSpace(item.PISCST) != "" && !strings.EqualFold(strings.TrimSpace(req.SourcePISCST), strings.TrimSpace(item.PISCST)) {
		warnings = append(warnings, "O CST de PIS capturado na nota de entrada difere da memoria fiscal encontrada. A sugestao foi conservadora nas contribuicoes.")
	}

	if strings.TrimSpace(req.SourceCOFINSCST) != "" && strings.TrimSpace(item.COFINSCST) != "" && !strings.EqualFold(strings.TrimSpace(req.SourceCOFINSCST), strings.TrimSpace(item.COFINSCST)) {
		warnings = append(warnings, "O CST de COFINS capturado na nota de entrada difere da memoria fiscal encontrada. Revise a saida antes de confirmar.")
	}

	if strings.TrimSpace(item.OrganizationID) == "" {
		warnings = append(warnings, "A memoria fiscal reaproveitada nao esta vinculada a uma organizacao especifica. Priorize validacao manual antes de usar em producao.")
	}

	if strings.EqualFold(strings.TrimSpace(resp.Suggestion.DIFALMode), "INTERSTATE") && strings.TrimSpace(resp.Suggestion.DIFALDifferenceRate) != "" {
		warnings = append(warnings, "A leitura de DIFAL foi reforcada com a matriz de aliquotas por UF. Revise excecoes por produto importado, beneficio estadual e regra especifica da operacao.")
	}

	return warnings
}

func (s *Service) enrichInterstateReference(
	ctx context.Context,
	req SuggestRequest,
	resp *SuggestResponse,
) {
	if s.icmsRateService == nil || resp == nil {
		return
	}

	issuerUF := strings.TrimSpace(strings.ToUpper(req.EmitterUF))
	recipientUF := strings.TrimSpace(strings.ToUpper(req.RecipientUF))
	if issuerUF == "" || recipientUF == "" {
		return
	}

	reference, err := s.icmsRateService.ResolveReference(ctx, issuerUF, recipientUF)
	if err != nil || reference == nil {
		return
	}

	resp.Suggestion.DIFALMode = reference.Mode
	resp.Suggestion.DIFALInternalRate = reference.InternalRate
	resp.Suggestion.DIFALInterstateRate = reference.InterstateRate
	resp.Suggestion.DIFALDifferenceRate = reference.DifferenceRate

	if strings.TrimSpace(resp.Suggestion.FCPRate) == "" {
		resp.Suggestion.FCPRate = reference.FCPRate
	}

	if strings.EqualFold(reference.Mode, "INTERSTATE") {
		resp.LegalBasis = append(resp.LegalBasis, LegalBasisItem{
			TaxType:       "DIFAL",
			Title:         "Referencia operacional de aliquotas por UF",
			ReferenceCode: "ICMS 2026",
			Jurisdiction:  "STATE",
			UF:            recipientUF,
			AppliedReason: firstNonEmpty(reference.Notes, "Aliquota interna e interestadual resolvidas a partir da matriz estadual vigente."),
			Weight:        "uf_reference",
		})
	}
}

func hasDistinctContext(stored string, requested string) bool {
	stored = strings.TrimSpace(stored)
	requested = strings.TrimSpace(requested)
	if stored == "" || requested == "" {
		return false
	}
	return !strings.EqualFold(stored, requested)
}
