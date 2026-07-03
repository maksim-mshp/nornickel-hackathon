import hashlib
import json
import math
import random
import re
import urllib.request

DIMENSIONS = 1024


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
        self._endpoint = endpoint.rstrip("/") + "/embeddings"
        self._api_key = api_key
        self._model = model

    def embed(self, texts: list[str]) -> list[list[float]]:
        payload = json.dumps({"model": self._model, "input": texts}).encode("utf-8")
        request = urllib.request.Request(
            self._endpoint,
            data=payload,
            headers={"Content-Type": "application/json", "Authorization": f"Bearer {self._api_key}"},
            method="POST",
        )
        with urllib.request.urlopen(request, timeout=30) as response:
            body = json.loads(response.read())
        return [item["embedding"] for item in body["data"]]


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
