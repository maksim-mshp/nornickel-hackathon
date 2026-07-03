import pathlib

import pytest
import yaml

from extract.numcore import extract_facts

GOLDEN = yaml.safe_load((pathlib.Path(__file__).parent / "golden.yml").read_text(encoding="utf-8"))


@pytest.mark.parametrize("case", GOLDEN, ids=[case["text"] for case in GOLDEN])
def test_numcore_golden(case: dict) -> None:
    facts = extract_facts(case["text"])
    assert facts, f"no fact extracted from {case['text']!r}"
    fact = facts[0]

    assert fact.operator == case["operator"]
    assert fact.unit_code == case["unit_code"]
    if "vmin" in case:
        assert fact.vmin == pytest.approx(case["vmin"])
    if "vmax" in case:
        assert fact.vmax == pytest.approx(case["vmax"])
    if "vmin_si" in case:
        assert fact.vmin_si == pytest.approx(case["vmin_si"])
    if "vmax_si" in case:
        assert fact.vmax_si == pytest.approx(case["vmax_si"])
    if "parameter_slug" in case:
        assert fact.parameter_slug == case["parameter_slug"]


def test_no_false_positive_on_year() -> None:
    facts = extract_facts("отчёт 2023 года без измеримых величин")
    assert facts == []


def test_range_uses_first_match_priority() -> None:
    facts = extract_facts("диапазон 60–70 °C стабильный")
    assert len(facts) == 1
    assert facts[0].operator == "range"
    assert facts[0].vmin == pytest.approx(60)
    assert facts[0].vmax == pytest.approx(70)
