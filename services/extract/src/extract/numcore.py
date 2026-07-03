import re
from dataclasses import dataclass, field


@dataclass(frozen=True)
class UnitDef:
    code: str
    factor: float
    offset: float
    parameter_slug: str
    parameter_name: str


UNITS: dict[str, UnitDef] = {
    "м/с": UnitDef("m_per_s", 1.0, 0.0, "parameter:flow-rate", "скорость потока"),
    "m/s": UnitDef("m_per_s", 1.0, 0.0, "parameter:flow-rate", "скорость потока"),
    "°c": UnitDef("celsius", 1.0, 273.15, "parameter:temperature", "температура"),
    "мг/дм³": UnitDef("mg_per_dm3", 1e-3, 0.0, "parameter:concentration", "концентрация"),
    "мг/дм3": UnitDef("mg_per_dm3", 1e-3, 0.0, "parameter:concentration", "концентрация"),
    "мг/л": UnitDef("mg_per_l", 1e-3, 0.0, "parameter:concentration", "концентрация"),
    "mg/l": UnitDef("mg_per_l", 1e-3, 0.0, "parameter:concentration", "концентрация"),
    "%": UnitDef("percent", 1.0, 0.0, "parameter:ratio", "доля"),
    "а/м²": UnitDef("a_per_m2", 1.0, 0.0, "parameter:current-density", "плотность тока"),
    "мпа": UnitDef("mpa", 1e6, 0.0, "parameter:pressure", "давление"),
}

_NUMBER = r"\d+(?:[.,]\d+)?"
_UNIT = r"(м/с|m/s|°c|мг/дм³|мг/дм3|мг/л|mg/l|%|а/м²|мпа)"

_RANGE_WORDS = re.compile(rf"от\s+({_NUMBER})\s+до\s+({_NUMBER})\s*{_UNIT}", re.IGNORECASE)
_RANGE = re.compile(rf"({_NUMBER})\s*[–—-]\s*({_NUMBER})\s*{_UNIT}", re.IGNORECASE)
_UPPER = re.compile(rf"(?:≤|не более|не выше|до|<)\s*({_NUMBER})\s*{_UNIT}", re.IGNORECASE)
_LOWER = re.compile(rf"(?:≥|не менее|не ниже|от|свыше|выше|>)\s*({_NUMBER})\s*{_UNIT}", re.IGNORECASE)
_PM = re.compile(rf"({_NUMBER})\s*±\s*({_NUMBER})\s*{_UNIT}", re.IGNORECASE)
_EQ = re.compile(rf"({_NUMBER})\s*{_UNIT}", re.IGNORECASE)

_NORMALIZE = {
    " ": " ",
    " ": " ",
    " ": " ",
    "−": "-",
    "º": "°",
}


def normalize(text: str) -> str:
    for source, target in _NORMALIZE.items():
        text = text.replace(source, target)
    return text


@dataclass
class Fact:
    operator: str
    value_raw: str
    vmin: float | None
    vmax: float | None
    unit_orig: str
    unit_code: str
    vmin_si: float | None
    vmax_si: float | None
    parameter_slug: str
    parameter_name: str
    quote: str
    char_from: int = 0
    char_to: int = 0
    conditions: dict[str, str] = field(default_factory=dict)


def _num(value: str) -> float:
    return float(value.replace(",", "."))


def _si(value: float | None, unit: UnitDef) -> float | None:
    if value is None:
        return None
    return value * unit.factor + unit.offset


def _unit(raw: str) -> UnitDef:
    return UNITS[raw.lower()]


def _quote(text: str, start: int, end: int) -> str:
    left = text.rfind(".", 0, start)
    right = text.find(".", end)
    left = 0 if left < 0 else left + 1
    right = len(text) if right < 0 else right
    return text[left:right].strip()


def extract_facts(text: str) -> list[Fact]:
    text = normalize(text)
    facts: list[Fact] = []
    seen: set[tuple[int, int]] = set()

    def add(match: re.Match, operator: str, vmin, vmax, unit_raw: str) -> None:
        span = (match.start(), match.end())
        for existing in seen:
            if not (span[1] <= existing[0] or span[0] >= existing[1]):
                return
        seen.add(span)
        unit = _unit(unit_raw)
        facts.append(
            Fact(
                operator=operator,
                value_raw=match.group(0).strip(),
                vmin=vmin,
                vmax=vmax,
                unit_orig=unit_raw,
                unit_code=unit.code,
                vmin_si=_si(vmin, unit),
                vmax_si=_si(vmax, unit),
                parameter_slug=unit.parameter_slug,
                parameter_name=unit.parameter_name,
                quote=_quote(text, match.start(), match.end()),
                char_from=match.start(),
                char_to=match.end(),
            )
        )

    for match in _RANGE_WORDS.finditer(text):
        add(match, "range", _num(match.group(1)), _num(match.group(2)), match.group(3))
    for match in _RANGE.finditer(text):
        add(match, "range", _num(match.group(1)), _num(match.group(2)), match.group(3))
    for match in _PM.finditer(text):
        center, delta = _num(match.group(1)), _num(match.group(2))
        add(match, "range", center - delta, center + delta, match.group(3))
    for match in _UPPER.finditer(text):
        add(match, "lte", None, _num(match.group(1)), match.group(2))
    for match in _LOWER.finditer(text):
        add(match, "gte", _num(match.group(1)), None, match.group(2))
    for match in _EQ.finditer(text):
        value = _num(match.group(1))
        add(match, "eq", value, value, match.group(2))

    return facts
