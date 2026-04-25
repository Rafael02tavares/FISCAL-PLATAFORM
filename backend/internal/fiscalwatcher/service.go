package fiscalwatcher

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListSources(ctx context.Context) ([]Source, error) {
	return s.repo.ListSources(ctx)
}

func (s *Service) ListEvents(ctx context.Context, status string, limit int) ([]Event, error) {
	return s.repo.ListEvents(ctx, status, limit)
}

func (s *Service) RunCheck(ctx context.Context, sourceCode string) ([]Event, error) {
	sources, err := s.resolveSources(ctx, sourceCode)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	items := make([]Event, 0, len(sources))

	for _, source := range sources {
		if err := s.repo.TouchSource(ctx, source.ID, now, "review_required"); err != nil {
			return nil, err
		}

		title, summary, severity := buildReviewMessage(source)
		event, err := s.repo.CreateEvent(ctx, CreateEventParams{
			SourceID:      source.ID,
			Status:        "review_required",
			Severity:      severity,
			DetectionMode: "manual",
			Title:         title,
			Summary:       summary,
			Payload: map[string]any{
				"source_code":  source.Code,
				"source_name":  source.Name,
				"authority":    source.Authority,
				"url":          source.URL,
				"checked_at":   now.Format(time.RFC3339),
				"review_stage": "pending_ai_review",
			},
		})
		if err != nil {
			return nil, err
		}

		event.SourceCode = source.Code
		event.SourceName = source.Name
		event.Authority = source.Authority
		items = append(items, event)
	}

	return items, nil
}

func (s *Service) resolveSources(ctx context.Context, sourceCode string) ([]Source, error) {
	code := strings.TrimSpace(sourceCode)
	if code == "" {
		return s.repo.ListSources(ctx)
	}

	source, err := s.repo.FindSourceByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("resolve watcher source: %w", err)
	}

	return []Source{source}, nil
}

func buildReviewMessage(source Source) (title string, summary string, severity string) {
	switch source.Code {
	case "planalto_lc87":
		return "Revisar alteracoes da Lei Kandir", "Verificar a fonte oficial do Planalto para identificar mudancas na LC 87 e avaliar impacto em ICMS, credito, ST e partilha.", "high"
	case "confaz_cest":
		return "Revisar alteracoes de catalogo CEST", "Conferir atualizacoes em listas do CONFAZ com potencial impacto em ICMS ST, FCP, segmentacao e enquadramento por produto.", "medium"
	case "portal_nfe":
		return "Revisar comunicados do Portal NF-e", "Analisar notas tecnicas, ajustes operacionais ou mudancas estruturais do ecossistema NF-e que afetem captura, parser ou enriquecimento fiscal.", "medium"
	default:
		return "Revisar fonte fiscal monitorada", "Executar verificacao da fonte oficial, resumir impacto e abrir rascunho de atualizacao para aprovacao humana.", "medium"
	}
}
