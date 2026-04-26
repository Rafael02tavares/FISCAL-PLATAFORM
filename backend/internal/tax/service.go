package tax

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/fiscaloperations"
	"github.com/rafa/fiscal-platform/backend/internal/icmsrates"
	"github.com/rafa/fiscal-platform/backend/internal/integrations"
	"github.com/rafa/fiscal-platform/backend/internal/legalbasis"
)

type Service struct {
	repo               *Repository
	fiscalOpService    *fiscaloperations.Service
	legalBasisService  *legalbasis.Service
	icmsRateService    *icmsrates.Service
	integrationService *integrations.Service
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
	IPICST            string `json:"ipi_cst"`
	IPIRate           string `json:"ipi_rate"`
	IPICEnq           string `json:"ipi_cenq"`
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

type monophaseNCMRule struct {
	Prefix      string
	Category    string
	Description string
}

var retailMonophaseNCMRules = []monophaseNCMRule{
	{Prefix: "2202", Category: "bebidas", Description: "refrigerantes e bebidas nao alcoolicas"},
	{Prefix: "2203", Category: "bebidas", Description: "cervejas"},
	{Prefix: "2204", Category: "bebidas", Description: "vinhos"},
	{Prefix: "2205", Category: "bebidas", Description: "vermutes"},
	{Prefix: "2206", Category: "bebidas", Description: "outras bebidas fermentadas"},
	{Prefix: "2208", Category: "bebidas", Description: "bebidas alcoolicas destiladas"},
	{Prefix: "2710", Category: "combustiveis", Description: "gasolina, diesel e outros oleos de petroleo"},
	{Prefix: "2711", Category: "combustiveis", Description: "gas e GLP"},
	{Prefix: "3003", Category: "farmaceuticos", Description: "medicamentos a granel"},
	{Prefix: "3004", Category: "farmaceuticos", Description: "medicamentos embalados"},
	{Prefix: "2402", Category: "cigarros", Description: "cigarros e produtos similares"},
}

func NewService(
	repo *Repository,
	fiscalOpService *fiscaloperations.Service,
	legalBasisService *legalbasis.Service,
	icmsRateService *icmsrates.Service,
	integrationService ...*integrations.Service,
) *Service {
	var integrationSvc *integrations.Service
	if len(integrationService) > 0 {
		integrationSvc = integrationService[0]
	}

	return &Service{
		repo:               repo,
		fiscalOpService:    fiscalOpService,
		legalBasisService:  legalBasisService,
		icmsRateService:    icmsRateService,
		integrationService: integrationSvc,
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
		item = s.findCosmosTaxMatch(ctx, req)
		if item == nil {
			item = buildFallbackTaxMatch(req)
		}
	}

	resp := s.buildBaseResponse(op, item)
	appendExternalProductIdentity(resp, item)
	applyTaxRegimeRules(req, item, resp)
	applyEntryProfileRules(req, op, item, resp)
	s.enrichCFOPSuggestion(ctx, req, op, resp)
	s.enrichCESTSuggestion(ctx, resp)
	applySubstitutionTaxEvidence(req, op, resp)
	applyRetailDefaultRules(req, op, resp)
	s.applyNCMTaxProfiles(ctx, req, op, resp)
	s.applyStateICMSRule(ctx, req, op, resp)

	preLegalBasis := append([]LegalBasisItem{}, resp.LegalBasis...)
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
		resp.LegalBasis = append(preLegalBasis, buildLegalBasisItems(rules)...)
		applyLegalRules(resp, rules)
	}

	applyReferenceProductFiscalRules(req, op, resp)
	applyRetailSpecialOutputRules(req, resp)
	applyFederalContributionBenefitReduction2026(req, resp)
	s.enrichInterstateReference(ctx, req, resp)
	applyTaxRegimeRules(req, item, resp)
	normalizeRegimeSpecificDefaults(req, resp)
	normalizeCFOPByOperationContext(req, op, resp)
	s.enrichWithAIAssistance(ctx, req, resp)
	resp.Warnings = collectSuggestWarnings(req, item, resp)
	resp.Diagnostics = buildTaxDiagnostics(req, resp)
	resp.DecisionSummary = buildDecisionSummary(req, resp)

	return resp, nil
}

func (s *Service) enrichCESTSuggestion(ctx context.Context, resp *SuggestResponse) {
	if resp == nil {
		return
	}

	match, err := s.repo.FindSuggestedCEST(ctx, resp.Suggestion.NCM, resp.Suggestion.CEST)
	if err != nil || match == nil || strings.TrimSpace(match.Code) == "" {
		return
	}

	if strings.TrimSpace(resp.Suggestion.CEST) == "" {
		resp.Suggestion.CEST = match.Code
	}

	resp.CESTReference = &CESTReference{
		Code:        match.Code,
		NCMCode:     match.NCMCode,
		Segment:     match.Segment,
		Description: match.Description,
		LegalSource: match.LegalSource,
	}

	if resp.MatchType == "ncm_catalog" && match.ConfidenceScore > resp.ConfidenceScore {
		resp.ConfidenceScore = match.ConfidenceScore
	}

	resp.LegalBasis = append(resp.LegalBasis, LegalBasisItem{
		TaxType:       "CEST",
		Title:         "Catalogo CEST",
		ReferenceCode: match.Code,
		Jurisdiction:  "NATIONAL",
		AppliedReason: firstNonEmpty(match.Description, "CEST identificado a partir do NCM. O codigo apoia a analise de ST, mas nao confirma substituicao tributaria sem regra estadual aplicavel."),
		Weight:        "0.68",
	})
}

func (s *Service) findCosmosTaxMatch(ctx context.Context, req SuggestRequest) *TaxMatch {
	if s.integrationService == nil {
		return nil
	}

	searchQuery := strings.TrimSpace(req.Description)
	if searchQuery == "" {
		searchQuery = strings.TrimSpace(req.GTIN)
	}
	if searchQuery == "" || strings.TrimSpace(req.OrganizationID) == "" {
		return nil
	}

	result, err := s.integrationService.SearchCosmosProducts(ctx, req.OrganizationID, searchQuery, "", 5)
	if err != nil || !result.OK || len(result.Items) == 0 {
		return nil
	}

	candidate := chooseCosmosCandidate(req, result.Items)
	if candidate == nil || strings.TrimSpace(candidate.NCM) == "" {
		return nil
	}

	confidence := 0.68
	if strings.TrimSpace(candidate.GTIN) != "" {
		confidence += 0.08
	}
	if strings.TrimSpace(candidate.CEST) != "" {
		confidence += 0.06
	}
	if strings.TrimSpace(req.Description) != "" && strings.Contains(
		strings.ToLower(strings.TrimSpace(candidate.Description)),
		strings.ToLower(strings.TrimSpace(req.Description)),
	) {
		confidence += 0.04
	}
	if confidence > 0.84 {
		confidence = 0.84
	}

	return &TaxMatch{
		OrganizationID:  strings.TrimSpace(req.OrganizationID),
		MatchType:       "cosmos_search",
		ConfidenceScore: confidence,
		SourceType:      "external_identity",
		SourceLabel:     "Cosmos BlueSoft",
		SourceReference: "cosmos_search",
		Description:     candidate.Description,
		GTIN:            candidate.GTIN,
		NCM:             normalizeNCMCode(candidate.NCM),
		CEST:            normalizeCESTCode(candidate.CEST),
	}
}

func chooseCosmosCandidate(req SuggestRequest, items []integrations.CosmosProductCandidate) *integrations.CosmosProductCandidate {
	var best *integrations.CosmosProductCandidate
	bestScore := -1
	requestGTIN := normalizeDigits(req.GTIN)
	requestNCM := normalizeNCMCode(req.NCMCode)
	requestDescription := strings.ToLower(strings.TrimSpace(req.Description))

	for i := range items {
		item := &items[i]
		score := 0
		if normalizeNCMCode(item.NCM) != "" {
			score += 20
		}
		if normalizeCESTCode(item.CEST) != "" {
			score += 12
		}
		if requestGTIN != "" && normalizeDigits(item.GTIN) == requestGTIN {
			score += 40
		} else if normalizeDigits(item.GTIN) != "" {
			score += 8
		}
		if requestNCM != "" && normalizeNCMCode(item.NCM) == requestNCM {
			score += 20
		}
		if requestDescription != "" && strings.Contains(strings.ToLower(item.Description), requestDescription) {
			score += 10
		}
		if score > bestScore {
			bestScore = score
			best = item
		}
	}

	return best
}

