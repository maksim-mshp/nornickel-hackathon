import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from embed.embedder import CachingEmbedder


class CountingEmbedder:
    def __init__(self):
        self.calls = 0

    def embed(self, texts):
        self.calls += len(texts)
        return [[float(len(text))] for text in texts]


def test_cache_hits_avoid_recompute():
    inner = CountingEmbedder()
    cache = CachingEmbedder(inner, 10)
    first = cache.embed(["a", "bb"])
    assert inner.calls == 2
    second = cache.embed(["a", "bb"])
    assert inner.calls == 2
    assert first == second == [[1.0], [2.0]]


def test_cache_partial_hit():
    inner = CountingEmbedder()
    cache = CachingEmbedder(inner, 10)
    cache.embed(["a"])
    result = cache.embed(["a", "ccc"])
    assert inner.calls == 2
    assert result == [[1.0], [3.0]]


def test_cache_lru_eviction():
    inner = CountingEmbedder()
    cache = CachingEmbedder(inner, 2)
    cache.embed(["a", "b"])
    cache.embed(["c"])
    cache.embed(["a"])
    assert inner.calls == 4


def test_cache_disabled():
    inner = CountingEmbedder()
    cache = CachingEmbedder(inner, 0)
    cache.embed(["a"])
    cache.embed(["a"])
    assert inner.calls == 2


def _run():
    passed = 0
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            fn()
            passed += 1
    print(f"{passed} cache tests passed")


if __name__ == "__main__":
    _run()
