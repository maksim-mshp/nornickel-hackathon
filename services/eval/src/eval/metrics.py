from __future__ import annotations

from collections.abc import Iterable, Sequence


def precision_at_k(retrieved: Sequence[str], relevant: Iterable[str], k: int) -> float:
    if k <= 0:
        return 0.0
    top = retrieved[:k]
    if not top:
        return 0.0
    relevant_set = set(relevant)
    hits = sum(1 for item in top if item in relevant_set)
    return hits / len(top)


def recall_at_k(retrieved: Sequence[str], relevant: Iterable[str], k: int) -> float:
    relevant_set = set(relevant)
    if not relevant_set:
        return 1.0
    top = set(retrieved[:k])
    hits = len(top & relevant_set)
    return hits / len(relevant_set)


def reciprocal_rank(retrieved: Sequence[str], relevant: Iterable[str]) -> float:
    relevant_set = set(relevant)
    for position, item in enumerate(retrieved, start=1):
        if item in relevant_set:
            return 1.0 / position
    return 0.0


def mrr(cases: Iterable[tuple[Sequence[str], Iterable[str]]]) -> float:
    ranks = [reciprocal_rank(retrieved, relevant) for retrieved, relevant in cases]
    return sum(ranks) / len(ranks) if ranks else 0.0


def hit_rate_at_k(retrieved: Sequence[str], relevant: Iterable[str], k: int) -> float:
    relevant_set = set(relevant)
    return 1.0 if relevant_set & set(retrieved[:k]) else 0.0


def prf1(predicted: Iterable[str], gold: Iterable[str]) -> tuple[float, float, float]:
    predicted_set = set(predicted)
    gold_set = set(gold)
    if not predicted_set and not gold_set:
        return 1.0, 1.0, 1.0
    true_positive = len(predicted_set & gold_set)
    precision = true_positive / len(predicted_set) if predicted_set else 0.0
    recall = true_positive / len(gold_set) if gold_set else 0.0
    f1 = 2 * precision * recall / (precision + recall) if (precision + recall) else 0.0
    return precision, recall, f1


def unit_accuracy(pairs: Iterable[tuple[str, str]]) -> float:
    pairs = list(pairs)
    if not pairs:
        return 1.0
    correct = sum(1 for predicted, gold in pairs if predicted == gold)
    return correct / len(pairs)


def macro_f1(per_type: Iterable[tuple[Iterable[str], Iterable[str]]]) -> float:
    scores = [prf1(predicted, gold)[2] for predicted, gold in per_type]
    return sum(scores) / len(scores) if scores else 0.0
