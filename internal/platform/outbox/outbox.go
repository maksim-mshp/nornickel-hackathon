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
	Drain(ctx context.Context, batch int, publish func(context.Context, Record) error) (int, error)
}

const claimSQL = `select id, aggregate_type, aggregate_id, event_type, payload, headers, created_at
from ops.outbox
where published_at is null
order by created_at
limit $1
for update skip locked`

const markPublishedSQL = `update ops.outbox set published_at = now() where id = any($1)`

type PGStore struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

type claimedRecord struct {
	id     uuid.UUID
	record Record
}

func (store *PGStore) Drain(ctx context.Context, batch int, publish func(context.Context, Record) error) (int, error) {
	if batch <= 0 {
		batch = DefaultBatch
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	claimed, err := claimRecords(ctx, tx, batch)
	if err != nil {
		return 0, err
	}

	var publishedIDs []uuid.UUID
	for _, item := range claimed {
		if err := publish(ctx, item.record); err != nil {
			continue
		}
		publishedIDs = append(publishedIDs, item.id)
	}
	if len(publishedIDs) > 0 {
		if _, err := tx.Exec(ctx, markPublishedSQL, publishedIDs); err != nil {
			return 0, fmt.Errorf("mark outbox published: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit outbox tx: %w", err)
	}
	return len(publishedIDs), nil
}

func claimRecords(ctx context.Context, tx pgx.Tx, batch int) ([]claimedRecord, error) {
	rows, err := tx.Query(ctx, claimSQL, batch)
	if err != nil {
		return nil, fmt.Errorf("claim outbox: %w", err)
	}
	defer rows.Close()

	var claimed []claimedRecord
	for rows.Next() {
		id, record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		claimed = append(claimed, claimedRecord{id: id, record: record})
	}
	return claimed, rows.Err()
}

func scanRecord(row pgx.Row) (uuid.UUID, Record, error) {
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
		return uuid.Nil, Record{}, fmt.Errorf("scan outbox row: %w", err)
	}
	env, err := events.Unmarshal(payload)
	if err != nil {
		return uuid.Nil, Record{}, fmt.Errorf("decode outbox envelope: %w", err)
	}
	var headers map[string]string
	if len(headersRaw) > 0 {
		if err := json.Unmarshal(headersRaw, &headers); err != nil {
			return uuid.Nil, Record{}, fmt.Errorf("decode outbox headers: %w", err)
		}
	}
	return id, Record{
		Envelope:      env,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Headers:       headers,
	}, nil
}
