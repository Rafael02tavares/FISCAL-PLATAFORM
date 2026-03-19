package companies

import (
	"context"
	"fmt"
)

type Service struct {
	client *Client
}

func NewService(client *Client) *Service {
	return &Service{client: client}
}

type LookupResult struct {
	CNPJ              string         `json:"cnpj"`
	LegalName         string         `json:"legal_name"`
	TradeName         string         `json:"trade_name"`
	Status            string         `json:"status"`
	Street            string         `json:"street"`
	Number            string         `json:"number"`
	District          string         `json:"district"`
	City              string         `json:"city"`
	State             string         `json:"state"`
	ZipCode           string         `json:"zip_code"`
	MainCNAE          string         `json:"main_cnae"`
	TaxRegimeHint     string         `json:"tax_regime_hint"`
	StateRegistrations []Registration `json:"state_registrations"`
}

type Registration struct {
	Number string `json:"number"`
	State  string `json:"state"`
	Status string `json:"status"`
}

func NewLookupResult(raw *OfficeResponse) *LookupResult {
	result := &LookupResult{
		CNPJ:      raw.TaxID,
		TradeName: raw.Alias,
	}

	if raw.Company != nil {
		if v, ok := raw.Company["name"].(string); ok {
			result.LegalName = v
		}
	}

	if raw.Status != nil {
		if v, ok := raw.Status["text"].(string); ok {
			result.Status = v
		}
	}

	if raw.Address != nil {
		if v, ok := raw.Address["street"].(string); ok {
			result.Street = v
		}
		if v, ok := raw.Address["number"].(string); ok {
			result.Number = v
		}
		if v, ok := raw.Address["district"].(string); ok {
			result.District = v
		}
		if v, ok := raw.Address["city"].(string); ok {
			result.City = v
		}
		if v, ok := raw.Address["state"].(string); ok {
			result.State = v
		}
		if v, ok := raw.Address["zip"].(string); ok {
			result.ZipCode = v
		}
	}

	if raw.MainActivity != nil {
		if v, ok := raw.MainActivity["id"].(string); ok {
			result.MainCNAE = v
		}
	}

	if raw.Simples != nil {
		if opted, ok := raw.Simples["optant"].(bool); ok && opted {
			result.TaxRegimeHint = "simples_nacional"
		}
	}

	for _, reg := range raw.Registrations {
		item := Registration{}
		if v, ok := reg["number"].(string); ok {
			item.Number = v
		}
		if v, ok := reg["state"].(string); ok {
			item.State = v
		}
		if status, ok := reg["status"].(map[string]any); ok {
			if txt, ok := status["text"].(string); ok {
				item.Status = txt
			}
		}
		result.StateRegistrations = append(result.StateRegistrations, item)
	}

	return result
}

func (s *Service) LookupByCNPJ(ctx context.Context, cnpj string) (*LookupResult, error) {
	cnpj = NormalizeCNPJ(cnpj)
	if len(cnpj) != 14 {
		return nil, fmt.Errorf("invalid cnpj")
	}

	raw, err := s.client.LookupOffice(ctx, cnpj)
	if err != nil {
		return nil, err
	}

	return NewLookupResult(raw), nil
}