func appendExternalProductIdentity(resp *SuggestResponse, item *TaxMatch) {
	if resp == nil || item == nil || item.MatchType != "cosmos_search" {
		return
	}

	reference := firstNonEmpty(item.GTIN, item.NCM, item.SourceReference)
	reason := "Produto localizado por descricao na Cosmos BlueSoft; NCM/CEST usados como identidade fiscal para acionar regras internas da plataforma."
	if item.Description != "" {
		reason = "Produto localizado na Cosmos BlueSoft: " + item.Description + ". NCM/CEST usados como identidade fiscal para acionar regras internas da plataforma."
	}

	resp.LegalBasis = append(resp.LegalBasis, LegalBasisItem{
		TaxType:       "PRODUCT_IDENTITY",
		Title:         firstNonEmpty(item.SourceLabel, "Cosmos BlueSoft"),
		ReferenceCode: reference,
		Jurisdiction:  "EXTERNAL",
		AppliedReason: reason,
		Weight:        "external_product_identity",
	})
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
			IPICST:              "",
			IPIRate:             "",
			IPICEnq:             "",
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

func (s *Service) enrichWithAIAssistance(ctx context.Context, req SuggestRequest, resp *SuggestResponse) {
	if s.integrationService == nil || resp == nil {
		return
	}
	if !shouldRequestAIAssistance(req, resp) {
		return
	}

	result, attempted, err := s.integrationService.ClassifyWithOpenAI(ctx, req.OrganizationID, integrations.OpenAIClassificationInput{
		Description: firstNonEmpty(req.Description, resp.Suggestion.NCM),
		GTIN:        req.GTIN,
		NCM:         firstNonEmpty(resp.Suggestion.NCM, req.NCMCode),
		CEST:        resp.Suggestion.CEST,
		UF:          firstNonEmpty(req.RecipientUF, req.EmitterUF),
		TaxRegime:   req.TaxRegime,
		Operation:   req.OperationCode,
	})
	if !attempted {
		return
	}
	if err != nil {
		resp.Warnings = append(resp.Warnings, "Classificacao assistida OpenAI indisponivel nesta consulta; o motor seguiu somente com regras internas.")
		return
	}

	resp.AIAssistance = buildAIAssistance(result)
	if resp.AIAssistance == nil {
		return
	}

	if resp.AIAssistance.Observation != "" || resp.AIAssistance.RecommendedAction != "" {
		resp.LegalBasis = append(resp.LegalBasis, LegalBasisItem{
			TaxType:       "AI_ASSIST",
			Title:         "Classificacao assistida OpenAI",
			ReferenceCode: result.Model,
			Jurisdiction:  "INTERNAL",
			AppliedReason: firstNonEmpty(resp.AIAssistance.RecommendedAction, resp.AIAssistance.Observation, "IA usada apenas como apoio de triagem, sem aplicar regra fiscal automaticamente."),
			Weight:        "ai_assist",
		})
	}

	resp.Warnings = append(resp.Warnings, "IA usada como apoio de triagem. Nao substitui regra fiscal, fonte legal ou aprovacao humana.")
}

func shouldRequestAIAssistance(req SuggestRequest, resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}
	if strings.TrimSpace(req.OrganizationID) == "" {
		return false
	}
	if resp.ConfidenceScore < 0.82 {
		return true
	}
	if strings.TrimSpace(resp.Suggestion.NCM) == "" || strings.TrimSpace(resp.Suggestion.CFOP) == "" {
		return true
	}
	if len(resp.LegalBasis) == 0 {
		return true
	}
	return false
}

func buildAIAssistance(result integrations.OpenAITestResult) *AIAssistance {
	if !result.OK {
		return &AIAssistance{
			Provider: "openai",
			Model:    result.Model,
			Status:   "error",
			Output:   result.Message,
		}
	}

	classification := result.Classification
	return &AIAssistance{
		Provider:          "openai",
		Model:             result.Model,
		Status:            "available",
		Category:          stringFromAIValue(classification["categoria_fiscal_provavel"]),
		Risk:              stringFromAIValue(classification["risco"]),
		Confidence:        stringFromAIValue(classification["confianca"]),
		RecommendedAction: stringFromAIValue(classification["acao_recomendada"]),
		Observation:       stringFromAIValue(classification["observacao"]),
		Signals:           stringSliceFromAIValue(classification["sinais"]),
		Output:            result.Output,
	}
}

func stringFromAIValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(typed, 'f', 2, 64), "0"), ".")
	case int:
		return strconv.Itoa(typed)
	default:
		return ""
	}
}

func stringSliceFromAIValue(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if single := stringFromAIValue(value); single != "" {
			return []string{single}
		}
		return []string{}
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := stringFromAIValue(item); text != "" {
			out = append(out, text)
		}
	}
	return out
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
	if payload.IPICST != "" {
		resp.Suggestion.IPICST = payload.IPICST
	}
	if payload.IPIRate != "" {
		resp.Suggestion.IPIRate = payload.IPIRate
	}
	if payload.IPICEnq != "" {
		resp.Suggestion.IPICEnq = payload.IPICEnq
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

func applySubstitutionTaxEvidence(
	req SuggestRequest,
	op *fiscaloperations.FiscalOperation,
	resp *SuggestResponse,
) {
	if resp == nil || op == nil {
		return
	}

	if !isOutboundDirection(op.Direction) {
		return
	}

	sourceCFOP := onlyDigits(req.SourceCFOP)
	if isOutputSubstitutionTaxCFOP(sourceCFOP) {
		resp.Suggestion.CFOP = sourceCFOP
		resp.SelectedOperation.CFOP = sourceCFOP
	}

	if !hasSubstitutionTaxEvidence(req, resp) {
		return
	}

	if strings.TrimSpace(resp.Suggestion.ICMSCST) == "" && strings.TrimSpace(req.SourceICMSCST) == "60" {
		resp.Suggestion.ICMSCST = "60"
	}

	if strings.TrimSpace(resp.Suggestion.CFOP) == "" || isGenericSaleCFOP(resp.Suggestion.CFOP) {
		if cfop := suggestedSubstitutionTaxCFOP(req); cfop != "" {
			resp.Suggestion.CFOP = cfop
			resp.SelectedOperation.CFOP = cfop
		}
	}
}

func applyRetailDefaultRules(
	req SuggestRequest,
	op *fiscaloperations.FiscalOperation,
	resp *SuggestResponse,
) {
	if resp == nil || op == nil || !isOutboundDirection(op.Direction) {
		return
	}

	isST := hasStrongSubstitutionTaxEvidence(req, resp)
	sourceICMSCST := onlyDigits(req.SourceICMSCST)

	if isSimpleRetailRegime(req) {
		resp.Suggestion.ICMSCST = ""
		if strings.TrimSpace(resp.Suggestion.CSOSN) == "" {
			if isST {
				resp.Suggestion.CSOSN = "500"
			} else {
				resp.Suggestion.CSOSN = "102"
			}
		}
	} else {
		resp.Suggestion.CSOSN = ""
		if strings.TrimSpace(resp.Suggestion.ICMSCST) == "" {
			switch {
			case isST:
				resp.Suggestion.ICMSCST = "60"
			case sourceICMSCST == "20":
				resp.Suggestion.ICMSCST = "20"
			case sourceICMSCST == "40" || sourceICMSCST == "41":
				resp.Suggestion.ICMSCST = sourceICMSCST
			default:
				resp.Suggestion.ICMSCST = "00"
			}
		}
	}

	if strings.TrimSpace(resp.Suggestion.CFOP) == "" || (isST && !isOutputSubstitutionTaxCFOP(resp.Suggestion.CFOP)) {
		if isST {
			resp.Suggestion.CFOP = suggestedSubstitutionTaxCFOP(req)
		} else if strings.TrimSpace(resp.Suggestion.CFOP) == "" {
			resp.Suggestion.CFOP = defaultRetailSaleCFOP(req)
		}
		resp.SelectedOperation.CFOP = resp.Suggestion.CFOP
	}

	applyRetailContributionDefaults(req, resp, isST)
	applyRetailContributionRates(req, resp)
	adjustRetailDefaultConfidence(req, resp, isST)
	appendRetailDefaultLegalBasis(resp, req, isST)
}

func applyRetailContributionDefaults(req SuggestRequest, resp *SuggestResponse, isST bool) {
	if resp == nil {
		return
	}

	if strings.TrimSpace(resp.Suggestion.PISCST) == "" {
		resp.Suggestion.PISCST = defaultRetailContributionCST(req, isST)
	}
	if strings.TrimSpace(resp.Suggestion.COFINSCST) == "" {
		resp.Suggestion.COFINSCST = defaultRetailContributionCST(req, isST)
	}
}

func defaultRetailContributionCST(req SuggestRequest, isST bool) string {
	if isSimpleRetailRegime(req) {
		return "99"
	}
	return "01"
}

func applyRetailContributionRates(req SuggestRequest, resp *SuggestResponse) {
	if resp == nil || isSimpleRetailRegime(req) {
		return
	}

	pisRate, cofinsRate := standardContributionRates(req)
	if strings.TrimSpace(resp.Suggestion.PISCST) == "01" && isEmptyOrZero(resp.Suggestion.PISRate) {
		resp.Suggestion.PISRate = pisRate
	}
	if strings.TrimSpace(resp.Suggestion.COFINSCST) == "01" && isEmptyOrZero(resp.Suggestion.COFINSRate) {
		resp.Suggestion.COFINSRate = cofinsRate
	}
}

func adjustRetailDefaultConfidence(req SuggestRequest, resp *SuggestResponse, isST bool) {
	if resp == nil {
		return
	}

	target := 0.70
	sourceICMSCST := onlyDigits(req.SourceICMSCST)
	switch {
	case isST && strings.TrimSpace(resp.Suggestion.CEST) != "":
		target = 0.85
	case isST:
		target = 0.78
	case sourceICMSCST == "00" || onlyDigits(req.SourceCFOP) == "1102" || onlyDigits(req.SourceCFOP) == "2102":
		target = 0.75
	case sourceICMSCST == "20":
		target = 0.65
	case sourceICMSCST == "40" || sourceICMSCST == "41":
		target = 0.55
	}

	if resp.ConfidenceScore < target {
		resp.ConfidenceScore = target
	}
}

func applyFederalContributionBenefitReduction2026(req SuggestRequest, resp *SuggestResponse) {
	if resp == nil {
		return
	}

	pisCST := onlyDigits(resp.Suggestion.PISCST)
	cofinsCST := onlyDigits(resp.Suggestion.COFINSCST)
	if pisCST != "06" && cofinsCST != "06" {
		return
	}

	pisRate, cofinsRate := standardContributionRates(req)
	reducedPISRate := tenPercentRate(pisRate)
	reducedCOFINSRate := tenPercentRate(cofinsRate)

	if pisCST == "06" && isEmptyOrZero(resp.Suggestion.PISRate) {
		resp.Suggestion.PISRate = reducedPISRate
	}
	if cofinsCST == "06" && isEmptyOrZero(resp.Suggestion.COFINSRate) {
		resp.Suggestion.COFINSRate = reducedCOFINSRate
	}

	resp.LegalBasis = append(resp.LegalBasis, LegalBasisItem{
		TaxType:       "PIS_COFINS",
		Title:         "Reducao linear de beneficio fiscal federal",
		ReferenceCode: "LC 224/2025",
		Jurisdiction:  "FEDERAL",
		AppliedReason: "CST 06 indica aliquota zero; a partir de 01/04/2026, a plataforma aplica 10% da aliquota padrao quando nao ha aliquota especifica cadastrada.",
		Weight:        "statutory_adjustment",
	})
}

func standardContributionRates(req SuggestRequest) (string, string) {
	regime := strings.ToLower(strings.TrimSpace(req.TaxRegime))
	if strings.Contains(regime, "real") || strings.Contains(regime, "nao cumul") || strings.Contains(regime, "não cumul") {
		return "1.6500", "7.6000"
	}
	return "0.6500", "3.0000"
}

func tenPercentRate(rate string) string {
	switch strings.TrimSpace(rate) {
	case "1.6500":
		return "0.1650"
	case "7.6000":
		return "0.7600"
	case "0.6500":
		return "0.0650"
	case "3.0000":
		return "0.3000"
	default:
		return ""
	}
}

func isEmptyOrZero(value string) bool {
	normalized := strings.TrimSpace(value)
	normalized = strings.TrimSuffix(normalized, "%")
	normalized = strings.ReplaceAll(normalized, ",", ".")
	return normalized == "" || normalized == "0" || normalized == "0.0" || normalized == "0.00" || normalized == "0.0000"
}

func defaultRetailSaleCFOP(req SuggestRequest) string {
	emitterUF := strings.ToUpper(strings.TrimSpace(req.EmitterUF))
	recipientUF := strings.ToUpper(strings.TrimSpace(req.RecipientUF))
	if emitterUF != "" && recipientUF != "" && emitterUF != recipientUF {
		return "6102"
	}
	return "5102"
}

func normalizeCFOPByOperationContext(
	req SuggestRequest,
	op *fiscaloperations.FiscalOperation,
	resp *SuggestResponse,
) {
	if resp == nil || op == nil {
		return
	}

	cfop := onlyDigits(resp.Suggestion.CFOP)
	if len(cfop) != 4 {
		return
	}

	direction := normalizeFiscalOperationDirection(op.Direction)
	expectedPrefix := expectedCFOPPrefix(direction, req.EmitterUF, req.RecipientUF)
	if expectedPrefix == 0 || cfop[0] == expectedPrefix {
		return
	}

	if isOutboundDirection(op.Direction) {
		if hasStrongSubstitutionTaxEvidence(req, resp) || isOutputSubstitutionTaxCFOP(cfop) {
			resp.Suggestion.CFOP = suggestedSubstitutionTaxCFOP(req)
		} else {
			resp.Suggestion.CFOP = defaultRetailSaleCFOP(req)
		}
		resp.SelectedOperation.CFOP = resp.Suggestion.CFOP
		return
	}

	if direction == "entrada" {
		resp.Suggestion.CFOP = defaultEntryCFOP(req)
		resp.SelectedOperation.CFOP = resp.Suggestion.CFOP
	}
}

func normalizeFiscalOperationDirection(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(value, "entr"), strings.Contains(value, "inbound"):
		return "entrada"
	case strings.Contains(value, "saida"), strings.Contains(value, "saída"), strings.Contains(value, "outbound"), strings.Contains(value, "exit"):
		return "saida"
	default:
		return ""
	}
}

