import logging
import os

import psycopg

from embed.config import load
from embed.embedder import LocalEmbedder

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger("embed-backfill")

SELECT_SQL = "SELECT id::text, text FROM core.chunks WHERE embedding IS NULL ORDER BY id LIMIT %s"
UPDATE_SQL = "UPDATE core.chunks SET embedding = %s::vector WHERE id = %s"


def _dsn() -> str:
    return os.environ.get("BACKFILL_DSN", "postgres://kmap:kmap@postgres:5432/kmap?sslmode=disable")


def _batch_size() -> int:
    return int(os.environ.get("BACKFILL_BATCH", "256"))


def main() -> None:
    cfg = load()
    embedder = LocalEmbedder(cfg.local_model, cfg.local_max_length, cfg.local_batch, cfg.local_threads)
    batch = _batch_size()
    total = 0
    with psycopg.connect(_dsn()) as conn:
        while True:
            rows = conn.execute(SELECT_SQL, (batch,)).fetchall()
            if not rows:
                break
            vectors = embedder.embed([text for _, text in rows])
            with conn.cursor() as cursor:
                for (chunk_id, _), vector in zip(rows, vectors):
                    literal = "[" + ",".join(repr(value) for value in vector) + "]"
                    cursor.execute(UPDATE_SQL, (literal, chunk_id))
            conn.commit()
            total += len(rows)
            logger.info("embedded %d chunks (total %d)", len(rows), total)
    logger.info("backfill complete: %d chunks", total)


if __name__ == "__main__":
    main()
