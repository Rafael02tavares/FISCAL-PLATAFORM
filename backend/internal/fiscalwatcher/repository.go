package fiscalwatcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Source struct {
	ID            string     `json:"id"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Authority     string     `json:"authority"`
	SourceType    string     `json:"source_type"`
	URL           string     `json:"url"`
	CadenceHours  int        `json:"cadence_hours"`
	Active        bool       `json:"active"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	LastStatus    string     `json:"last_status"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Event struct {
	ID            string         `json:"id"`
	SourceID      string         `json:"source_id"`
	SourceCode    string         `json:"source_code"`
	SourceName    string         `json:"source_name"`
	Authority     string         `json:"authority"`
	Status        string         `json:"status"`
	Severity      string         `json:"severity"`
	DetectionMode string         `json:"detection_mode"`
	Title         string         `json:"title"`
	Summary       string         `json:"summary"`
	Payload       map[string]any `json:"payload"`
	DetectedAt    time.Time      `json:"detected_at"`
}

type CreateEventParams struct {
	SourceID      string
	Status        string
	Severity      string
	DetectionMode string
	Title         string
	Summary       string
	Payload       map[string]any
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListSources(ctx context.Context) ([]Source, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, authority, source_type, url, cadence_hours, active, last_checked_at, last_status, updated_at
		FROM fiscal_watcher_sources
		WHERE active = TRUE
		ORDER BY authority, name
	`)
	if err != nil {
		return nil, fmt.Errorf("query watcher sources: %w", err)
	}
	defer rows.Close()

	var items []Source
	for rows.Next() {
		var item Source
		if err := rows.Scan(
			&item.ID,
			&item.Code,
			&item.Name,
			&item.Authority,
			&item.SourceType,
			&item.URL,
			&item.CadenceHours,
			&item.Active,
			&item.LastCheckedAt,
			&item.LastStatus,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan watcher source: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watcher sources: %w", err)
	}

	return items, nil
}

func (r *Repository) FindSourceByCode(ctx context.Context, code string) (Source, error) {
	var item Source
	err := r.db.QueryRow(ctx, `
		SELECT id, code, name, authority, source_type, url, cadence_hours, active, last_checked_at, last_status, updated_at
		FROM fiscal_watcher_sources
		WHERE code = $1 AND active = TRUE
	`, strings.TrimSpace(code)).Scan(
		&item.ID,
		&item.Code,
		&item.Name,
		&item.Authority,
		&item.SourceType,
		&item.URL,
		&item.CadenceHours,
		&item.Active,
		&item.LastCheckedAt,
		&item.LastStatus,
		&item.UpdatedAt,
	)
	if err != nil {
		return Source{}, fmt.Errorf("find watcher source: %w", err)
	}
	return item, nil
}

func (r *Repository) TouchSource(ctx context.Context, sourceID string, checkedAt time.Time, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE fiscal_watcher_sources
		SET last_checked_at = $2, last_status = $3, updated_at = NOW()
		WHERE id = $1
	`, sourceID, checkedAt, strings.TrimSpace(status))
	if err != nil {
		return fmt.Errorf("touch watcher source: %w", err)
	}
	return nil
}

func (r *Repository) CreateEvent(ctx context.Context, params CreateEventParams) (Event, error) {
	var item Event
	err := r.db.QueryRow(ctx, `
		INSERT INTO fiscal_watcher_events (
			source_id,
			status,
			severity,
			detection_mode,
			title,
			summary,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, '{}'::jsonb))
		RETURNING id, source_id, status, severity, detection_mode, title, summary, payload, detected_at
	`,
		params.SourceID,
		strings.TrimSpace(params.Status),
		strings.TrimSpace(params.Severity),
		strings.TrimSpace(params.DetectionMode),
		strings.TrimSpace(params.Title),
		strings.TrimSpace(params.Summary),
		params.Payload,
	).Scan(
		&item.ID,
		&item.SourceID,
		&item.Status,
		&item.Severity,
		&item.DetectionMode,
		&item.Title,
		&item.Summary,
		&item.Payload,
		&item.DetectedAt,
	)
	if err != nil {
		return Event{}, fmt.Errorf("insert watcher event: %w", err)
	}
	return item, nil
}

func (r *Repository) ListEvents(ctx context.Context, status string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			e.id,
			e.source_id,
			s.code,
			s.name,
			s.authority,
			e.status,
			e.severity,
			e.detection_mode,
			e.title,
			e.summary,
			e.payload,
			e.detected_at
		FROM fiscal_watcher_events e
		INNER JOIN fiscal_watcher_sources s ON s.id = e.source_id
		WHERE ($1 = '' OR e.status = $1)
		ORDER BY e.detected_at DESC
		LIMIT $2
	`, strings.TrimSpace(status), limit)
	if err != nil {
		return nil, fmt.Errorf("query watcher events: %w", err)
	}
	defer rows.Close()

	var items []Event
	for rows.Next() {
		var item Event
		if err := rows.Scan(
			&item.ID,
			&item.SourceID,
			&item.SourceCode,
			&item.SourceName,
			&item.Authority,
			&item.Status,
			&item.Severity,
			&item.DetectionMode,
			&item.Title,
			&item.Summary,
			&item.Payload,
			&item.DetectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan watcher event: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watcher events: %w", err)
	}

	return items, nil
}