func expectedCFOPPrefix(direction string, emitterUF string, recipientUF string) byte {
	emitterUF = strings.ToUpper(strings.TrimSpace(emitterUF))
	recipientUF = strings.ToUpper(strings.TrimSpace(recipientUF))
	sameUF := emitterUF == "" || recipientUF == "" || emitterUF == recipientUF

	switch direction {
	case "entrada":
		if sameUF {
			return '1'
		}
		return '2'
	case "saida":
		if sameUF {
			return '5'
		}
		return '6'
	default:
		return 0
	}
}

func defaultEntryCFOP(req SuggestRequest) string {
	emitterUF := strings.ToUpper(strings.TrimSpace(req.EmitterUF))
	recipientUF := strings.ToUpper(strings.TrimSpace(req.RecipientUF))
	if emitterUF != "" && recipientUF != "" && emitterUF != recipientUF {
		return "2102"
	}
	return "1102"
}

func isSimpleRetailRegime(req SuggestRequest) bool {
	regime := strings.ToLower(strings.TrimSpace(req.TaxRegime))
	crt := onlyDigits(req.TargetCRT)
	return crt == "1" || strings.Contains(regime, "simples")
}

func appendRetailDefaultLegalBasis(resp *SuggestResponse, req SuggestRequest, isST bool) {
	if resp == nil {
		return
	}

	reason := "Default operacional de varejo aplicado quando nao ha regra especifica cadastrada."
	if isST {
		reason = "Default operacional de varejo com evidencia de substituicao tributaria aplicado quando nao ha regra especifica cadastrada."
	}

	resp.LegalBasis = append(resp.LegalBasis, LegalBasisItem{
		TaxType:       "RETAIL_DEFAULT",
		Title:         "Perfil padrao varejista",
		ReferenceCode: "DEFAULT_RETAIL",
		Jurisdiction:  "OPERATIONAL",
		UF:            strings.ToUpper(strings.TrimSpace(req.RecipientUF)),
		AppliedReason: reason,
		Weight:        "fallback",
	})
}

func (s *Service) applyNCMTaxProfiles(
	ctx context.Context,
	req SuggestRequest,
	op *fiscaloperations.FiscalOperation,
	resp *SuggestResponse,
) {
	if s == nil || s.repo == nil || resp == nil || op == nil {
		return
	}

	ncm := onlyDigits(firstNonEmpty(resp.Suggestion.NCM, req.NCMCode))
	if ncm == "" {
		return
	}

	profiles, err := s.repo.FindNCMTaxProfiles(
		ctx,
		ncm,
		firstNonEmpty(req.RecipientUF, req.EmitterUF),
		op.Code,
		req.TaxRegime,
		req.TargetCRT,
	)
	if err != nil || len(profiles) == 0 {
		return
	}

	for _, profile := range profiles {
		applyNCMTaxProfile(req, resp, profile)
		appendNCMTaxProfileLegalBasis(resp, profile)
		if resp.ConfidenceScore < profile.ConfidenceScore {
			resp.ConfidenceScore = profile.ConfidenceScore
		}
	}
}

func applyNCMTaxProfile(req SuggestRequest, resp *SuggestResponse, profile NCMTaxProfile) {
	if resp == nil {
		return
	}

	if strings.TrimSpace(profile.CEST) != "" {
		resp.Suggestion.CEST = profile.CEST
	}
	if strings.TrimSpace(profile.CFOP) != "" {
		resp.Suggestion.CFOP = profile.CFOP
		resp.SelectedOperation.CFOP = profile.CFOP
	}
	if strings.TrimSpace(profile.CClasTrib) != "" {
		resp.Suggestion.CClasTrib = profile.CClasTrib
	}

	switch strings.TrimSpace(profile.TaxType) {
	case "ICMS", "ICMS_ST":
		applyNCMTaxProfileICMS(req, resp, profile)
	case "PIS_COFINS":
		applyNCMTaxProfileContributions(resp, profile)
	case "IPI":
		applyNCMTaxProfileIPI(resp, profile)
	case "IBS_CBS":
		applyNCMTaxProfileReform(resp, profile)
	case "SELECTIVE_TAX":
		applyNCMTaxProfileSelectiveTax(resp, profile)
	}
}

