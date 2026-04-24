package icmsrates

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	repo *Repository
	now  func() time.Time
}

type RateReference struct {
	Mode            string    `json:"mode"`
	IssuerUF        string    `json:"issuer_uf"`
	RecipientUF     string    `json:"recipient_uf"`
	InternalRate    string    `json:"internal_rate"`
	InterstateRate  string    `json:"interstate_rate"`
	FCPRate         string    `json:"fcp_rate"`
	DifferenceRate  string    `json:"difference_rate"`
	ValidFrom       string    `json:"valid_from"`
	ValidTo         string    `json:"valid_to"`
	SourceReference string    `json:"source_reference"`
	SourceURL       string    `json:"source_url"`
	Notes           string    `json:"notes"`
	ResolvedAt      time.Time `json:"resolved_at"`
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) ListStateRates(ctx context.Context) ([]StateRate, error) {
	return s.repo.ListStateRates(ctx)
}

func (s *Service) UpsertStateRate(ctx context.Context, p UpsertStateRateParams) (string, error) {
	p.UF = normalizeUF(p.UF)
	p.InternalRate = normalizeDecimalInput(p.InternalRate)
	p.FCPRate = normalizeDecimalInput(p.FCPRate)
	p.ValidFrom = strings.TrimSpace(p.ValidFrom)
	p.ValidTo = strings.TrimSpace(p.ValidTo)
	p.SourceReference = strings.TrimSpace(p.SourceReference)
	p.SourceURL = strings.TrimSpace(p.SourceURL)
	p.Notes = strings.TrimSpace(p.Notes)

	if p.UF == "" {
		return "", errors.New("uf is required")
	}
	if p.ValidFrom == "" {
		return "", errors.New("valid_from is required")
	}
	if _, err := time.Parse("2006-01-02", p.ValidFrom); err != nil {
		return "", errors.New("valid_from must be in YYYY-MM-DD format")
	}
	if p.ValidTo != "" {
		if _, err := time.Parse("2006-01-02", p.ValidTo); err != nil {
			return "", errors.New("valid_to must be in YYYY-MM-DD format")
		}
	}
	if _, err := parseRate(p.InternalRate); err != nil {
		return "", errors.New("internal_rate must be numeric")
	}
	if p.FCPRate != "" {
		if _, err := parseRate(p.FCPRate); err != nil {
			return "", errors.New("fcp_rate must be numeric")
		}
	}
	if p.SourceReference == "" {
		p.SourceReference = "Cadastro manual da plataforma"
	}

	return s.repo.UpsertStateRate(ctx, p)
}

func (s *Service) ResolveReference(ctx context.Context, issuerUF string, recipientUF string) (*RateReference, error) {
	issuerUF = normalizeUF(issuerUF)
	recipientUF = normalizeUF(recipientUF)
	if issuerUF == "" || recipientUF == "" {
		return nil, nil
	}

	referenceDate := s.now().Format("2006-01-02")
	stateRate, err := s.repo.FindApplicableStateRate(ctx, recipientUF, referenceDate)
	if err != nil {
		return nil, fmt.Errorf("resolve destination state rate: %w", err)
	}
	if stateRate == nil {
		return nil, nil
	}

	interstateRate, mode, notes := deriveInterstateRate(issuerUF, recipientUF)
	if mode == "INTERNAL" {
		interstateRate = stateRate.InternalRate
	}

	differenceRate := "0.00"
	if mode == "INTERSTATE" {
		internalNumeric, _ := parseRate(stateRate.InternalRate)
		interstateNumeric, _ := parseRate(interstateRate)
		diff := internalNumeric - interstateNumeric
		if diff < 0 {
			diff = 0
		}
		differenceRate = formatRate(diff)
	}

	return &RateReference{
		Mode:            mode,
		IssuerUF:        issuerUF,
		RecipientUF:     recipientUF,
		InternalRate:    stateRate.InternalRate,
		InterstateRate:  interstateRate,
		FCPRate:         defaultZero(stateRate.FCPRate),
		DifferenceRate:  differenceRate,
		ValidFrom:       stateRate.ValidFrom,
		ValidTo:         stateRate.ValidTo,
		SourceReference: stateRate.SourceReference,
		SourceURL:       stateRate.SourceURL,
		Notes:           firstNonEmpty(stateRate.Notes, notes),
		ResolvedAt:      s.now(),
	}, nil
}

func deriveInterstateRate(issuerUF string, recipientUF string) (string, string, string) {
	if issuerUF == recipientUF {
		return "", "INTERNAL", "Operacao interna: a partilha interestadual nao se aplica."
	}

	if isSouthOrSoutheastExcludingES(issuerUF) && isNorthNortheastCenterWestOrES(recipientUF) {
		return "7.00", "INTERSTATE", "Aliquota interestadual padrao de 7% para origem Sul/Sudeste (exceto ES) e destino Norte, Nordeste, Centro-Oeste ou ES."
	}

	return "12.00", "INTERSTATE", "Aliquota interestadual padrao de 12% para as demais combinacoes estaduais internas ao territorio nacional."
}

func normalizeUF(value string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	if len(value) != 2 {
		return ""
	}
	return value
}

func normalizeDecimalInput(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", "."))
	return value
}

func parseRate(value string) (float64, error) {
	return strconv.ParseFloat(normalizeDecimalInput(value), 64)
}

func formatRate(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func defaultZero(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "0.00"
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

func isSouthOrSoutheastExcludingES(uf string) bool {
	switch uf {
	case "SP", "RJ", "MG", "PR", "SC", "RS":
		return true
	default:
		return false
	}
}

func isNorthNortheastCenterWestOrES(uf string) bool {
	switch uf {
	case "AC", "AL", "AM", "AP", "BA", "CE", "DF", "ES", "GO", "MA", "MS", "MT", "PA", "PB", "PE", "PI", "RN", "RO", "RR", "SE", "TO":
		return true
	default:
		return false
	}
}
