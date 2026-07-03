import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from eval.metrics import (
    hit_rate_at_k,
    macro_f1,
    mrr,
    precision_at_k,
    prf1,
    recall_at_k,
    reciprocal_rank,
    unit_accuracy,
)


def test_precision_at_k():
    assert precision_at_k(["a", "b", "c", "d"], ["a", "c"], 2) == 0.5
    assert precision_at_k(["a", "c"], ["a", "c"], 10) == 1.0
    assert precision_at_k([], ["a"], 5) == 0.0


def test_recall_at_k():
    assert recall_at_k(["a", "b", "c"], ["a", "c", "x"], 3) == 2 / 3
    assert recall_at_k(["a"], [], 5) == 1.0


def test_reciprocal_rank_and_mrr():
    assert reciprocal_rank(["x", "y", "a"], ["a"]) == 1 / 3
    assert reciprocal_rank(["x"], ["a"]) == 0.0
    assert mrr([(["a", "b"], ["a"]), (["x", "b"], ["b"])]) == (1.0 + 0.5) / 2


def test_hit_rate():
    assert hit_rate_at_k(["e1", "e2", "e3"], ["e3"], 3) == 1.0
    assert hit_rate_at_k(["e1", "e2", "e3"], ["e9"], 3) == 0.0


def test_prf1():
    p, r, f = prf1(["a", "b", "c"], ["a", "b", "d"])
    assert p == 2 / 3
    assert r == 2 / 3
    assert abs(f - 2 / 3) < 1e-9
    assert prf1([], []) == (1.0, 1.0, 1.0)


def test_unit_accuracy():
    assert unit_accuracy([("m/s", "m/s"), ("K", "K"), ("Pa", "kPa")]) == 2 / 3
    assert unit_accuracy([]) == 1.0


def test_macro_f1():
    assert macro_f1([(["a"], ["a"]), (["b"], ["c"])]) == 0.5


def _run():
    passed = 0
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            fn()
            passed += 1
    print(f"{passed} metric tests passed")


if __name__ == "__main__":
    _run()