func applyNCMTaxProfileICMS(req SuggestRequest, resp *SuggestResponse, profile NCMTaxProfile) {
	if isSimpleRetailRegime(req) {
		resp.Suggestion.ICMSCST = ""
		if strings.TrimSpace(profile.CSOSN) != "" {
			resp.Suggestion.CSOSN = profile.CSOSN
		}
	} else {
		resp.Suggestion.CSOSN = ""
		if strings.TrimSpace(profile.ICMSCST) != "" {
			resp.Suggestion.ICMSCST = profile.ICMSCST
		}
	}

	if strings.TrimSpace(profile.ICMSRate) != "" {
		resp.Suggestion.ICMSRate = profile.ICMSRate
	}
	if strings.TrimSpace(profile.FCPRate) != "" {
		resp.Suggestion.FCPRate = profile.FCPRate
	}
	if strings.TrimSpace(profile.ICMSSTRate) != "" {
		resp.Suggestion.ICMSSTRate = profile.ICMSSTRate
	}
}

func applyNCMTaxProfileContributions(resp *SuggestResponse, profile NCMTaxProfile) {
	if strings.TrimSpace(profile.PISCST) != "" {
		resp.Suggestion.PISCST = profile.PISCST
	}
	if strings.TrimSpace(profile.COFINSCST) != "" {
		resp.Suggestion.COFINSCST = profile.COFINSCST
	}
	if strings.TrimSpace(profile.PISRate) != "" {
		resp.Suggestion.PISRate = profile.PISRate
	}
	if strings.TrimSpace(profile.COFINSRate) != "" {
		resp.Suggestion.COFINSRate = profile.COFINSRate
	}
	if strings.TrimSpace(profile.PISRevenueCode) != "" {
		resp.Suggestion.PISRevenueCode = profile.PISRevenueCode
	}
	if strings.TrimSpace(profile.COFINSRevenueCode) != "" {
		resp.Suggestion.COFINSRevenueCode = profile.COFINSRevenueCode
	}
}

func applyNCMTaxProfileIPI(resp *SuggestResponse, profile NCMTaxProfile) {
	if strings.TrimSpace(profile.IPICST) != "" {
		resp.Suggestion.IPICST = profile.IPICST
	}
	if strings.TrimSpace(profile.IPIRate) != "" {
		resp.Suggestion.IPIRate = profile.IPIRate
	}
	if strings.TrimSpace(profile.IPICEnq) != "" {
		resp.Suggestion.IPICEnq = profile.IPICEnq
	}
}

func applyNCMTaxProfileReform(resp *SuggestResponse, profile NCMTaxProfile) {
	if strings.TrimSpace(profile.CClasTrib) != "" {
		resp.Suggestion.CClasTrib = profile.CClasTrib
	}
	if strings.TrimSpace(profile.IBSRate) != "" {
		resp.Suggestion.IBSRate = profile.IBSRate
	}
	if strings.TrimSpace(profile.CBSRate) != "" {
		resp.Suggestion.CBSRate = profile.CBSRate
	}
}

func applyNCMTaxProfileSelectiveTax(resp *SuggestResponse, profile NCMTaxProfile) {
	if strings.TrimSpace(profile.SelectiveTaxCode) != "" {
		resp.Suggestion.SelectiveTaxCode = profile.SelectiveTaxCode
	}
	if strings.TrimSpace(profile.SelectiveTaxRate) != "" {
		resp.Suggestion.SelectiveTaxRate = profile.SelectiveTaxRate
	}
}

func appendNCMTaxProfileLegalBasis(resp *SuggestResponse, profile NCMTaxProfile) {
	if resp == nil {
		return
	}

	title := "Perfil tributario por NCM"
	if strings.TrimSpace(profile.TaxGroup) != "" {
		title = "Perfil NCM " + strings.TrimSpace(profile.TaxGroup)
	}

	taxType := profile.TaxType
	if strings.TrimSpace(profile.TaxType) == "PIS_COFINS" && strings.Contains(strings.ToLower(profile.TaxGroup), "monofas") {
		taxType = "PIS_COFINS_MONOPHASIC"
	}

	resp.LegalBasis = append(resp.LegalBasis, LegalBasisItem{
		TaxType:       taxType,
		Title:         title,
		ReferenceCode: firstNonEmpty(profile.SourceReference, "NCM_TAX_PROFILE_"+profile.NCMPattern),
		Jurisdiction:  jurisdictionFromNCMTaxProfile(profile),
		UF:            strings.ToUpper(strings.TrimSpace(profile.UF)),
		AppliedReason: firstNonEmpty(profile.Notes, "Regra aplicada a partir da matriz NCM x tributo cadastrada na plataforma."),
		Weight:        "ncm_profile",
	})
}

func jurisdictionFromNCMTaxProfile(profile NCMTaxProfile) string {
	if strings.TrimSpace(profile.UF) != "" {
		return "STATE"
	}
	switch strings.TrimSpace(profile.TaxType) {
	case "ICMS", "ICMS_ST":
		return "STATE"
	default:
		return "FEDERAL"
	}
}

func (s *Service) applyStateICMSRule(
	ctx context.Context,
	req SuggestRequest,
	op *fiscaloperations.FiscalOperation,
	resp *SuggestResponse,
) {
	if s == nil || s.repo == nil || resp == nil || op == nil || !isOutboundDirection(op.Direction) {
		return
	}

	ncm := onlyDigits(firstNonEmpty(resp.Suggestion.NCM, req.NCMCode))
	uf := strings.ToUpper(strings.TrimSpace(firstNonEmpty(req.RecipientUF, req.EmitterUF)))
	if ncm == "" || uf == "" {
		return
	}

	rule, err := s.repo.FindStateICMSRule(
		ctx,
		ncm,
		resp.Suggestion.CEST,
		uf,
		op.Code,
		req.TaxRegime,
		req.TargetCRT,
	)
	if err != nil || rule == nil {
		return
	}

	if hasStrongSubstitutionTaxEvidence(req, resp) && !isStateICMSRuleST(*rule) && isGenericStateICMSRule(*rule) {
		return
	}

	applyStateICMSRuleSuggestion(req, resp, *rule)
	appendStateICMSRuleLegalBasis(resp, *rule)
	if resp.ConfidenceScore < rule.ConfidenceScore {
		resp.ConfidenceScore = rule.ConfidenceScore
	}
}

func applyStateICMSRuleSuggestion(req SuggestRequest, resp *SuggestResponse, rule StateICMSRule) {
	if resp == nil {
		return
	}

	if strings.TrimSpace(rule.CEST) != "" && strings.TrimSpace(resp.Suggestion.CEST) == "" {
		resp.Suggestion.CEST = rule.CEST
	}

	cfop := strings.TrimSpace(rule.CFOP)
	if cfop == "" {
		if isStateICMSRuleST(rule) {
			cfop = suggestedSubstitutionTaxCFOP(req)
		} else if strings.TrimSpace(resp.Suggestion.CFOP) == "" || isGenericSaleCFOP(resp.Suggestion.CFOP) {
			cfop = defaultRetailSaleCFOP(req)
		}
	}
	if cfop != "" {
		resp.Suggestion.CFOP = cfop
		resp.SelectedOperation.CFOP = cfop
	}

	if isSimpleRetailRegime(req) {
		resp.Suggestion.ICMSCST = ""
		switch {
		case strings.TrimSpace(rule.CSOSN) != "":
			resp.Suggestion.CSOSN = rule.CSOSN
		case isStateICMSRuleST(rule):
			resp.Suggestion.CSOSN = "500"
		case strings.TrimSpace(resp.Suggestion.CSOSN) == "":
			resp.Suggestion.CSOSN = "102"
		}
	} else {
		resp.Suggestion.CSOSN = ""
		switch {
		case strings.TrimSpace(rule.ICMSCST) != "":
			resp.Suggestion.ICMSCST = rule.ICMSCST
		case isStateICMSRuleST(rule):
			resp.Suggestion.ICMSCST = "60"
		case strings.TrimSpace(resp.Suggestion.ICMSCST) == "":
			resp.Suggestion.ICMSCST = "00"
		}
	}

	if strings.TrimSpace(rule.ICMSRate) != "" {
		resp.Suggestion.ICMSRate = rule.ICMSRate
	}
	if strings.TrimSpace(rule.FCPRate) != "" {
		resp.Suggestion.FCPRate = rule.FCPRate
	}
	if strings.TrimSpace(rule.ICMSSTRate) != "" {
		resp.Suggestion.ICMSSTRate = rule.ICMSSTRate
	}
	if strings.TrimSpace(rule.ICMSBaseReduction) != "" {
		resp.Suggestion.ICMSBaseReduction = rule.ICMSBaseReduction
	}
	if strings.TrimSpace(rule.CBenef) != "" {
		resp.Suggestion.CBenef = rule.CBenef
	}
}

