import hashlib
import json
import math
import random
import re
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
    def __init__(self, endpoint: str, api_key: str, model: str) -> None:
        from openai import OpenAI

        self._client = OpenAI(base_url=endpoint, api_key=api_key)
        self._model = model

    def embed(self, texts: list[str]) -> list[list[float]]:
        response = self._client.embeddings.create(model=self._model, input=texts)
        return [item.embedding for item in response.data]


class RemoteReranker:
    def __init__(self, endpoint: str, api_key: str, model: str) -> None:
        self._endpoint = endpoint.rstrip("/") + "/rerank"
        self._api_key = api_key
        self._model = model

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
        with urllib.request.urlopen(request, timeout=30) as response:
            body = json.loads(response.read())
        scored = [(int(item["index"]), float(item["relevance_score"])) for item in body["results"]]
        scored.sort(key=lambda item: item[1], reverse=True)
        return scored


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
