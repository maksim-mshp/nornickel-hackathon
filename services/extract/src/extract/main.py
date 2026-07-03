import asyncio
import base64
import hashlib
import io
import json
import logging
import signal
import uuid
from datetime import datetime, timezone

import nats
from minio import Minio

from extract.config import Config, load
from extract.numcore import Fact, extract_facts

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger("extract")

PARSED = "kmap.doc.v1.parsed"
EXTRACTED = "kmap.doc.v1.extracted"
EXTRACTOR_VERSION = "numcore-py-1.0"


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


def _condition_hash(conditions: dict[str, str]) -> str:
    canonical = json.dumps(conditions, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    return base64.b64encode(hashlib.sha256(canonical.encode("utf-8")).digest()).decode("ascii")


def _bundle(document_id: str, text: str, facts: list[Fact]) -> dict:
    short = document_id.replace("-", "")[:8]
    subject_slug = f"topic:doc-{short}"
    entities = [{"slug": subject_slug, "etype": "topic", "name": f"Документ {short}"}]
    seen_params: set[str] = set()
    for fact in facts:
        if fact.parameter_slug in seen_params:
            continue
        seen_params.add(fact.parameter_slug)
        entities.append({"slug": fact.parameter_slug, "etype": "parameter", "name": fact.parameter_name})

    numeric_facts = [
        {
            "subject_slug": subject_slug,
            "parameter_slug": fact.parameter_slug,
            "operator": fact.operator,
            "value_raw": fact.value_raw,
            "vmin": fact.vmin,
            "vmax": fact.vmax,
            "unit_orig": fact.unit_orig,
            "unit_code": fact.unit_code,
            "vmin_si": fact.vmin_si,
            "vmax_si": fact.vmax_si,
            "conditions": fact.conditions,
            "condition_hash": _condition_hash(fact.conditions),
            "quote": fact.quote,
            "char_from": fact.char_from,
            "char_to": fact.char_to,
            "page": 1,
            "geography": "unknown",
            "extraction_method": "deterministic",
            "extractor_version": EXTRACTOR_VERSION,
            "confidence": 0.95,
        }
        for fact in facts
    ]
    return {
        "document_id": document_id,
        "extractor_version": EXTRACTOR_VERSION,
        "entities": entities,
        "chunks": [{"ordinal": 0, "text": text[:4000], "kind": "text", "page_from": 1}],
        "numeric_facts": numeric_facts,
    }


def _envelope(event_type: str, subject: str, data: dict) -> tuple[str, bytes]:
    event_id = str(uuid.uuid4())
    envelope = {
        "specversion": "1.0",
        "id": event_id,
        "source": "kmap/extract",
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

    connection = await nats.connect(cfg.nats_url, name="kmap-extract")
    js = connection.jetstream()
    try:
        await js.add_stream(name="KMAP_DOCS", subjects=["kmap.doc.v1.>"])
    except Exception:
        pass

    async def handle(msg) -> None:
        try:
            envelope = json.loads(msg.data)
            data = envelope.get("data", {})
            document_id = data.get("document_id")
            docir_uri = data.get("docir_uri")
            if not document_id or not docir_uri:
                await msg.ack()
                return

            bucket, key = _parse_uri(docir_uri)
            response = store.get_object(bucket, key)
            try:
                raw = response.read()
            finally:
                response.close()
                response.release_conn()
            docir = json.loads(raw)
            text = docir.get("full_text", "")

            facts = extract_facts(text)
            bundle = _bundle(document_id, text, facts)
            bundle_key = f"{document_id}/bundle.json"
            payload = json.dumps(bundle).encode("utf-8")
            store.put_object(cfg.s3.bundles_bucket, bundle_key, io.BytesIO(payload), length=len(payload))
            bundle_uri = f"s3://{cfg.s3.bundles_bucket}/{bundle_key}"

            event_id, envelope_bytes = _envelope(
                EXTRACTED, document_id, {"document_id": document_id, "bundle_uri": bundle_uri}
            )
            await js.publish(EXTRACTED, envelope_bytes, headers={"Nats-Msg-Id": event_id})
            logger.info("extracted document %s: %d facts", document_id, len(facts))
            await msg.ack()
        except Exception as error:
            logger.exception("extract failed: %s", error)
            await msg.nak()

    await js.subscribe(PARSED, durable="kmap-extract-parsed", cb=handle, manual_ack=True)
    logger.info("extract worker subscribed to %s", PARSED)

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
