package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
)

const (
	DefaultBatch    = 100
	DefaultInterval = time.Second
)

type Record struct {
	Envelope      events.Envelope
	AggregateType string
	AggregateID   *uuid.UUID
	Headers       map[string]string
}

const appendSQL = `insert into ops.outbox
(id, aggregate_type, aggregate_id, event_type, payload, headers, created_at)
values ($1, $2, $3, $4, $5, $6, $7)`

func Append(ctx context.Context, tx pgx.Tx, record Record) error {
	id, err := uuid.Parse(record.Envelope.ID)
	if err != nil {
		return fmt.Errorf("outbox envelope id must be uuid: %w", err)
	}
	payload, err := record.Envelope.Marshal()
	if err != nil {
		return fmt.Errorf("marshal outbox envelope: %w", err)
	}
	headers := record.Headers
	if headers == nil {
		headers = map[string]string{}
	}
	if _, err := tx.Exec(ctx, appendSQL, id, record.AggregateType, record.AggregateID,
		record.Envelope.Type, payload, headers, record.Envelope.Time); err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}
	return nil
}

type Store interface {
	Claim(ctx context.Context, limit int) ([]Record, error)
	MarkPublished(ctx context.Context, id uuid.UUID) error
}

const claimSQL = `select id, aggregate_type, aggregate_id, event_type, payload, headers, created_at
from ops.outbox
where published_at is null
order by created_at
limit $1`

const markPublishedSQL = `update ops.outbox set published_at = now() where id = $1`

type PGStore struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (store *PGStore) Claim(ctx context.Context, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = DefaultBatch
	}
	rows, err := store.pool.Query(ctx, claimSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("claim outbox: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (store *PGStore) MarkPublished(ctx context.Context, id uuid.UUID) error {
	if _, err := store.pool.Exec(ctx, markPublishedSQL, id); err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}
	return nil
}

func scanRecord(row pgx.Row) (Record, error) {
	var (
		id            uuid.UUID
		aggregateType string
		aggregateID   *uuid.UUID
		eventType     string
		payload       []byte
		headersRaw    []byte
		createdAt     time.Time
	)
	if err := row.Scan(&id, &aggregateType, &aggregateID, &eventType, &payload, &headersRaw, &createdAt); err != nil {
		return Record{}, fmt.Errorf("scan outbox row: %w", err)
	}
	env, err := events.Unmarshal(payload)
	if err != nil {
		return Record{}, fmt.Errorf("decode outbox envelope: %w", err)
	}
	var headers map[string]string
	if len(headersRaw) > 0 {
		if err := json.Unmarshal(headersRaw, &headers); err != nil {
			return Record{}, fmt.Errorf("decode outbox headers: %w", err)
		}
	}
	return Record{
		Envelope:      env,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Headers:       headers,
	}, nil
}
