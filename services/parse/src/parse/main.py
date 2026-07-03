import asyncio
import io
import json
import logging
import signal
import uuid
from datetime import datetime, timezone

import nats
from minio import Minio
from minio.error import S3Error

from parse.config import Config, load
from parse.docir import build_docir, extract_text

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger("parse")

REGISTERED = "kmap.doc.v1.registered"
PARSED = "kmap.doc.v1.parsed"
PARSE_FAILED = "kmap.doc.v1.parse-failed"
TERMINAL_S3_CODES = {"NoSuchKey", "NoSuchBucket"}
RETRY_DELAY_SECONDS = 10


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

    connection = await nats.connect(cfg.nats_url, name="kmap-parse")
    js = connection.jetstream()
    try:
        await js.add_stream(name="KMAP_DOCS", subjects=["kmap.doc.v1.>"])
    except Exception:
        pass

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

            try:
                text, source_format, pages = extract_text(raw)
            except Exception as error:
                await fail_terminal(msg, document_id, f"unparseable document: {error}")
                return
            docir = build_docir(document_id, text, source_format, pages)
            docir_key = f"{document_id}/docir.json"
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

    await js.subscribe(REGISTERED, durable="kmap-parse", cb=handle, manual_ack=True)
    logger.info("parse worker subscribed to %s", REGISTERED)

    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, stop.set)
        except NotImplementedError:
            pass
    await stop.wait()
    await connection.drain()


def main() -> None:
    asyncio.run(run())


if __name__ == "__main__":
    main()