func appendStateICMSRuleLegalBasis(resp *SuggestResponse, rule StateICMSRule) {
	if resp == nil {
		return
	}

	taxType := "ICMS_STATE"
	title := "Regra estadual de ICMS"
	if isStateICMSRuleST(rule) {
		taxType = "ICMS_ST"
		title = "Regra estadual de ICMS-ST"
	}

	resp.LegalBasis = append(resp.LegalBasis, LegalBasisItem{
		TaxType:       taxType,
		Title:         title,
		ReferenceCode: firstNonEmpty(rule.SourceReference, "STATE_ICMS_"+rule.UF+"_"+rule.NCMPattern),
		Jurisdiction:  "STATE",
		UF:            strings.ToUpper(strings.TrimSpace(rule.UF)),
		AppliedReason: firstNonEmpty(rule.Notes, "Regra estadual aplicada conforme UF da organizacao e identidade fiscal do item."),
		Weight:        stateICMSRuleWeight(rule),
	})
}

func isStateICMSRuleST(rule StateICMSRule) bool {
	return strings.EqualFold(strings.TrimSpace(rule.RuleKind), "ST")
}

func stateICMSRuleWeight(rule StateICMSRule) string {
	if strings.TrimSpace(rule.NCMPattern) == "" {
		return "state_icms_rule_generic"
	}
	if strings.EqualFold(strings.TrimSpace(rule.MatchType), "exact") {
		return "state_icms_rule_exact"
	}
	return "state_icms_rule_prefix"
}

func isGenericStateICMSRule(rule StateICMSRule) bool {
	return strings.TrimSpace(rule.NCMPattern) == "" && !isStateICMSRuleST(rule)
}

func applyRetailSpecialOutputRules(req SuggestRequest, resp *SuggestResponse) {
	if resp == nil {
		return
	}

	if applyRetailMonophasicRule(req, resp) {
		return
	}
}

func applyRetailMonophasicRule(req SuggestRequest, resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}

	ncm := onlyDigits(firstNonEmpty(resp.Suggestion.NCM, req.NCMCode))
	rule, ok := findRetailMonophaseNCMRule(ncm)
	if !ok {
		return false
	}

	if onlyDigits(req.SourcePISCST) != "02" && onlyDigits(req.SourceCOFINSCST) != "02" &&
		onlyDigits(resp.Suggestion.PISCST) != "02" && onlyDigits(resp.Suggestion.COFINSCST) != "02" {
		return false
	}

	resp.Suggestion.PISCST = "04"
	resp.Suggestion.COFINSCST = "04"
	resp.Suggestion.PISRate = "0.0000"
	resp.Suggestion.COFINSRate = "0.0000"
	resp.Suggestion.PISValue = ""
	resp.Suggestion.COFINSValue = ""

	if hasStrongSubstitutionTaxEvidence(req, resp) {
		resp.Suggestion.CFOP = suggestedSubstitutionTaxCFOP(req)
		resp.SelectedOperation.CFOP = resp.Suggestion.CFOP
		resp.Suggestion.ICMSRate = firstNonEmpty(resp.Suggestion.ICMSRate, "0.0000")
		if isSimpleRetailRegime(req) {
			resp.Suggestion.ICMSCST = ""
			resp.Suggestion.CSOSN = "500"
		} else {
			resp.Suggestion.CSOSN = ""
			resp.Suggestion.ICMSCST = "60"
		}
	}

	resp.LegalBasis = append(resp.LegalBasis, LegalBasisItem{
		TaxType:       "PIS_COFINS_MONOPHASIC",
		Title:         "Produto em regime monofasico no varejo",
		ReferenceCode: "RETAIL_MONOPHASIC_" + strings.ToUpper(rule.Category),
		Jurisdiction:  "FEDERAL",
		AppliedReason: "NCM de " + rule.Description + " com CST base 02 indica tributacao concentrada na cadeia anterior. Para venda varejista, a plataforma sugere PIS/COFINS CST 04 com aliquota zero.",
		Weight:        "0.92",
	})

	targetConfidence := 0.82
	if hasStrongSubstitutionTaxEvidence(req, resp) && strings.TrimSpace(resp.Suggestion.CEST) != "" {
		targetConfidence = 0.92
	}
	if resp.ConfidenceScore < targetConfidence {
		resp.ConfidenceScore = targetConfidence
	}

	return true
}

func findRetailMonophaseNCMRule(ncm string) (monophaseNCMRule, bool) {
	ncm = onlyDigits(ncm)
	for _, rule := range retailMonophaseNCMRules {
		if strings.HasPrefix(ncm, rule.Prefix) {
			return rule, true
		}
	}
	return monophaseNCMRule{}, false
}

func applyReferenceProductFiscalRules(
	req SuggestRequest,
	op *fiscaloperations.FiscalOperation,
	resp *SuggestResponse,
) {
	if resp == nil || op == nil || !isOutboundDirection(op.Direction) {
		return
	}

	ncm := onlyDigits(firstNonEmpty(resp.Suggestion.NCM, req.NCMCode))
	if ncm != "22029100" {
		return
	}

	emitterUF := strings.ToUpper(strings.TrimSpace(req.EmitterUF))
	recipientUF := strings.ToUpper(strings.TrimSpace(req.RecipientUF))
	if emitterUF != "GO" || recipientUF != "GO" {
		return
	}

	regime := strings.ToLower(strings.TrimSpace(req.TaxRegime))
	if regime != "" && strings.Contains(regime, "simples") {
		return
	}

	resp.Suggestion.NCM = "22029100"
	resp.Suggestion.CEST = firstNonEmpty(resp.Suggestion.CEST, "0302201")
	resp.Suggestion.CFOP = "5405"
	resp.SelectedOperation.CFOP = "5405"
	resp.Suggestion.ICMSCST = "60"
	resp.Suggestion.ICMSRate = "21.0000"
	resp.Suggestion.PISCST = "02"
	resp.Suggestion.COFINSCST = "02"
	resp.Suggestion.PISRate = "1.8600"
	resp.Suggestion.COFINSRate = "8.5400"
	resp.Suggestion.PISRevenueCode = "419"
	resp.Suggestion.COFINSRevenueCode = "419"
	resp.Suggestion.IPICST = "50"
	resp.Suggestion.IPIRate = "3.9000"
	resp.Suggestion.IPICEnq = "999"
	resp.Suggestion.CClasTrib = firstNonEmpty(resp.Suggestion.CClasTrib, "000001")
	resp.Suggestion.IBSRate = firstNonEmpty(resp.Suggestion.IBSRate, "0.1000")
	resp.Suggestion.CBSRate = firstNonEmpty(resp.Suggestion.CBSRate, "0.9000")
	resp.Suggestion.SelectiveTaxCode = firstNonEmpty(resp.Suggestion.SelectiveTaxCode, "2")

	if resp.ConfidenceScore < 0.88 {
		resp.ConfidenceScore = 0.88
	}

	resp.LegalBasis = append(resp.LegalBasis,
		LegalBasisItem{
			TaxType:       "ICMS_ST",
			Title:         "Bebidas NCM 22029100 em GO",
			ReferenceCode: "RCTE/GO art. 34 Ap. II inc. I An. VIII",
			Jurisdiction:  "STATE",
			UF:            "GO",
			AppliedReason: "Produto enquadrado como bebida sujeita a substituicao tributaria em operacao interna; CFOP 5405 e CST 60.",
			Weight:        "reference_case",
		},
		LegalBasisItem{
			TaxType:       "PIS_COFINS",
			Title:         "Bebidas frias - aliquota diferenciada",
			ReferenceCode: "Lei 13.097/2015 art. 25; Decreto 8.442/2015 art. 20",
			Jurisdiction:  "FEDERAL",
			AppliedReason: "Saida para varejista ou consumidor final com CST 02, PIS 1,86%, COFINS 8,54% e codigo de natureza da receita 419.",
			Weight:        "reference_case",
		},
		LegalBasisItem{
			TaxType:       "IPI",
			Title:         "TIPI bebida NCM 22029100",
			ReferenceCode: "RIPI/2010 art. 190; Decreto 11.158/2022",
			Jurisdiction:  "FEDERAL",
			AppliedReason: "Saida tributada com CST 50, CEnq 999 e aliquota de IPI de 3,90% para o caso de referencia.",
			Weight:        "reference_case",
		},
		LegalBasisItem{
			TaxType:       "IBS_CBS",
			Title:         "Regra transitoria IBS/CBS 2026",
			ReferenceCode: "LC 214/2025 arts. 343, 344 e 346",
			Jurisdiction:  "FEDERAL",
			AppliedReason: "CST 000 e cClasTrib 000001 para situacao tributada integralmente no periodo de teste 2026.",
			Weight:        "reference_case",
		},
	)
}

func hasSubstitutionTaxEvidence(req SuggestRequest, resp *SuggestResponse) bool {
	return hasStrongSubstitutionTaxEvidence(req, resp)
}

