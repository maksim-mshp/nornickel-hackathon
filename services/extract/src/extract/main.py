import asyncio
import base64
import concurrent.futures
import hashlib
import io
import json
import logging
import multiprocessing
import os
import re
import signal
import uuid
from datetime import datetime, timezone

import grpc
import nats
from google.protobuf.json_format import MessageToDict
from google.protobuf.struct_pb2 import Struct
from minio import Minio
from minio.error import S3Error
from nats.errors import TimeoutError as NatsTimeoutError
from nats.js.api import AckPolicy, ConsumerConfig, DeliverPolicy

from extract.config import Config, load
from extract.numcore import Fact, extract_facts
from kmap.v1 import embed_pb2, embed_pb2_grpc, llm_pb2, llm_pb2_grpc

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger("extract")

PARSED = "kmap.doc.v1.parsed"
EXTRACTED = "kmap.doc.v1.extracted"
DURABLE = "kmap-extract-parsed"
EXTRACTOR_VERSION = "numcore-py-1.1"
LLM_TASK = "extract_entities"
LLM_TIMEOUT_SECONDS = 120
EMBED_TIMEOUT_SECONDS = 60
MAX_DELIVER = 5
ACK_WAIT_SECONDS = 900
FETCH_TIMEOUT_SECONDS = 5
TERMINAL_S3_CODES = {"NoSuchKey", "NoSuchBucket"}
CHUNK_TARGET_CHARS = 4000
CHUNK_OVERLAP_CHARS = 500
CHUNK_MIN_CHARS = 400
ENTITY_ETYPES = (
    "material",
    "process",
    "equipment",
    "property",
    "parameter",
    "method",
    "technology",
    "organization",
    "person",
    "geography",
)
MAX_LLM_ENTITIES = 40
SYSTEM_PROMPT = (
    "Ты — экстрактор сущностей из научно-технических текстов горно-металлургической отрасли (R&D). "
    "Извлекай только явно упомянутые в тексте сущности. Возвращай строго валидный JSON без пояснений."
)
_SLUG_RE = re.compile(r"[^a-z0-9а-яё]+")


def _chunk_text(text: str) -> list[dict]:
    text = text.strip()
    total = len(text)
    if total == 0:
        return []
    chunks: list[dict] = []
    start = 0
    ordinal = 0
    while start < total:
        end = min(start + CHUNK_TARGET_CHARS, total)
        if end < total:
            window = text[start:end]
            boundary = max(window.rfind("\n\n"), window.rfind(". "), window.rfind("\n"))
            if boundary > CHUNK_MIN_CHARS:
                end = start + boundary + 1
        body = text[start:end].strip()
        if body:
            lang = "ru" if re.search(r"[а-яё]", body, re.IGNORECASE) else "en"
            chunks.append(
                {
                    "ordinal": ordinal,
                    "text": body,
                    "kind": "text",
                    "lang": lang,
                    "page_from": 1,
                    "char_from": start,
                    "char_to": end,
                }
            )
            ordinal += 1
        if end >= total:
            break
        start = max(end - CHUNK_OVERLAP_CHARS, start + 1)
    return chunks


async def _dead_letter(js, msg, reason: str) -> None:
    subject = "kmap.dlq." + msg.subject.removeprefix("kmap.")
    await js.publish(
        subject,
        msg.data,
        headers={"Kmap-Dlq-Reason": reason, "Kmap-Dlq-Origin": msg.subject},
    )


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


