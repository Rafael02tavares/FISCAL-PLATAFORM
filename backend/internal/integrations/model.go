package integrations

type IntegrationSetting struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Provider       string `json:"provider"`
	Enabled        bool   `json:"enabled"`
	BaseURL        string `json:"base_url"`
	ModelName      string `json:"model_name"`
	HasToken       bool   `json:"has_token"`
	TokenPreview   string `json:"token_preview"`
	Notes          string `json:"notes"`
	UpdatedAt      string `json:"updated_at"`
}

type OpenAITestResult struct {
	OK             bool           `json:"ok"`
	StatusCode     int            `json:"status_code"`
	Message        string         `json:"message"`
	Model          string         `json:"model"`
	Output         string         `json:"output"`
	Classification map[string]any `json:"classification,omitempty"`
}

type CosmosTestResult struct {
	OK          bool           `json:"ok"`
	StatusCode  int            `json:"status_code"`
	Message     string         `json:"message"`
	GTIN        string         `json:"gtin"`
	Description string         `json:"description"`
	NCM         string         `json:"ncm"`
	Raw         map[string]any `json:"raw,omitempty"`
}

type CosmosProductCandidate struct {
	Description    string         `json:"description"`
	GTIN           string         `json:"gtin"`
	NCM            string         `json:"ncm"`
	NCMDescription string         `json:"ncm_description"`
	CEST           string         `json:"cest"`
	Brand          string         `json:"brand"`
	Thumbnail      string         `json:"thumbnail"`
	Source         string         `json:"source"`
	Raw            map[string]any `json:"raw,omitempty"`
}

type CosmosSearchResult struct {
	OK         bool                     `json:"ok"`
	StatusCode int                      `json:"status_code"`
	Message    string                   `json:"message"`
	Query      string                   `json:"query"`
	Items      []CosmosProductCandidate `json:"items"`
}
