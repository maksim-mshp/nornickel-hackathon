import hashlib
import json
import math
import random
import re
import threading
import time
import urllib.error
import urllib.request
from collections import OrderedDict

DIMENSIONS = 1024


class CachingEmbedder:
    def __init__(self, inner, capacity: int) -> None:
        self._inner = inner
        self._capacity = max(0, capacity)
        self._cache: OrderedDict[str, list[float]] = OrderedDict()

    def embed(self, texts: list[str]) -> list[list[float]]:
        if self._capacity == 0:
            return self._inner.embed(texts)
        result: list[list[float] | None] = [None] * len(texts)
        missing_texts: list[str] = []
        missing_slots: list[tuple[int, str]] = []
        for index, text in enumerate(texts):
            key = hashlib.sha256(text.encode("utf-8")).hexdigest()
            cached = self._cache.get(key)
            if cached is None:
                missing_texts.append(text)
                missing_slots.append((index, key))
            else:
                self._cache.move_to_end(key)
                result[index] = cached
        if missing_texts:
            vectors = self._inner.embed(missing_texts)
            for (index, key), vector in zip(missing_slots, vectors):
                result[index] = vector
                self._cache[key] = vector
                self._cache.move_to_end(key)
                while len(self._cache) > self._capacity:
                    self._cache.popitem(last=False)
        return [vector for vector in result if vector is not None]


class DeterministicEmbedder:
    def embed(self, texts: list[str]) -> list[list[float]]:
        return [self._vector(text) for text in texts]

    def _vector(self, text: str) -> list[float]:
        seed = int.from_bytes(hashlib.sha256(text.encode("utf-8")).digest()[:8], "big")
        rng = random.Random(seed)
        values = [rng.gauss(0.0, 1.0) for _ in range(DIMENSIONS)]
        norm = math.sqrt(sum(value * value for value in values)) or 1.0
        return [value / norm for value in values]


class RemoteEmbedder:
    def __init__(
        self, endpoint: str, api_key: str, model: str, max_retries: int = 6, max_concurrency: int = 2
    ) -> None:
        from openai import OpenAI

        self._client = OpenAI(base_url=endpoint, api_key=api_key, max_retries=max(0, max_retries))
        self._model = model
        self._semaphore = threading.Semaphore(max(1, max_concurrency))

    def embed(self, texts: list[str]) -> list[list[float]]:
        with self._semaphore:
            response = self._client.embeddings.create(model=self._model, input=texts)
        return [item.embedding for item in response.data]


class RemoteReranker:
    def __init__(self, endpoint: str, api_key: str, model: str, max_retries: int = 6) -> None:
        self._endpoint = endpoint.rstrip("/") + "/rerank"
        self._api_key = api_key
        self._model = model
        self._max_retries = max(0, max_retries)

    def rerank(self, query: str, passages: list[str]) -> list[tuple[int, float]]:
        if not passages:
            return []
        payload = json.dumps(
            {"model": self._model, "query": query, "documents": passages, "top_n": len(passages)}
        ).encode("utf-8")
        request = urllib.request.Request(
            self._endpoint,
            data=payload,
            headers={"Content-Type": "application/json", "Authorization": f"Bearer {self._api_key}"},
            method="POST",
        )
        body = self._send(request)
        scored = [(int(item["index"]), float(item["relevance_score"])) for item in body["results"]]
        scored.sort(key=lambda item: item[1], reverse=True)
        return scored

    def _send(self, request: urllib.request.Request) -> dict:
        for attempt in range(self._max_retries + 1):
            try:
                with urllib.request.urlopen(request, timeout=30) as response:
                    return json.loads(response.read())
            except urllib.error.HTTPError as error:
                if error.code != 429 or attempt == self._max_retries:
                    raise
                time.sleep(_retry_delay(error, attempt))
        raise RuntimeError("rerank retries exhausted")


def _retry_delay(error: urllib.error.HTTPError, attempt: int) -> float:
    header = error.headers.get("Retry-After") if error.headers else None
    if header:
        try:
            return min(float(header), 30.0)
        except ValueError:
            pass
    return min(2.0**attempt, 30.0) + random.uniform(0.0, 0.5)


class LocalReranker:
    def rerank(self, query: str, passages: list[str]) -> list[tuple[int, float]]:
        return rerank(query, passages)


def rerank(query: str, passages: list[str]) -> list[tuple[int, float]]:
    query_tokens = _tokens(query)
    scores = []
    for index, passage in enumerate(passages):
        passage_tokens = _tokens(passage)
        union = query_tokens | passage_tokens
        overlap = query_tokens & passage_tokens
        score = 0.0 if not union else len(overlap) / len(union)
        scores.append((index, score))
    scores.sort(key=lambda item: item[1], reverse=True)
    return scores


def _tokens(text: str) -> set[str]:
    return set(re.findall(r"[a-zа-яё0-9]+", text.lower()))