func hasStrongSubstitutionTaxEvidence(req SuggestRequest, resp *SuggestResponse) bool {
	if isSubstitutionTaxCST(req.SourceICMSCST) {
		return true
	}
	if isSubstitutionTaxCSOSN(req.SourceICMSCSOSN) {
		return true
	}
	if resp != nil && isSubstitutionTaxCST(resp.Suggestion.ICMSCST) {
		return true
	}
	if resp != nil && isSubstitutionTaxCSOSN(resp.Suggestion.CSOSN) {
		return true
	}
	if isSubstitutionTaxCFOP(req.SourceCFOP) {
		return true
	}
	if isEntrySubstitutionTaxCFOP(req.SourceCFOP) {
		return true
	}
	return resp != nil && isSubstitutionTaxCFOP(resp.Suggestion.CFOP)
}

func isSubstitutionTaxCST(value string) bool {
	switch onlyDigits(value) {
	case "10", "30", "60", "70":
		return true
	default:
		return false
	}
}

func isSubstitutionTaxCSOSN(value string) bool {
	switch onlyDigits(value) {
	case "201", "202", "203", "500":
		return true
	default:
		return false
	}
}

func suggestedSubstitutionTaxCFOP(req SuggestRequest) string {
	emitterUF := strings.ToUpper(strings.TrimSpace(req.EmitterUF))
	recipientUF := strings.ToUpper(strings.TrimSpace(req.RecipientUF))

	if emitterUF == "" || recipientUF == "" || emitterUF == recipientUF {
		return "5405"
	}

	return "6404"
}

func isGenericSaleCFOP(value string) bool {
	switch onlyDigits(value) {
	case "", "5101", "5102", "6101", "6102":
		return true
	default:
		return false
	}
}

func isSubstitutionTaxCFOP(value string) bool {
	return isOutputSubstitutionTaxCFOP(value) || isEntrySubstitutionTaxCFOP(value)
}

func isOutputSubstitutionTaxCFOP(value string) bool {
	switch onlyDigits(value) {
	case "5403", "5405", "6403", "6404":
		return true
	default:
		return false
	}
}

