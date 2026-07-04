import asyncio
import concurrent.futures
import io
import json
import logging
import os
import signal
import uuid
from datetime import datetime, timezone

import nats
from minio import Minio
from minio.error import S3Error
from nats.errors import TimeoutError as NatsTimeoutError
from nats.js.api import AckPolicy, ConsumerConfig, DeliverPolicy

from parse.config import Config, load
from parse.docir import build_docir, extract_text

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger("parse")

REGISTERED = "kmap.doc.v1.registered"
PARSED = "kmap.doc.v1.parsed"
PARSE_FAILED = "kmap.doc.v1.parse-failed"
DURABLE = "kmap-parse"
TERMINAL_S3_CODES = {"NoSuchKey", "NoSuchBucket"}
RETRY_DELAY_SECONDS = 10
MAX_DOCUMENT_BYTES = 200 * 1024 * 1024
MAX_PAGES = 2000
ACK_WAIT_SECONDS = 900
MAX_DELIVER = 5
FETCH_TIMEOUT_SECONDS = 5


def _minio(cfg: Config) -> Minio:
    return Minio(
        cfg.s3.endpoint,
        access_key=cfg.s3.access_key,
        secret_key=cfg.s3.secret_key,
        secure=cfg.s3.use_ssl,
    )


def _parse_uri(uri: str) -> tuple[str, str]:
    rest = uri.removeprefix("s3://")
    bucket, _, key = rest.partition("/")
    return bucket, key


def _worker_count(cfg: Config) -> int:
    if cfg.workers > 0:
        return cfg.workers
    return os.cpu_count() or 4


def _object_exists(store: Minio, bucket: str, key: str) -> bool:
    try:
        store.stat_object(bucket, key)
        return True
    except S3Error as error:
        if error.code in TERMINAL_S3_CODES:
            return False
        raise


def _envelope(event_type: str, subject: str, data: dict) -> tuple[str, bytes]:
    event_id = str(uuid.uuid4())
    envelope = {
        "specversion": "1.0",
        "id": event_id,
        "source": "kmap/parse",
        "type": event_type,
        "subject": subject,
        "time": datetime.now(timezone.utc).isoformat(),
        "datacontenttype": "application/json",
        "data": data,
    }
    return event_id, json.dumps(envelope).encode("utf-8")


async def run() -> None:
    cfg = load()
    store = _minio(cfg)
    workers = _worker_count(cfg)
    pool = concurrent.futures.ProcessPoolExecutor(max_workers=workers)
    loop = asyncio.get_running_loop()

    connection = await nats.connect(cfg.nats_url, name="kmap-parse")
    js = connection.jetstream()
    try:
        await js.add_stream(name="KMAP_DOCS", subjects=["kmap.doc.v1.>"])
    except Exception:
        pass

    try:
        await js.delete_consumer("KMAP_DOCS", DURABLE)
    except Exception:
        pass
    psub = await js.pull_subscribe(
        REGISTERED,
        durable=DURABLE,
        config=ConsumerConfig(
            ack_wait=ACK_WAIT_SECONDS,
            max_deliver=MAX_DELIVER,
            ack_policy=AckPolicy.EXPLICIT,
            deliver_policy=DeliverPolicy.ALL,
        ),
    )

    async def fail_terminal(msg, document_id: str, reason: str) -> None:
        attempt = msg.metadata.num_delivered if msg.metadata else 1
        event_id, envelope_bytes = _envelope(
            PARSE_FAILED,
            document_id,
            {"document_id": document_id, "reason": reason, "attempt": attempt},
        )
        await js.publish(PARSE_FAILED, envelope_bytes, headers={"Nats-Msg-Id": event_id})
        logger.error("parse failed terminally for document %s: %s", document_id, reason)
        await msg.term()

    async def handle(msg) -> None:
        try:
            envelope = json.loads(msg.data)
            data = envelope.get("data", {})
            document_id = data.get("document_id")
            blob_uri = data.get("blob_uri")
            if not document_id or not blob_uri:
                await msg.ack()
                return

            docir_key = f"{document_id}/docir.json"
            if _object_exists(store, cfg.s3.docir_bucket, docir_key):
                await msg.ack()
                return

            bucket, key = _parse_uri(blob_uri)
            try:
                response = store.get_object(bucket, key)
                try:
                    raw = response.read()
                finally:
                    response.close()
                    response.release_conn()
            except S3Error as error:
                if error.code in TERMINAL_S3_CODES:
                    await fail_terminal(msg, document_id, f"blob not found: {blob_uri}")
                    return
                raise

            if len(raw) > MAX_DOCUMENT_BYTES:
                await fail_terminal(
                    msg, document_id, f"document too large: {len(raw)} bytes > {MAX_DOCUMENT_BYTES}"
                )
                return

            try:
                text, source_format, pages = await loop.run_in_executor(pool, extract_text, raw)
            except Exception as error:
                await fail_terminal(msg, document_id, f"unparseable document: {error}")
                return

            if pages > MAX_PAGES:
                await fail_terminal(msg, document_id, f"too many pages: {pages} > {MAX_PAGES}")
                return
            docir = build_docir(document_id, text, source_format, pages)
            payload = json.dumps(docir).encode("utf-8")
            store.put_object(cfg.s3.docir_bucket, docir_key, io.BytesIO(payload), length=len(payload))
            docir_uri = f"s3://{cfg.s3.docir_bucket}/{docir_key}"

            event_id, envelope_bytes = _envelope(
                PARSED,
                document_id,
                {
                    "document_id": document_id,
                    "docir_uri": docir_uri,
                    "lang": docir["lang"],
                    "doc_type_detected": source_format,
                    "pages": pages,
                    "tables": 0,
                },
            )
            await js.publish(PARSED, envelope_bytes, headers={"Nats-Msg-Id": event_id})
            logger.info("parsed document %s: %s, %d pages, %d blocks", document_id, source_format, pages, len(docir["blocks"]))
            await msg.ack()
        except Exception as error:
            logger.exception("parse failed: %s", error)
            await msg.nak(delay=RETRY_DELAY_SECONDS)

    stop = asyncio.Event()
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, stop.set)
        except NotImplementedError:
            pass

    async def worker() -> None:
        while not stop.is_set():
            try:
                msgs = await psub.fetch(1, timeout=FETCH_TIMEOUT_SECONDS)
            except (NatsTimeoutError, asyncio.TimeoutError):
                continue
            except Exception as error:
                logger.warning("fetch failed: %s", error)
                await asyncio.sleep(1)
                continue
            for msg in msgs:
                await handle(msg)

    tasks = [asyncio.create_task(worker()) for _ in range(workers)]
    logger.info("parse worker pool started: %d workers", workers)

    await stop.wait()
    for task in tasks:
        task.cancel()
    await asyncio.gather(*tasks, return_exceptions=True)
    pool.shutdown(wait=False, cancel_futures=True)
    await connection.drain()


def main() -> None:
    asyncio.run(run())


if __name__ == "__main__":
    main()
