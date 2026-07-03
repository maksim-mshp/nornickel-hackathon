import pathlib

import pytest
import yaml

from extract.numcore import extract_facts

GOLDEN = yaml.safe_load((pathlib.Path(__file__).parent / "golden.yml").read_text(encoding="utf-8"))


@pytest.mark.parametrize("case", GOLDEN, ids=[case["text"] for case in GOLDEN])
def test_numcore_golden(case: dict) -> None:
    facts = extract_facts(case["text"])

    if case.get("none"):
        assert facts == [], f"expected no facts from {case['text']!r}, got {facts}"
        return

    assert facts, f"no fact extracted from {case['text']!r}"
    if "count" in case:
        assert len(facts) == case["count"], f"{case['text']!r}: expected {case['count']} facts, got {len(facts)}"
    fact = facts[0]

    if "operator" in case:
        assert fact.operator == case["operator"]
    if "unit_code" in case:
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
    if "flags" in case:
        assert fact.flags == case["flags"]
    if "confidence" in case:
        assert fact.confidence == pytest.approx(case["confidence"])


def test_no_false_positive_on_year() -> None:
    facts = extract_facts("отчёт 2023 года без измеримых величин")
    assert facts == []


def test_range_uses_first_match_priority() -> None:
    facts = extract_facts("диапазон 60–70 °C стабильный")
    assert len(facts) == 1
    assert facts[0].operator == "range"
    assert facts[0].vmin == pytest.approx(60)
    assert facts[0].vmax == pytest.approx(70)


def test_worded_range_keeps_both_bounds() -> None:
    facts = extract_facts("скорость поддерживали от 0.6 до 0.9 м/с в ячейке")
    assert len(facts) == 1
    assert facts[0].operator == "range"
    assert facts[0].vmin == pytest.approx(0.6)
    assert facts[0].vmax == pytest.approx(0.9)
    assert facts[0].unit_code == "m_per_s"


def test_unicode_minus_normalized_to_range() -> None:
    facts = extract_facts("температура 60−80 °C держалась")
    assert len(facts) == 1
    assert facts[0].operator == "range"
    assert facts[0].vmin == pytest.approx(60)
    assert facts[0].vmax == pytest.approx(80)


def test_nonbreaking_space_normalized() -> None:
    facts = extract_facts("порог 250 мг/л не превышен")
    assert facts
    assert facts[0].operator == "eq"
    assert facts[0].vmin == pytest.approx(250)
    assert facts[0].unit_code == "mg_per_l"


def test_span_offsets_captured() -> None:
    text = "скорость 0.8 м/с при нагрузке"
    facts = extract_facts(text)
    assert facts
    fact = facts[0]
    assert fact.char_to > fact.char_from
    assert text[fact.char_from : fact.char_to] == fact.value_raw