func isEntrySubstitutionTaxCFOP(value string) bool {
	switch onlyDigits(value) {
	case "1401", "1403", "1406", "1407", "2401", "2403", "2406", "2407":
		return true
	default:
		return false
	}
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
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

	if !isOutboundDirection(op.Direction) {
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

func normalizeDigits(value string) string {
	var builder strings.Builder
	for _, char := range strings.TrimSpace(value) {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
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

func normalizeRegimeSpecificDefaults(req SuggestRequest, resp *SuggestResponse) {
	if resp == nil {
		return
	}

	if isSimpleRetailRegime(req) {
		resp.Suggestion.ICMSCST = ""
		if strings.TrimSpace(resp.Suggestion.CSOSN) == "" {
			if isSubstitutionTaxCFOP(resp.Suggestion.CFOP) {
				resp.Suggestion.CSOSN = "500"
			} else {
				resp.Suggestion.CSOSN = "102"
			}
		}
		return
	}

	resp.Suggestion.CSOSN = ""
	if strings.TrimSpace(resp.Suggestion.ICMSCST) == "" {
		if isSubstitutionTaxCFOP(resp.Suggestion.CFOP) {
			resp.Suggestion.ICMSCST = "60"
		} else {
			resp.Suggestion.ICMSCST = "00"
		}
	}
}

func collectSuggestWarnings(req SuggestRequest, item *TaxMatch, resp *SuggestResponse) []string {
	warnings := make([]string, 0, 4)

	if item == nil || resp == nil {
		return warnings
	}

	switch strings.TrimSpace(resp.MatchType) {
	case "context_fallback":
		warnings = append(warnings, "A sugestao foi montada sem historico de produto; o perfil padrao de varejo foi usado como apoio inicial.")
	case "ncm_catalog":
		warnings = append(warnings, "A sugestao foi baseada principalmente no catalogo NCM. Revise CSTs e aliquotas antes de usar em producao.")
	case "ncm_profile":
		warnings = append(warnings, "A sugestao reaproveitou um produto cadastrado com o mesmo NCM na memoria fiscal da organizacao.")
	case "entry_memory":
		warnings = append(warnings, "A sugestao reaproveitou a identidade fiscal cadastrada do item. A saida foi calculada pelo contexto da organizacao.")
	case "cosmos_search":
		warnings = append(warnings, "Identidade fiscal obtida na Cosmos BlueSoft. A plataforma usa NCM/CEST para acionar regras internas, mas a tributacao deve continuar validada por UF, regime e fonte legal.")
	}

	if strings.TrimSpace(req.TaxRegime) == "" {
		warnings = append(warnings, "O regime tributario nao foi informado. A plataforma usa default de varejo normal, mas Simples Nacional pode exigir CSOSN 102/500.")
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

	if warning := buildCFOPContextWarning(req, resp); warning != "" {
		warnings = append(warnings, warning)
	}

	if resp.CESTReference != nil && !hasConfirmedSubstitutionTax(resp) {
		warnings = append(warnings, "CEST encontrado para o NCM informado, mas isso nao confirma substituicao tributaria. Valide a regra estadual, CST/CSOSN, CFOP, MVA, FCP ST e excecoes da UF antes de concluir.")
	}

	if hasDistinctContext(item.TargetCRT, req.TargetCRT) {
		warnings = append(warnings, "A memoria fiscal encontrada foi registrada para outro CRT. Revise CST de ICMS, CSOSN e contribuicoes antes de usar.")
	}

	if strings.TrimSpace(req.SourcePISCST) != "" && strings.TrimSpace(item.PISCST) != "" && !strings.EqualFold(strings.TrimSpace(req.SourcePISCST), strings.TrimSpace(item.PISCST)) {
		warnings = append(warnings, "O CST de PIS informado difere da memoria fiscal encontrada. A sugestao foi conservadora nas contribuicoes.")
	}

	if strings.TrimSpace(req.SourceCOFINSCST) != "" && strings.TrimSpace(item.COFINSCST) != "" && !strings.EqualFold(strings.TrimSpace(req.SourceCOFINSCST), strings.TrimSpace(item.COFINSCST)) {
		warnings = append(warnings, "O CST de COFINS informado difere da memoria fiscal encontrada. Revise a saida antes de confirmar.")
	}

	if contributionCSTNeedsRevenueNature(resp) && !hasContributionRevenueCode(resp) {
		warnings = append(warnings, "PIS/COFINS CST 04, 05 ou 06 exige natureza da receita para validar monofasico, substituicao tributaria ou aliquota zero. Cadastre o codigo correto antes de publicar a regra.")
	}

	if hasContributionBenefitReduction2026(resp) {
		warnings = append(warnings, "LC 224/2025: CST 06 de PIS/COFINS recebeu ajuste de 10% da aliquota padrao a partir de 01/04/2026. Confirme se o produto nao esta em excecao legal, como cesta basica ou anexo especifico.")
	}

	if hasMonophasicOrSubstitutionContributionCST(resp) {
		warnings = append(warnings, "CST 04/05 em PIS/COFINS depende de produto monofasico ou substituicao tributaria. A plataforma mantem o CST, mas exige natureza da receita e base legal antes de publicar como regra definitiva.")
	}

	if hasRetailDefault(resp) && !hasStateICMSLegalBasis(resp) {
		warnings = append(warnings, "Default varejista aplicado: confirme o produto, regime e natureza da receita antes de transformar a sugestao em regra permanente.")
	}

	if item := stateICMSLegalBasis(resp); item != nil && strings.TrimSpace(item.Weight) == "state_icms_rule_generic" {
		warnings = append(warnings, "ICMS calculado por regra estadual geral da UF. Para aumentar a confianca, cadastre regra especifica por NCM, CEST, beneficio ou ST.")
	}

	if strings.TrimSpace(item.OrganizationID) == "" {
		warnings = append(warnings, "A memoria fiscal reaproveitada nao esta vinculada a uma organizacao especifica. Priorize validacao manual antes de usar em producao.")
	}

	if strings.EqualFold(strings.TrimSpace(resp.Suggestion.DIFALMode), "INTERSTATE") && strings.TrimSpace(resp.Suggestion.DIFALDifferenceRate) != "" {
		warnings = append(warnings, "A leitura de DIFAL foi reforcada com a matriz de aliquotas por UF. Revise excecoes por produto importado, beneficio estadual e regra especifica da operacao.")
	}

	return warnings
}

func buildCFOPContextWarning(req SuggestRequest, resp *SuggestResponse) string {
	if resp == nil {
		return ""
	}

	cfop := onlyDigits(resp.Suggestion.CFOP)
	if len(cfop) != 4 {
		return ""
	}

	emitterUF := strings.ToUpper(strings.TrimSpace(req.EmitterUF))
	recipientUF := strings.ToUpper(strings.TrimSpace(req.RecipientUF))
	if emitterUF != "" && recipientUF != "" && emitterUF != recipientUF && cfop[0] == '5' {
		return "CFOP ajustado para operacao interestadual quando UF de origem e destino sao diferentes; confira se a venda deve usar grupo 6.xxx."
	}
	if (emitterUF == "" || recipientUF == "" || emitterUF == recipientUF) && cfop[0] == '6' {
		return "CFOP ajustado para operacao interna quando UF de origem e destino sao iguais; confira se a venda deve usar grupo 5.xxx."
	}
	return ""
}

func buildTaxDiagnostics(req SuggestRequest, resp *SuggestResponse) TaxDiagnostics {
	items := []TaxDiagnosticItem{
		buildClassificationDiagnostic(resp),
		buildOperationDiagnostic(resp),
		buildICMSDiagnostic(req, resp),
		buildContributionsDiagnostic(req, resp),
		buildReformDiagnostic(resp),
		buildLegalBasisDiagnostic(resp),
	}

	out := TaxDiagnostics{Items: items}
	for _, item := range items {
		switch item.Status {
		case "ready":
			out.Ready++
		case "missing":
			out.Missing++
		default:
			out.Attention++
		}
	}
	return out
}

func buildDecisionSummary(req SuggestRequest, resp *SuggestResponse) DecisionSummary {
	diagnostics := resp.Diagnostics
	blockingReasons := make([]string, 0)
	nextActions := make([]string, 0)

	for _, item := range diagnostics.Items {
		if item.Status == "missing" {
			blockingReasons = append(blockingReasons, item.Title)
			if strings.TrimSpace(item.Action) != "" {
				nextActions = append(nextActions, item.Action)
			}
			continue
		}
		if item.Status == "attention" && strings.TrimSpace(item.Action) != "" {
			nextActions = append(nextActions, item.Action)
		}
	}

	confidence := resp.ConfidenceScore
	hasLegalBasis := hasSpecificLegalBasis(resp)
	hasCriticalMissing := hasMissingArea(diagnostics, "classification") || hasMissingArea(diagnostics, "operation")
	hasTaxMissing := hasMissingArea(diagnostics, "icms") || hasMissingArea(diagnostics, "pis_cofins")

	summary := DecisionSummary{
		Status:               "MANUAL_REVIEW_REQUIRED",
		Severity:             "warning",
		Title:                "Sugestao fiscal exige revisao",
		Message:              "O motor encontrou dados uteis, mas ainda ha pendencias antes de usar como regra operacional.",
		CanUseOperationally:  false,
		RequiresManualReview: true,
		BlockingReasons:      dedupeNonEmpty(blockingReasons),
		NextActions:          dedupeNonEmpty(nextActions),
	}

	switch {
	case hasCriticalMissing:
		summary.Status = "BLOCKED_BY_MISSING_DATA"
		summary.Severity = "danger"
		summary.Title = "Consulta bloqueada por dados essenciais"
		summary.Message = "Faltam classificacao ou operacao fiscal para uma decisao tributaria segura."
	case hasTaxMissing:
		summary.Status = "MANUAL_REVIEW_REQUIRED"
		summary.Severity = "warning"
		summary.Title = "Tributacao incompleta"
		summary.Message = "Classificacao e operacao existem, mas ainda faltam blocos tributarios relevantes."
	case confidence >= 0.90 && hasLegalBasis && diagnostics.Missing == 0 && diagnostics.Attention == 0:
		summary.Status = "READY_FOR_OPERATION"
		summary.Severity = "success"
		summary.Title = "Decisao fiscal pronta"
		summary.Message = "A consulta possui classificacao, regra fiscal, base legal e boa confianca para uso operacional."
		summary.CanUseOperationally = true
		summary.RequiresManualReview = false
	case confidence >= 0.70 && diagnostics.Missing == 0:
		summary.Status = "SUGGESTED_WITH_WARNINGS"
		summary.Severity = "info"
		summary.Title = "Sugestao utilizavel com validacao"
		summary.Message = "A consulta esta bem encaminhada, mas ainda recomenda revisao fiscal antes de publicar regra permanente."
	case strings.TrimSpace(req.TaxRegime) == "":
		summary.Status = "MANUAL_REVIEW_REQUIRED"
		summary.Severity = "warning"
		summary.Title = "Regime tributario indefinido"
		summary.Message = "Informe o regime tributario para melhorar PIS/COFINS, ICMS e regras por CRT."
	}

	if len(summary.NextActions) == 0 && summary.RequiresManualReview {
		summary.NextActions = []string{"Revise a base fiscal do produto e vincule regra legal antes de publicar."}
	}

	return summary
}

func hasSpecificLegalBasis(resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}
	for _, item := range resp.LegalBasis {
		taxType := strings.TrimSpace(item.TaxType)
		if taxType != "RETAIL_DEFAULT" && taxType != "CEST" && taxType != "AI_ASSIST" && taxType != "PRODUCT_IDENTITY" {
			return true
		}
	}
	return false
}

func hasStateICMSLegalBasis(resp *SuggestResponse) bool {
	return stateICMSLegalBasis(resp) != nil
}

func stateICMSLegalBasis(resp *SuggestResponse) *LegalBasisItem {
	if resp == nil {
		return nil
	}
	for i := range resp.LegalBasis {
		item := &resp.LegalBasis[i]
		if strings.HasPrefix(strings.TrimSpace(item.Weight), "state_icms_rule") {
			return item
		}
	}
	return nil
}

func hasMissingArea(diagnostics TaxDiagnostics, area string) bool {
	for _, item := range diagnostics.Items {
		if item.Area == area && item.Status == "missing" {
			return true
		}
	}
	return false
}

func dedupeNonEmpty(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func buildClassificationDiagnostic(resp *SuggestResponse) TaxDiagnosticItem {
	s := resp.Suggestion
	if strings.TrimSpace(s.NCM) != "" && strings.TrimSpace(s.CEST) != "" {
		return TaxDiagnosticItem{
			Area:   "classification",
			Status: "ready",
			Title:  "Identidade fiscal localizada",
			Detail: "NCM e CEST estao disponiveis. CEST identifica o item para analise, mas nao confirma ST sem regra estadual aplicavel.",
			Action: "Valide UF, CFOP, CST/CSOSN e regra estadual antes de classificar como substituicao tributaria.",
		}
	}
	if strings.TrimSpace(s.NCM) != "" {
		return TaxDiagnosticItem{
			Area:   "classification",
			Status: "attention",
			Title:  "Classificacao parcial",
			Detail: "O NCM foi identificado, mas o CEST ainda nao foi confirmado para este produto.",
			Action: "Importe ou revise a tabela CEST quando o produto puder estar sujeito a ICMS ST.",
		}
	}
	return TaxDiagnosticItem{
		Area:   "classification",
		Status: "missing",
		Title:  "Classificacao ausente",
		Detail: "A consulta nao encontrou NCM suficiente para classificar o item.",
		Action: "Informe NCM, GTIN ou cadastre o produto na memoria fiscal.",
	}
}

func buildOperationDiagnostic(resp *SuggestResponse) TaxDiagnosticItem {
	s := resp.Suggestion
	if strings.TrimSpace(s.CFOP) != "" {
		return TaxDiagnosticItem{
			Area:   "operation",
			Status: "ready",
			Title:  "Operacao fiscal definida",
			Detail: "A sugestao possui CFOP ou operacao padrao para orientar a saida fiscal.",
			Action: "Revise se a natureza da operacao corresponde ao caso real.",
		}
	}
	return TaxDiagnosticItem{
		Area:   "operation",
		Status: "missing",
		Title:  "Operacao fiscal incompleta",
		Detail: "Sem CFOP, o motor nao consegue fechar o fluxo fiscal com seguranca.",
		Action: "Selecione uma operacao ou cadastre a regra CFOP correspondente.",
	}
}

func buildICMSDiagnostic(req SuggestRequest, resp *SuggestResponse) TaxDiagnosticItem {
	s := resp.Suggestion
	if stateRule := stateICMSLegalBasis(resp); stateRule != nil {
		detail := "Regra estadual aplicada para UF " + firstNonEmpty(stateRule.UF, strings.ToUpper(strings.TrimSpace(req.RecipientUF)), strings.ToUpper(strings.TrimSpace(req.EmitterUF))) + "."
		if strings.TrimSpace(stateRule.ReferenceCode) != "" {
			detail += " Fonte: " + strings.TrimSpace(stateRule.ReferenceCode) + "."
		}
		if strings.TrimSpace(stateRule.Weight) == "state_icms_rule_exact" {
			detail += " Correspondencia exata por NCM/CEST."
		} else if strings.TrimSpace(stateRule.Weight) == "state_icms_rule_prefix" {
			detail += " Correspondencia por prefixo de NCM."
		} else {
			detail += " Regra geral da UF usada como fallback."
		}
		return TaxDiagnosticItem{
			Area:   "icms",
			Status: "ready",
			Title:  "ICMS estadual aplicado",
			Detail: detail,
			Action: "Revise MVA, FCP, beneficio e excecoes da UF antes de publicar como regra definitiva.",
		}
	}

	if strings.Contains(strings.ToLower(strings.TrimSpace(req.TaxRegime)), "simples") {
		if strings.TrimSpace(s.CSOSN) != "" {
			return TaxDiagnosticItem{
				Area:   "icms",
				Status: "ready",
				Title:  "ICMS do Simples mapeado",
				Detail: "A sugestao possui CSOSN para o regime informado.",
				Action: "Valide excecoes de ST, FCP e beneficio estadual.",
			}
		}
		return TaxDiagnosticItem{
			Area:   "icms",
			Status: "missing",
			Title:  "CSOSN pendente",
			Detail: "Para Simples Nacional, o CSOSN ainda precisa ser definido.",
			Action: "Cadastre regra de ICMS/CSOSN por NCM, CFOP, UF e operacao.",
		}
	}

	if strings.TrimSpace(s.ICMSCST) != "" && strings.TrimSpace(s.ICMSRate) != "" {
		return TaxDiagnosticItem{
			Area:   "icms",
			Status: "ready",
			Title:  "ICMS estruturado",
			Detail: "CST e aliquota de ICMS foram retornados para o contexto.",
			Action: "Confira reducao de base, FCP, ST e DIFAL quando aplicavel.",
		}
	}
	if strings.TrimSpace(s.ICMSCST) != "" || strings.TrimSpace(s.ICMSRate) != "" || strings.TrimSpace(s.DIFALInternalRate) != "" {
		return TaxDiagnosticItem{
			Area:   "icms",
			Status: "attention",
			Title:  "ICMS parcial",
			Detail: "Ha informacao de ICMS, mas ainda falta CST, aliquota ou excecao estadual para fechar a regra.",
			Action: "Complete a aba ICMS com CST, aliquota, FCP, ST e reducoes por UF.",
		}
	}
	return TaxDiagnosticItem{
		Area:   "icms",
		Status: "missing",
		Title:  "ICMS nao definido",
		Detail: "O motor ainda nao possui regra de ICMS suficiente para este contexto.",
		Action: "Cadastre uma regra de ICMS ou aceite uma regra capturada validada.",
	}
}

func buildContributionsDiagnostic(req SuggestRequest, resp *SuggestResponse) TaxDiagnosticItem {
	s := resp.Suggestion
	hasCST := strings.TrimSpace(s.PISCST) != "" && strings.TrimSpace(s.COFINSCST) != ""
	hasRates := strings.TrimSpace(s.PISRate) != "" && strings.TrimSpace(s.COFINSRate) != ""
	if contributionCSTNeedsRevenueNature(resp) && !hasContributionRevenueCode(resp) {
		return TaxDiagnosticItem{
			Area:   "pis_cofins",
			Status: "attention",
			Title:  "Natureza da receita pendente",
			Detail: "CST 04/05/06 indica monofasico, substituicao tributaria ou aliquota zero em PIS/COFINS, mas o codigo de natureza da receita ainda nao foi definido.",
			Action: "Cadastre a natureza da receita do anexo correspondente ao CST antes de usar a sugestao operacionalmente.",
		}
	}
	if hasCST && hasRates {
		return TaxDiagnosticItem{
			Area:   "pis_cofins",
			Status: "ready",
			Title:  "PIS/COFINS completos",
			Detail: "CST e aliquotas foram definidos para o regime informado.",
			Action: "Revise codigo de receita e excecoes monofasicas quando houver.",
		}
	}
	if hasCST || hasRates {
		return TaxDiagnosticItem{
			Area:   "pis_cofins",
			Status: "attention",
			Title:  "PIS/COFINS parciais",
			Detail: "A regra possui parte da contribuicao, mas ainda nao fecha CST e aliquotas.",
			Action: "Complete regras por regime: normal, presumido ou Simples, conforme a operacao.",
		}
	}
	detail := "Nao ha CST ou aliquota de PIS/COFINS para a consulta."
	if strings.TrimSpace(req.TaxRegime) != "" {
		detail += " Regime consultado: " + strings.TrimSpace(req.TaxRegime) + "."
	}
	return TaxDiagnosticItem{
		Area:   "pis_cofins",
		Status: "missing",
		Title:  "PIS/COFINS ausentes",
		Detail: detail,
		Action: "Cadastre CST, aliquotas e codigos de receita por regime tributario.",
	}
}

func contributionCSTNeedsRevenueNature(resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}
	if hasRetailMonophasicOutput(resp) {
		return false
	}
	pis := strings.TrimSpace(resp.Suggestion.PISCST)
	cofins := strings.TrimSpace(resp.Suggestion.COFINSCST)
	return pis == "04" || pis == "05" || pis == "06" || cofins == "04" || cofins == "05" || cofins == "06"
}

func hasContributionRevenueCode(resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}
	return strings.TrimSpace(resp.Suggestion.PISRevenueCode) != "" || strings.TrimSpace(resp.Suggestion.COFINSRevenueCode) != ""
}

func hasContributionBenefitReduction2026(resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}
	hasCST06 := onlyDigits(resp.Suggestion.PISCST) == "06" || onlyDigits(resp.Suggestion.COFINSCST) == "06"
	if !hasCST06 {
		return false
	}
	for _, item := range resp.LegalBasis {
		if strings.EqualFold(strings.TrimSpace(item.ReferenceCode), "LC 224/2025") {
			return true
		}
	}
	return false
}

func hasMonophasicOrSubstitutionContributionCST(resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}
	if hasRetailMonophasicOutput(resp) {
		return false
	}
	pis := onlyDigits(resp.Suggestion.PISCST)
	cofins := onlyDigits(resp.Suggestion.COFINSCST)
	return pis == "04" || pis == "05" || cofins == "04" || cofins == "05"
}

func hasRetailMonophasicOutput(resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}
	for _, item := range resp.LegalBasis {
		if strings.TrimSpace(item.TaxType) == "PIS_COFINS_MONOPHASIC" {
			return true
		}
	}
	return false
}