def _condition_hash(conditions: dict[str, str]) -> str:
    canonical = json.dumps(conditions, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    return base64.b64encode(hashlib.sha256(canonical.encode("utf-8")).digest()).decode("ascii")


def _slugify(value: str) -> str:
    slug = _SLUG_RE.sub("-", value.strip().lower()).strip("-")
    return slug[:80]


def _entity_messages(text: str, limit: int) -> list[dict]:
    excerpt = text[:limit]
    etypes = ", ".join(ENTITY_ETYPES)
    user = (
        "Извлеки из текста ключевые сущности предметной области.\n"
        f"Допустимые типы (etype): {etypes}.\n"
        'Верни JSON: {"entities": [{"etype": "...", "name": "<как в тексте>", "name_en": "<англ. термин или пусто>"}]}.\n'
        f"Не выдумывай сущности и числа. Максимум {MAX_LLM_ENTITIES} сущностей, без дубликатов.\n\n"
        f"Текст:\n{excerpt}"
    )
    return [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": user},
    ]


async def _embed_chunks(stub, texts: list[str], mode: str, batch: int) -> list[list[float]]:
    vectors: list[list[float]] = []
    for start in range(0, len(texts), max(1, batch)):
        window = texts[start : start + max(1, batch)]
        response = await stub.Embed(embed_pb2.EmbedRequest(texts=window, mode=mode), timeout=EMBED_TIMEOUT_SECONDS)
        vectors.extend(list(vector.values) for vector in response.vectors)
    return vectors


async def _llm_entities(stub, text: str, limit: int) -> list[dict]:
    payload = Struct()
    payload.update({"messages": _entity_messages(text, limit)})
    response = await stub.Complete(
        llm_pb2.CompleteRequest(task=LLM_TASK, payload=payload), timeout=LLM_TIMEOUT_SECONDS
    )
    if not response.valid:
        return []
    data = MessageToDict(response.json)
    entities: list[dict] = []
    seen: set[str] = set()
    for item in data.get("entities", []):
        if not isinstance(item, dict) or len(entities) >= MAX_LLM_ENTITIES:
            continue
        name = str(item.get("name", "")).strip()
        if not name:
            continue
        etype = str(item.get("etype", "")).strip().lower()
        if etype not in ENTITY_ETYPES:
            etype = "topic"
        name_en = str(item.get("name_en", "")).strip()
        base = _slugify(name_en or name)
        if not base:
            continue
        slug = f"{etype}:{base}"
        if slug in seen:
            continue
        seen.add(slug)
        entities.append({"slug": slug, "etype": etype, "name": name, "name_en": name_en})
    return entities


def _bundle(document_id: str, chunks: list[dict], facts: list[Fact], llm_entities: list[dict]) -> dict:
    short = document_id.replace("-", "")[:8]
    subject_slug = f"topic:doc-{short}"
    entities = [{"slug": subject_slug, "etype": "topic", "name": f"Документ {short}"}]
    seen: set[str] = {subject_slug}
    for entity in llm_entities:
        if entity["slug"] not in seen:
            seen.add(entity["slug"])
            entities.append(entity)
    for fact in facts:
        if fact.parameter_slug in seen:
            continue
        seen.add(fact.parameter_slug)
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
            "confidence": fact.confidence,
            "flags": fact.flags,
        }
        for fact in facts
    ]
    return {
        "document_id": document_id,
        "extractor_version": EXTRACTOR_VERSION,
        "entities": entities,
        "chunks": chunks,
        "numeric_facts": numeric_facts,
        "quality": {
            "nc_count": len(facts),
            "nc_suspect": sum(1 for fact in facts if fact.flags),
            "llm_entities": len(llm_entities),
        },
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
    workers = _worker_count(cfg)
    pool = concurrent.futures.ProcessPoolExecutor(
        max_workers=workers, mp_context=multiprocessing.get_context("forkserver")
    )
    loop = asyncio.get_running_loop()

    embed_channel = grpc.aio.insecure_channel(cfg.embed_addr)
    embed_stub = embed_pb2_grpc.EmbedServiceStub(embed_channel)
    llm_channel = grpc.aio.insecure_channel(cfg.llm_addr)
    llm_stub = llm_pb2_grpc.LLMServiceStub(llm_channel)

    connection = await nats.connect(cfg.nats_url, name="kmap-extract")
    js = connection.jetstream()
    for stream, subjects in (("KMAP_DOCS", ["kmap.doc.v1.>"]), ("KMAP_DLQ", ["kmap.dlq.>"])):
        try:
            await js.add_stream(name=stream, subjects=subjects)
        except Exception:
            pass

    try:
        await js.delete_consumer("KMAP_DOCS", DURABLE)
    except Exception:
        pass
    psub = await js.pull_subscribe(
        PARSED,
        durable=DURABLE,
        config=ConsumerConfig(
            ack_wait=ACK_WAIT_SECONDS,
            max_deliver=MAX_DELIVER,
            ack_policy=AckPolicy.EXPLICIT,
            deliver_policy=DeliverPolicy.ALL,
        ),
    )

    async def handle(msg) -> None:
        try:
            envelope = json.loads(msg.data)
            data = envelope.get("data", {})
            document_id = data.get("document_id")
            docir_uri = data.get("docir_uri")
            if not document_id or not docir_uri:
                await msg.ack()
                return

            bundle_key = f"{document_id}/bundle.json"
            if _object_exists(store, cfg.s3.bundles_bucket, bundle_key):
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

            facts = await loop.run_in_executor(pool, extract_facts, text)
            chunks = _chunk_text(text)

            if chunks:
                try:
                    vectors = await _embed_chunks(
                        embed_stub, [chunk["text"] for chunk in chunks], cfg.embed_mode, cfg.embed_batch
                    )
                    for chunk, vector in zip(chunks, vectors):
                        chunk["embedding"] = vector
                except Exception as error:
                    logger.warning("embed failed for %s: %s", document_id, error)

            llm_entities: list[dict] = []
            if text.strip():
                try:
                    llm_entities = await _llm_entities(llm_stub, text, cfg.llm_char_limit)
                except Exception as error:
                    logger.warning("llm extraction failed for %s: %s", document_id, error)

            bundle = _bundle(document_id, chunks, facts, llm_entities)
            payload = json.dumps(bundle).encode("utf-8")
            store.put_object(cfg.s3.bundles_bucket, bundle_key, io.BytesIO(payload), length=len(payload))
            bundle_uri = f"s3://{cfg.s3.bundles_bucket}/{bundle_key}"

            event_id, envelope_bytes = _envelope(
                EXTRACTED, document_id, {"document_id": document_id, "bundle_uri": bundle_uri}
            )
            await js.publish(EXTRACTED, envelope_bytes, headers={"Nats-Msg-Id": event_id})
            logger.info(
                "extracted document %s: %d facts, %d entities, %d chunks",
                document_id,
                len(facts),
                len(llm_entities),
                len(chunks),
            )
            await msg.ack()
        except Exception as error:
            logger.exception("extract failed: %s", error)
            if msg.metadata.num_delivered >= MAX_DELIVER:
                await _dead_letter(js, msg, str(error))
                await msg.term()
            else:
                await msg.nak()

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
    logger.info("extract worker pool started: %d workers", workers)

    await stop.wait()
    for task in tasks:
        task.cancel()
    await asyncio.gather(*tasks, return_exceptions=True)
    pool.shutdown(wait=False, cancel_futures=True)
    await embed_channel.close()
    await llm_channel.close()
    await connection.drain()


def main() -> None:
    asyncio.run(run())


if __name__ == "__main__":
    main()
