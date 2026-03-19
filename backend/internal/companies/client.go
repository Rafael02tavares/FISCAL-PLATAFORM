package companies

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL: "https://open.cnpja.com",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type OfficeResponse struct {
	TaxID          string           `json:"taxId"`
	Company        map[string]any   `json:"company"`
	Alias          string           `json:"alias"`
	Founded        string           `json:"founded"`
	Status         map[string]any   `json:"status"`
	Address        map[string]any   `json:"address"`
	Phones         []map[string]any `json:"phones"`
	Emails         []map[string]any `json:"emails"`
	MainActivity   map[string]any   `json:"mainActivity"`
	SideActivities []map[string]any `json:"sideActivities"`
	Registrations  []map[string]any `json:"registrations"`
	Suframa        any              `json:"suframa"`
	Simples        map[string]any   `json:"simples"`
}

func (c *Client) LookupOffice(ctx context.Context, cnpj string) (*OfficeResponse, error) {
	url := fmt.Sprintf("%s/office/%s", c.baseURL, cnpj)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request cnpja: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cnpja returned status %d", resp.StatusCode)
	}

	var out OfficeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode cnpja response: %w", err)
	}

	return &out, nil
}

func NormalizeCNPJ(v string) string {
	var b strings.Builder
	for _, r := range v {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