func hasConfirmedSubstitutionTax(resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}
	if strings.TrimSpace(resp.Suggestion.ICMSCST) == "60" || strings.TrimSpace(resp.Suggestion.CSOSN) == "500" {
		return true
	}
	if isSubstitutionTaxCFOP(resp.Suggestion.CFOP) {
		return true
	}
	for _, item := range resp.LegalBasis {
		if strings.TrimSpace(item.TaxType) == "ICMS_ST" && strings.TrimSpace(item.Weight) != "0.68" {
			return true
		}
	}
	return false
}

func buildReformDiagnostic(resp *SuggestResponse) TaxDiagnosticItem {
	s := resp.Suggestion
	if strings.TrimSpace(s.CClasTrib) != "" && (strings.TrimSpace(s.IBSRate) != "" || strings.TrimSpace(s.CBSRate) != "") {
		return TaxDiagnosticItem{
			Area:   "reform",
			Status: "ready",
			Title:  "Reforma tributaria preparada",
			Detail: "cClasTrib e aliquotas de IBS/CBS estao presentes na sugestao.",
			Action: "Revise vigencia e regra transitoria antes de publicar.",
		}
	}
	if strings.TrimSpace(s.CClasTrib) != "" || strings.TrimSpace(s.IBSRate) != "" || strings.TrimSpace(s.CBSRate) != "" {
		return TaxDiagnosticItem{
			Area:   "reform",
			Status: "attention",
			Title:  "Reforma parcial",
			Detail: "Ha informacao de IBS/CBS, mas o cadastro ainda nao esta completo.",
			Action: "Complete cClasTrib, IBS, CBS e imposto seletivo quando aplicavel.",
		}
	}
	return TaxDiagnosticItem{
		Area:   "reform",
		Status: "missing",
		Title:  "Reforma nao mapeada",
		Detail: "A consulta ainda nao possui parametros de IBS/CBS para este item.",
		Action: "Alimente a aba IBS, CBS e cClasTrib com regras oficiais.",
	}
}

func buildLegalBasisDiagnostic(resp *SuggestResponse) TaxDiagnosticItem {
	if hasSpecificLegalBasis(resp) {
		return TaxDiagnosticItem{
			Area:   "legal_basis",
			Status: "ready",
			Title:  "Base legal conectada",
			Detail: "A sugestao retornou regras ou referencias legais aplicadas ao contexto.",
			Action: "Use a triagem para aprovar novas regras capturadas antes de publicar.",
		}
	}
	if hasRetailDefault(resp) {
		return TaxDiagnosticItem{
			Area:   "legal_basis",
			Status: "attention",
			Title:  "Default varejista aplicado",
			Detail: "A consulta usou o perfil padrao de varejo, mas ainda nao retornou uma regra legal especifica.",
			Action: "Cadastre regra fiscal por produto, CST, CFOP, regime e UF para aumentar confianca.",
		}
	}
	return TaxDiagnosticItem{
		Area:   "legal_basis",
		Status: "attention",
		Title:  "Base legal nao vinculada",
		Detail: "A sugestao foi gerada, mas nao retornou uma regra legal especifica.",
		Action: "Vincule regra fiscal, fonte legal e condicoes para aumentar confianca.",
	}
}

func hasRetailDefault(resp *SuggestResponse) bool {
	if resp == nil {
		return false
	}
	for _, item := range resp.LegalBasis {
		if strings.TrimSpace(item.TaxType) == "RETAIL_DEFAULT" {
			return true
		}
	}
	return false
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

func isOutboundDirection(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(value, "saida") ||
		strings.Contains(value, "saída") ||
		strings.Contains(value, "outbound") ||
		strings.Contains(value, "exit")
}
