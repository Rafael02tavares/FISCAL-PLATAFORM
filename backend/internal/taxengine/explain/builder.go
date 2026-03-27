package explain

import (
	"fmt"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/taxengine/domain"
)

type Builder struct {
	legalBasisRepo domain.LegalBasisRepository
}

func NewBuilder(legalBasisRepo domain.LegalBasisRepository) *Builder {
	return &Builder{
		legalBasisRepo: legalBasisRepo,
	}
}

func (b *Builder) Build(
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
	evaluation domain.RuleEvaluationResult,
	taxes domain.TaxDecisionSet,
) []domain.DecisionExplanation {
	explanations := make([]domain.DecisionExplanation, 0, 8)

	explanations = append(explanations, b.buildClassificationExplanation(normalized, classification))

	if taxes.IPI != nil {
		explanations = append(explanations, b.buildIPIExplanation(taxes.IPI, evaluation.MatchedRules))
	}
	if taxes.PIS != nil {
		explanations = append(explanations, b.buildPISExplanation(taxes.PIS, evaluation.MatchedRules))
	}
	if taxes.COFINS != nil {
		explanations = append(explanations, b.buildCOFINSExplanation(taxes.COFINS, evaluation.MatchedRules))
	}
	if taxes.ICMS != nil {
		explanations = append(explanations, b.buildICMSExplanation(taxes.ICMS, evaluation.MatchedRules))
	}
	if taxes.ICMSST != nil {
		explanations = append(explanations, b.buildICMSSTExplanation(taxes.ICMSST, evaluation.MatchedRules))
	}
	if taxes.FCP != nil {
		explanations = append(explanations, b.buildFCPExplanation(taxes.FCP, evaluation.MatchedRules))
	}
	if taxes.DIFAL != nil {
		explanations = append(explanations, b.buildDIFALExplanation(taxes.DIFAL, evaluation.MatchedRules))
	}

	return compactExplanations(explanations)
}

func (b *Builder) buildClassificationExplanation(
	normalized domain.NormalizedContext,
	classification domain.ClassificationDecision,
) domain.DecisionExplanation {
	conclusion := fmt.Sprintf(
		"Classificação resolvida para o item '%s' com NCM '%s', confiança %.2f e origem '%s'.",
		normalized.ProductDescription,
		emptyFallback(classification.NCM, "não definido"),
		classification.Confidence,
		classification.Source,
	)

	if classification.NeedsReview {
		conclusion += " A decisão exige revisão manual."
	}

	return domain.DecisionExplanation{
		TaxType:         "",
		RuleCode:        "CLASSIFICATION",
		RuleName:        "Classificação Fiscal",
		MatchedFacts:    cloneStrings(classification.Reasons),
		LegalReferences: nil,
		Conclusion:      conclusion,
	}
}

func (b *Builder) buildICMSExplanation(
	decision *domain.ICMSDecision,
	matchedRules []domain.MatchedRule,
) domain.DecisionExplanation {
	rule := findMatchedRuleByID(matchedRules, decision.RuleID)

	conclusion := fmt.Sprintf(
		"ICMS %s com CST '%s', base %.2f, alíquota %.2f%% e valor %.2f.",
		appliesText(decision.Applies),
		emptyFallback(decision.CST, "não informado"),
		decision.BaseValue,
		decision.Rate,
		decision.Amount,
	)

	return domain.DecisionExplanation{
		TaxType:         domain.TaxTypeICMS,
		RuleCode:        rule.RuleCode,
		RuleName:        rule.RuleName,
		MatchedFacts:    cloneStrings(rule.MatchedFacts),
		LegalReferences: cloneStrings(decision.LegalBasisIDs),
		Conclusion:      firstNonEmpty(decision.Reason, conclusion),
	}
}

func (b *Builder) buildICMSSTExplanation(
	decision *domain.ICMSSTDecision,
	matchedRules []domain.MatchedRule,
) domain.DecisionExplanation {
	rule := findMatchedRuleByID(matchedRules, decision.RuleID)

	conclusion := fmt.Sprintf(
		"ICMS ST %s com CST '%s', base %.2f, alíquota %.2f%% e valor %.2f.",
		appliesText(decision.Applies),
		emptyFallback(decision.CST, "não informado"),
		decision.BaseValue,
		decision.Rate,
		decision.Amount,
	)

	return domain.DecisionExplanation{
		TaxType:         domain.TaxTypeICMSST,
		RuleCode:        rule.RuleCode,
		RuleName:        rule.RuleName,
		MatchedFacts:    cloneStrings(rule.MatchedFacts),
		LegalReferences: cloneStrings(decision.LegalBasisIDs),
		Conclusion:      firstNonEmpty(decision.Reason, conclusion),
	}
}

func (b *Builder) buildFCPExplanation(
	decision *domain.FCPDecision,
	matchedRules []domain.MatchedRule,
) domain.DecisionExplanation {
	rule := findMatchedRuleByID(matchedRules, decision.RuleID)

	conclusion := fmt.Sprintf(
		"FCP %s com base %.2f, alíquota %.2f%% e valor %.2f.",
		appliesText(decision.Applies),
		decision.BaseValue,
		decision.Rate,
		decision.Amount,
	)

	return domain.DecisionExplanation{
		TaxType:         domain.TaxTypeFCP,
		RuleCode:        rule.RuleCode,
		RuleName:        rule.RuleName,
		MatchedFacts:    cloneStrings(rule.MatchedFacts),
		LegalReferences: cloneStrings(decision.LegalBasisIDs),
		Conclusion:      firstNonEmpty(decision.Reason, conclusion),
	}
}

