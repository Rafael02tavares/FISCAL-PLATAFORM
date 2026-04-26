package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var ErrInvalidIntegrationInput = errors.New("invalid integration input")
var ErrIntegrationTokenMissing = errors.New("integration token missing")

type Service struct {
	repo   *Repository
	client *http.Client
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		client: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

func (s *Service) GetCosmos(ctx context.Context, organizationID string) (IntegrationSetting, error) {
	if strings.TrimSpace(organizationID) == "" {
		return IntegrationSetting{}, ErrInvalidIntegrationInput
	}

	item, err := s.repo.Get(ctx, organizationID, CosmosProvider)
	if err != nil {
		return IntegrationSetting{}, err
	}

	return toSetting(item), nil
}

type SaveCosmosParams struct {
	OrganizationID string
	Enabled        bool
	BaseURL        string
	APIToken       string
	Notes          string
}

func (s *Service) SaveCosmos(ctx context.Context, params SaveCosmosParams) (IntegrationSetting, error) {
	if strings.TrimSpace(params.OrganizationID) == "" {
		return IntegrationSetting{}, ErrInvalidIntegrationInput
	}

	baseURL := strings.TrimRight(strings.TrimSpace(params.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultCosmosBaseURL
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return IntegrationSetting{}, ErrInvalidIntegrationInput
	}

	item, err := s.repo.Upsert(ctx, UpsertParams{
		OrganizationID: params.OrganizationID,
		Provider:       CosmosProvider,
		Enabled:        params.Enabled,
		BaseURL:        baseURL,
		ModelName:      "",
		APIToken:       params.APIToken,
		Notes:          params.Notes,
	})
	if err != nil {
		return IntegrationSetting{}, err
	}

	return toSetting(item), nil
}

func (s *Service) GetOpenAI(ctx context.Context, organizationID string) (IntegrationSetting, error) {
	if strings.TrimSpace(organizationID) == "" {
		return IntegrationSetting{}, ErrInvalidIntegrationInput
	}

	item, err := s.repo.Get(ctx, organizationID, OpenAIProvider)
	if err != nil {
		return IntegrationSetting{}, err
	}
	if item.BaseURL == "" {
		item.BaseURL = DefaultOpenAIBaseURL
	}
	if item.ModelName == "" {
		item.ModelName = DefaultOpenAIModel
	}

	return toSetting(item), nil
}

type SaveOpenAIParams struct {
	OrganizationID string
	Enabled        bool
	BaseURL        string
	ModelName      string
	APIToken       string
	Notes          string
}

func (s *Service) SaveOpenAI(ctx context.Context, params SaveOpenAIParams) (IntegrationSetting, error) {
	if strings.TrimSpace(params.OrganizationID) == "" {
		return IntegrationSetting{}, ErrInvalidIntegrationInput
	}

	baseURL := strings.TrimRight(strings.TrimSpace(params.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultOpenAIBaseURL
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return IntegrationSetting{}, ErrInvalidIntegrationInput
	}

	modelName := strings.TrimSpace(params.ModelName)
	if modelName == "" {
		modelName = DefaultOpenAIModel
	}

	item, err := s.repo.Upsert(ctx, UpsertParams{
		OrganizationID: params.OrganizationID,
		Provider:       OpenAIProvider,
		Enabled:        params.Enabled,
		BaseURL:        baseURL,
		ModelName:      modelName,
		APIToken:       params.APIToken,
		Notes:          params.Notes,
	})
	if err != nil {
		return IntegrationSetting{}, err
	}

	return toSetting(item), nil
}

type OpenAIClassificationInput struct {
	Description string
	GTIN        string
	NCM         string
	CEST        string
	UF          string
	TaxRegime   string
	Operation   string
}

func (s *Service) TestOpenAI(ctx context.Context, organizationID string, transientToken string, modelName string, input OpenAIClassificationInput) (OpenAITestResult, error) {
	if strings.TrimSpace(organizationID) == "" {
		return OpenAITestResult{}, ErrInvalidIntegrationInput
	}

	item, err := s.repo.Get(ctx, organizationID, OpenAIProvider)
	if err != nil {
		return OpenAITestResult{}, err
	}

	token := strings.TrimSpace(transientToken)
	if token == "" {
		token = strings.TrimSpace(item.APIToken)
	}
	if token == "" {
		return OpenAITestResult{}, ErrIntegrationTokenMissing
	}

	baseURL := strings.TrimRight(strings.TrimSpace(item.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultOpenAIBaseURL
	}

	model := strings.TrimSpace(modelName)
	if model == "" {
		model = strings.TrimSpace(item.ModelName)
	}
	if model == "" {
		model = DefaultOpenAIModel
	}

	input.Description = strings.TrimSpace(input.Description)
	input.GTIN = normalizeDigits(input.GTIN)
	input.NCM = normalizeDigits(input.NCM)
	input.CEST = normalizeDigits(input.CEST)
	input.UF = strings.ToUpper(strings.TrimSpace(input.UF))
	input.TaxRegime = strings.TrimSpace(input.TaxRegime)
	input.Operation = strings.TrimSpace(input.Operation)
	if input.Description == "" {
		input.Description = "HEINEKEN LN ZERO 6X330ML RV"
		input.NCM = "22029100"
		input.GTIN = "7896045506057"
	}
	if input.UF == "" {
		input.UF = "GO"
	}
	if input.TaxRegime == "" {
		input.TaxRegime = "simples_nacional"
	}
	if input.Operation == "" {
		input.Operation = "venda interna para consumidor final"
	}

	payload := map[string]any{
		"model": model,
		"input": buildOpenAIClassificationPrompt(input),
		"text": map[string]any{
			"format": map[string]any{
				"type": "json_object",
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return OpenAITestResult{}, fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return OpenAITestResult{}, fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	response, err := s.client.Do(req)
	if err != nil {
		return OpenAITestResult{}, fmt.Errorf("openai request failed: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return OpenAITestResult{
			OK:         false,
			StatusCode: response.StatusCode,
			Message:    "OpenAI respondeu, mas o corpo da resposta nao pode ser lido",
			Model:      model,
		}, nil
	}

	var responsePayload map[string]any
	if err := json.Unmarshal(responseBody, &responsePayload); err != nil {
		bodyText := strings.TrimSpace(string(responseBody))
		if bodyText == "" {
			bodyText = "Resposta vazia da OpenAI."
		}
		return OpenAITestResult{
			OK:         response.StatusCode >= 200 && response.StatusCode < 300,
			StatusCode: response.StatusCode,
			Message:    "OpenAI retornou texto fora do envelope JSON esperado",
			Model:      model,
			Output:     bodyText,
			Classification: map[string]any{
				"produto_normalizado":       input.Description,
				"categoria_fiscal_provavel": "revisao manual",
				"ncm_informado":             input.NCM,
				"cest_informado":            input.CEST,
				"sinais":                    []any{"resposta bruta da OpenAI nao estruturada"},
				"risco":                     "medio",
				"confianca":                 0.3,
				"acao_recomendada":          "revisar resposta bruta e repetir a consulta se necessario",
				"observacao":                truncateText(bodyText, 500),
			},
		}, nil
	}

	output := extractOpenAIOutputText(responsePayload)
	classification := parseJSONObjectFromText(output)
	if classification == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
		classification = fallbackClassificationFromOutput(input, output)
	}
	result := OpenAITestResult{
		OK:             response.StatusCode >= 200 && response.StatusCode < 300,
		StatusCode:     response.StatusCode,
		Message:        "Consulta OpenAI executada",
		Model:          model,
		Output:         output,
		Classification: classification,
	}
	if !result.OK {
		result.Message = extractOpenAIErrorMessage(responsePayload)
	}
	if result.OK && result.Output == "" {
		result.Output = "Conexao validada, sem texto retornado."
	}

	return result, nil
}

func (s *Service) ClassifyWithOpenAI(ctx context.Context, organizationID string, input OpenAIClassificationInput) (OpenAITestResult, bool, error) {
	item, err := s.repo.Get(ctx, organizationID, OpenAIProvider)
	if err != nil {
		return OpenAITestResult{}, false, err
	}
	if !item.Enabled || strings.TrimSpace(item.APIToken) == "" {
		return OpenAITestResult{}, false, nil
	}

	result, err := s.TestOpenAI(ctx, organizationID, "", item.ModelName, input)
	if err != nil {
		return OpenAITestResult{}, true, err
	}
	return result, true, nil
}

func buildOpenAIClassificationPrompt(input OpenAIClassificationInput) string {
	return fmt.Sprintf(`Voce e um assistente de classificacao fiscal para varejo brasileiro.
Sua tarefa e apoiar triagem humana, sem inventar lei e sem aplicar regra fiscal automaticamente.

Contexto padrao:
- Operacao: %s
- UF: %s
- Regime tributario: %s

Produto:
- Descricao: %s
- GTIN: %s
- NCM: %s
- CEST: %s

Responda somente em JSON valido, sem markdown, usando este formato:
{
  "produto_normalizado": "texto curto",
  "categoria_fiscal_provavel": "texto curto",
  "ncm_informado": "codigo",
  "cest_informado": "codigo ou vazio",
  "sinais": ["lista curta"],
  "risco": "baixo|medio|alto",
  "confianca": 0.0,
  "acao_recomendada": "texto curto",
  "observacao": "texto curto"
}`, input.Operation, input.UF, input.TaxRegime, input.Description, input.GTIN, input.NCM, input.CEST)
}

func (s *Service) TestCosmosGTIN(ctx context.Context, organizationID string, gtin string, transientToken string) (CosmosTestResult, error) {
	gtin = normalizeDigits(gtin)
	if strings.TrimSpace(organizationID) == "" || gtin == "" {
		return CosmosTestResult{}, ErrInvalidIntegrationInput
	}

	item, err := s.repo.Get(ctx, organizationID, CosmosProvider)
	if err != nil {
		return CosmosTestResult{}, err
	}

	token := strings.TrimSpace(transientToken)
	if token == "" {
		token = strings.TrimSpace(item.APIToken)
	}
	if token == "" {
		return CosmosTestResult{}, ErrIntegrationTokenMissing
	}

	baseURL := strings.TrimRight(strings.TrimSpace(item.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultCosmosBaseURL
	}

	requestURL := fmt.Sprintf("%s/gtins/%s.json", baseURL, gtin)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return CosmosTestResult{}, fmt.Errorf("create cosmos request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cosmos-Token", token)

	response, err := s.client.Do(req)
	if err != nil {
		return CosmosTestResult{}, fmt.Errorf("cosmos request failed: %w", err)
	}
	defer response.Body.Close()

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return CosmosTestResult{
			OK:         false,
			StatusCode: response.StatusCode,
			Message:    "Cosmos respondeu, mas o JSON nao pode ser interpretado",
			GTIN:       gtin,
		}, nil
	}

	result := CosmosTestResult{
		OK:          response.StatusCode >= 200 && response.StatusCode < 300,
		StatusCode:  response.StatusCode,
		Message:     "Consulta Cosmos executada",
		GTIN:        gtin,
		Description: stringFromAny(firstPayloadValue(payload, "description", "descricao", "name", "nome", "product_description")),
		NCM:         normalizeDigits(stringFromAny(firstPayloadValue(payload, "ncm", "ncm_code"))),
		Raw:         payload,
	}
	if !result.OK {
		result.Message = "Cosmos retornou erro para a consulta"
	}

	return result, nil
}

func (s *Service) SearchCosmosProducts(ctx context.Context, organizationID string, query string, transientToken string, limit int) (CosmosSearchResult, error) {
	query = strings.TrimSpace(query)
	if strings.TrimSpace(organizationID) == "" || query == "" {
		return CosmosSearchResult{}, ErrInvalidIntegrationInput
	}
	if limit <= 0 || limit > 30 {
		limit = 10
	}

	item, err := s.repo.Get(ctx, organizationID, CosmosProvider)
	if err != nil {
		return CosmosSearchResult{}, err
	}

	token := strings.TrimSpace(transientToken)
	if token == "" {
		token = strings.TrimSpace(item.APIToken)
	}
	if token == "" {
		return CosmosSearchResult{}, ErrIntegrationTokenMissing
	}

	baseURL := strings.TrimRight(strings.TrimSpace(item.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultCosmosBaseURL
	}

	result, err := s.fetchCosmosProductSearch(ctx, baseURL, token, query, limit, false)
	if err != nil {
		return CosmosSearchResult{}, err
	}
	if result.StatusCode == http.StatusNotFound {
		fallbackResult, fallbackErr := s.fetchCosmosProductSearch(ctx, baseURL, token, query, limit, true)
		if fallbackErr == nil {
			result = fallbackResult
		}
	}
	return result, nil
}

func (s *Service) fetchCosmosProductSearch(ctx context.Context, baseURL string, token string, query string, limit int, useJSONPath bool) (CosmosSearchResult, error) {
	endpoint := "/products"
	if useJSONPath {
		endpoint = "/products.json"
	}
	requestURL := fmt.Sprintf("%s%s?query=%s&per_page=%d", baseURL, endpoint, url.QueryEscape(query), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return CosmosSearchResult{}, fmt.Errorf("create cosmos search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "FiscalPlatform/1.0")
	req.Header.Set("X-Cosmos-Token", token)

	response, err := s.client.Do(req)
	if err != nil {
		return CosmosSearchResult{}, fmt.Errorf("cosmos search request failed: %w", err)
	}
	defer response.Body.Close()

	var payload any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return CosmosSearchResult{
			OK:         false,
			StatusCode: response.StatusCode,
			Message:    "Cosmos respondeu, mas o JSON da busca nao pode ser interpretado",
			Query:      query,
			Items:      []CosmosProductCandidate{},
		}, nil
	}

	items := extractCosmosCandidates(payload, limit)
	result := CosmosSearchResult{
		OK:         response.StatusCode >= 200 && response.StatusCode < 300,
		StatusCode: response.StatusCode,
		Message:    "Busca Cosmos executada",
		Query:      query,
		Items:      items,
	}
	if !result.OK {
		result.Message = "Cosmos retornou erro para a busca por descricao"
	}
	if result.OK && len(result.Items) == 0 {
		result.Message = "Busca executada, mas nenhum produto com NCM/GTIN foi identificado"
	}
	return result, nil
}

func toSetting(item integrationRecord) IntegrationSetting {
	token := strings.TrimSpace(item.APIToken)
	return IntegrationSetting{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		Provider:       item.Provider,
		Enabled:        item.Enabled,
		BaseURL:        firstNonEmpty(item.BaseURL, DefaultCosmosBaseURL),
		ModelName:      item.ModelName,
		HasToken:       token != "",
		TokenPreview:   maskToken(token),
		Notes:          item.Notes,
		UpdatedAt:      item.UpdatedAt,
	}
}

func extractCosmosCandidates(payload any, limit int) []CosmosProductCandidate {
	rawItems := extractCosmosRawItems(payload)
	items := make([]CosmosProductCandidate, 0, len(rawItems))
	seen := make(map[string]bool)

	for _, raw := range rawItems {
		candidate := mapCosmosCandidate(raw)
		if candidate.Description == "" && candidate.GTIN == "" && candidate.NCM == "" {
			continue
		}
		key := firstNonEmpty(candidate.GTIN, candidate.NCM+"|"+strings.ToLower(candidate.Description))
		if key != "" && seen[key] {
			continue
		}
		seen[key] = true
		items = append(items, candidate)
		if limit > 0 && len(items) >= limit {
			break
		}
	}

	return items
}

func extractCosmosRawItems(payload any) []map[string]any {
	switch typed := payload.(type) {
	case []any:
		return mapsFromAnySlice(typed)
	case map[string]any:
		for _, key := range []string{"products", "items", "data", "results", "content"} {
			if value, ok := typed[key]; ok {
				if items := extractCosmosRawItems(value); len(items) > 0 {
					return items
				}
			}
		}
		return []map[string]any{typed}
	default:
		return nil
	}
}

func mapsFromAnySlice(values []any) []map[string]any {
	items := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			items = append(items, item)
		}
	}
	return items
}

func mapCosmosCandidate(payload map[string]any) CosmosProductCandidate {
	ncmValue := firstNestedPayloadValue(payload,
		[]string{"ncm"},
		[]string{"ncm_code"},
		[]string{"ncm", "code"},
		[]string{"ncm", "codigo"},
	)
	cestValue := firstNestedPayloadValue(payload,
		[]string{"cest"},
		[]string{"cest_code"},
		[]string{"cest", "code"},
		[]string{"cest", "codigo"},
	)

	return CosmosProductCandidate{
		Description: stringFromAny(firstNestedPayloadValue(payload,
			[]string{"description"},
			[]string{"descricao"},
			[]string{"name"},
			[]string{"nome"},
			[]string{"product_description"},
		)),
		GTIN: normalizeDigits(stringFromAny(firstCosmosGTINValue(payload))),
		NCM:  normalizeDigits(stringFromAny(ncmValue)),
		NCMDescription: stringFromAny(firstNestedPayloadValue(payload,
			[]string{"ncm_description"},
			[]string{"ncm", "description"},
			[]string{"ncm", "descricao"},
		)),
		CEST: normalizeDigits(stringFromAny(cestValue)),
		Brand: stringFromAny(firstNestedPayloadValue(payload,
			[]string{"brand"},
			[]string{"brand", "name"},
			[]string{"marca"},
			[]string{"marca", "nome"},
		)),
		Thumbnail: stringFromAny(firstNestedPayloadValue(payload,
			[]string{"thumbnail"},
			[]string{"image"},
			[]string{"image_url"},
			[]string{"picture"},
			[]string{"url_image"},
		)),
		Source: "cosmos_bluesoft",
		Raw:    payload,
	}
}

func firstCosmosGTINValue(payload map[string]any) any {
	if value := firstNestedPayloadValue(payload,
		[]string{"gtin"},
		[]string{"barcode"},
		[]string{"ean"},
		[]string{"codigo_barras"},
	); value != nil {
		return value
	}
	if gtins, ok := payload["gtins"].([]any); ok && len(gtins) > 0 {
		if first, ok := gtins[0].(map[string]any); ok {
			return firstNestedPayloadValue(first, []string{"gtin"}, []string{"code"}, []string{"codigo"})
		}
		return gtins[0]
	}
	return nil
}

func firstNestedPayloadValue(payload map[string]any, paths ...[]string) any {
	for _, path := range paths {
		value := nestedPayloadValue(payload, path)
		if stringFromAny(value) != "" {
			return value
		}
	}
	return nil
}

func nestedPayloadValue(payload map[string]any, path []string) any {
	var current any = payload
	for _, key := range path {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		value, ok := currentMap[key]
		if !ok {
			return nil
		}
		current = value
	}
	return current
}

func extractOpenAIErrorMessage(payload map[string]any) string {
	if errorPayload, ok := payload["error"].(map[string]any); ok {
		if message := stringFromAny(errorPayload["message"]); message != "" {
			return message
		}
	}
	return "OpenAI retornou erro para a consulta"
}

func extractOpenAIOutputText(payload map[string]any) string {
	if value := stringFromAny(payload["output_text"]); value != "" {
		return value
	}

	output, ok := payload["output"].([]any)
	if !ok {
		return ""
	}

	parts := make([]string, 0)
	for _, item := range output {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		content, ok := itemMap["content"].([]any)
		if !ok {
			continue
		}
		for _, contentItem := range content {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			if text := stringFromAny(contentMap["text"]); text != "" {
				parts = append(parts, text)
			}
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func parseJSONObjectFromText(value string) map[string]any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	start := strings.Index(value, "{")
	end := strings.LastIndex(value, "}")
	if start < 0 || end <= start {
		return nil
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(value[start:end+1]), &parsed); err != nil {
		return nil
	}
	return parsed
}

func fallbackClassificationFromOutput(input OpenAIClassificationInput, output string) map[string]any {
	output = strings.TrimSpace(output)
	if output == "" {
		output = "A OpenAI nao retornou texto classificavel."
	}
	return map[string]any{
		"produto_normalizado":       firstNonEmpty(input.Description, "Item informado"),
		"categoria_fiscal_provavel": "revisao manual",
		"ncm_informado":             input.NCM,
		"cest_informado":            input.CEST,
		"sinais":                    []any{"resposta sem JSON estruturado"},
		"risco":                     "medio",
		"confianca":                 0.4,
		"acao_recomendada":          "revisar classificacao assistida antes de usar no motor",
		"observacao":                truncateText(output, 500),
	}
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "••••"
	}
	return token[:4] + "••••" + token[len(token)-4:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var digitPattern = regexp.MustCompile(`\D+`)

func normalizeDigits(value string) string {
	return digitPattern.ReplaceAllString(strings.TrimSpace(value), "")
}

func firstPayloadValue(payload map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value
		}
	}
	return nil
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}
