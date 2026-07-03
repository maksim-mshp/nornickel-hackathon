package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	kmapv1 "github.com/maksim-mshp/nornickel-hackathon/contracts/gen/go/kmap/v1"
	"github.com/maksim-mshp/nornickel-hackathon/internal/answer/app"
	"google.golang.org/protobuf/encoding/protojson"
)

type Cache struct {
	pool *pgxpool.Pool
	ttl  time.Duration
}

func NewCache(pool *pgxpool.Pool, ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Cache{pool: pool, ttl: ttl}
}

type storedAnswer struct {
	Answer   json.RawMessage `json:"answer"`
	Evidence json.RawMessage `json:"evidence"`
}

const getCacheSQL = `select plan, answer from ops.answer_cache where key = $1 and expires_at > now()`

const putCacheSQL = `insert into ops.answer_cache (key, plan, answer, entity_slugs, expires_at)
values ($1, $2, $3, $4, $5)
on conflict (key) do update set plan = excluded.plan, answer = excluded.answer, entity_slugs = excluded.entity_slugs, created_at = now(), expires_at = excluded.expires_at`

const resolveSlugsSQL = `select slug from kg.entities where id::text = any($1)`

const invalidateSQL = `delete from ops.answer_cache where entity_slugs && $1`

func (cache *Cache) Get(ctx context.Context, key []byte) (*app.CachedAnswer, bool, error) {
	var planRaw, answerRaw []byte
	err := cache.pool.QueryRow(ctx, getCacheSQL, key).Scan(&planRaw, &answerRaw)
	if err == pgx.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read answer cache: %w", err)
	}

	plan := &kmapv1.QueryPlan{}
	if err := protojson.Unmarshal(planRaw, plan); err != nil {
		return nil, false, nil
	}
	var stored storedAnswer
	if err := json.Unmarshal(answerRaw, &stored); err != nil {
		return nil, false, nil
	}
	answer := &kmapv1.AnswerDoc{}
	if err := protojson.Unmarshal(stored.Answer, answer); err != nil {
		return nil, false, nil
	}
	evidence := &kmapv1.EvidencePack{}
	if len(stored.Evidence) > 0 {
		if err := protojson.Unmarshal(stored.Evidence, evidence); err != nil {
			return nil, false, nil
		}
	}
	return &app.CachedAnswer{Plan: plan, Evidence: evidence, Answer: answer}, true, nil
}

func (cache *Cache) Put(ctx context.Context, key []byte, value *app.CachedAnswer) error {
	planRaw, err := protojson.Marshal(value.Plan)
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	answerRaw, err := protojson.Marshal(value.Answer)
	if err != nil {
		return fmt.Errorf("marshal answer: %w", err)
	}
	evidenceRaw, err := protojson.Marshal(value.Evidence)
	if err != nil {
		return fmt.Errorf("marshal evidence: %w", err)
	}
	stored, err := json.Marshal(storedAnswer{Answer: answerRaw, Evidence: evidenceRaw})
	if err != nil {
		return fmt.Errorf("marshal stored answer: %w", err)
	}
	if _, err := cache.pool.Exec(ctx, putCacheSQL, key, planRaw, stored, planSlugs(value.Plan), time.Now().Add(cache.ttl)); err != nil {
		return fmt.Errorf("write answer cache: %w", err)
	}
	return nil
}

func (cache *Cache) ResolveSlugs(ctx context.Context, entityIDs []string) ([]string, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}
	rows, err := cache.pool.Query(ctx, resolveSlugsSQL, entityIDs)
	if err != nil {
		return nil, fmt.Errorf("resolve slugs: %w", err)
	}
	defer rows.Close()
	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

func (cache *Cache) Invalidate(ctx context.Context, slugs []string) (int64, error) {
	if len(slugs) == 0 {
		return 0, nil
	}
	tag, err := cache.pool.Exec(ctx, invalidateSQL, slugs)
	if err != nil {
		return 0, fmt.Errorf("invalidate answer cache: %w", err)
	}
	return tag.RowsAffected(), nil
}

func planSlugs(plan *kmapv1.QueryPlan) []string {
	slugs := []string{}
	fields := plan.GetEntities().GetFields()
	for _, group := range []string{"materials", "processes", "properties"} {
		list := fields[group].GetListValue()
		if list == nil {
			continue
		}
		for _, item := range list.GetValues() {
			if slug := item.GetStructValue().GetFields()["slug"].GetStringValue(); slug != "" {
				slugs = append(slugs, slug)
			}
		}
	}
	return slugs
}