func (b *Builder) buildDIFALExplanation(
	decision *domain.DIFALDecision,
	matchedRules []domain.MatchedRule,
) domain.DecisionExplanation {
	rule := findMatchedRuleByID(matchedRules, decision.RuleID)

	conclusion := fmt.Sprintf(
		"DIFAL %s com base %.2f, alíquota interna %.2f%%, interestadual %.2f%% e valor destino %.2f.",
		appliesText(decision.Applies),
		decision.BaseValue,
		decision.InternalRate,
		decision.InterstateRate,
		decision.AmountDestinationUF,
	)

	return domain.DecisionExplanation{
		TaxType:         domain.TaxTypeDIFAL,
		RuleCode:        rule.RuleCode,
		RuleName:        rule.RuleName,
		MatchedFacts:    cloneStrings(rule.MatchedFacts),
		LegalReferences: cloneStrings(decision.LegalBasisIDs),
		Conclusion:      firstNonEmpty(decision.Reason, conclusion),
	}
}

func (b *Builder) buildPISExplanation(
	decision *domain.PISDecision,
	matchedRules []domain.MatchedRule,
) domain.DecisionExplanation {
	rule := findMatchedRuleByID(matchedRules, decision.RuleID)

	conclusion := fmt.Sprintf(
		"PIS %s com CST '%s', base %.2f, alíquota %.2f%% e valor %.2f.",
		appliesText(decision.Applies),
		emptyFallback(decision.CST, "não informado"),
		decision.BaseValue,
		decision.Rate,
		decision.Amount,
	)

	return domain.DecisionExplanation{
		TaxType:         domain.TaxTypePIS,
		RuleCode:        rule.RuleCode,
		RuleName:        rule.RuleName,
		MatchedFacts:    cloneStrings(rule.MatchedFacts),
		LegalReferences: cloneStrings(decision.LegalBasisIDs),
		Conclusion:      firstNonEmpty(decision.Reason, conclusion),
	}
}

func (b *Builder) buildCOFINSExplanation(
	decision *domain.COFINSDecision,
	matchedRules []domain.MatchedRule,
) domain.DecisionExplanation {
	rule := findMatchedRuleByID(matchedRules, decision.RuleID)

	conclusion := fmt.Sprintf(
		"COFINS %s com CST '%s', base %.2f, alíquota %.2f%% e valor %.2f.",
		appliesText(decision.Applies),
		emptyFallback(decision.CST, "não informado"),
		decision.BaseValue,
		decision.Rate,
		decision.Amount,
	)

	return domain.DecisionExplanation{
		TaxType:         domain.TaxTypeCOFINS,
		RuleCode:        rule.RuleCode,
		RuleName:        rule.RuleName,
		MatchedFacts:    cloneStrings(rule.MatchedFacts),
		LegalReferences: cloneStrings(decision.LegalBasisIDs),
		Conclusion:      firstNonEmpty(decision.Reason, conclusion),
	}
}

func (b *Builder) buildIPIExplanation(
	decision *domain.IPIDecision,
	matchedRules []domain.MatchedRule,
) domain.DecisionExplanation {
	rule := findMatchedRuleByID(matchedRules, decision.RuleID)

	conclusion := fmt.Sprintf(
		"IPI %s com CST '%s', base %.2f, alíquota %.2f%% e valor %.2f.",
		appliesText(decision.Applies),
		emptyFallback(decision.CST, "não informado"),
		decision.BaseValue,
		decision.Rate,
		decision.Amount,
	)

	return domain.DecisionExplanation{
		TaxType:         domain.TaxTypeIPI,
		RuleCode:        rule.RuleCode,
		RuleName:        rule.RuleName,
		MatchedFacts:    cloneStrings(rule.MatchedFacts),
		LegalReferences: cloneStrings(decision.LegalBasisIDs),
		Conclusion:      firstNonEmpty(decision.Reason, conclusion),
	}
}

func findMatchedRuleByID(rules []domain.MatchedRule, ruleID string) domain.MatchedRule {
	for _, rule := range rules {
		if rule.RuleID == ruleID {
			return rule
		}
	}
	return domain.MatchedRule{}
}

func appliesText(applies bool) string {
	if applies {
		return "aplicado"
	}
	return "não aplicado"
}

func compactExplanations(values []domain.DecisionExplanation) []domain.DecisionExplanation {
	out := make([]domain.DecisionExplanation, 0, len(values))

	for _, v := range values {
		if strings.TrimSpace(v.RuleCode) == "" &&
			strings.TrimSpace(v.RuleName) == "" &&
			strings.TrimSpace(v.Conclusion) == "" &&
			len(v.MatchedFacts) == 0 &&
			len(v.LegalReferences) == 0 {
			continue
		}
		out = append(out, v)
	}

	return out
}

func emptyFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, len(values))
	copy(out, values)
	return out
}

var _ domain.Explainer = (*Builder)(nil)