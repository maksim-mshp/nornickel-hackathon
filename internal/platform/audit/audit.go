package audit

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Record struct {
	ActorID    string
	Action     string
	ObjectType string
	ObjectID   string
	RequestID  string
	IP         string
	Details    map[string]any
}

type Writer struct {
	pool *pgxpool.Pool
}

func NewWriter(pool *pgxpool.Pool) *Writer {
	return &Writer{pool: pool}
}

const insertSQL = `insert into ops.audit_log (actor_id, action, object_type, object_id, request_id, ip, details)
values ($1, $2, nullif($3, ''), nullif($4, ''), nullif($5, ''), nullif($6, '')::inet, $7)`

func (writer *Writer) Write(ctx context.Context, record Record) error {
	details := record.Details
	if details == nil {
		details = map[string]any{}
	}
	_, err := writer.pool.Exec(ctx, insertSQL,
		record.ActorID, record.Action, record.ObjectType, record.ObjectID,
		record.RequestID, record.IP, details)
	if err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}
