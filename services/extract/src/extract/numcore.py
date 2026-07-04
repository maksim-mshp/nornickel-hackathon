import pathlib
import re
from dataclasses import dataclass, field

import yaml

_REGISTRY_PATH = pathlib.Path(__file__).parent / "units.yml"


@dataclass(frozen=True)
class UnitDef:
    code: str
    factor: float
    offset: float
    dimension: str
    parameter_slug: str
    parameter_name: str


@dataclass(frozen=True)
class ParameterDef:
    slug: str
    name: str
    si_unit: str
    plausible_min: float | None
    plausible_max: float | None


def _load_registry() -> tuple[dict[str, UnitDef], dict[str, ParameterDef], list[str]]:
    data = yaml.safe_load(_REGISTRY_PATH.read_text(encoding="utf-8"))
    parameters: dict[str, ParameterDef] = {}
    for dimension, spec in data["parameters"].items():
        parameters[dimension] = ParameterDef(
            slug=spec["slug"],
            name=spec["name"],
            si_unit=spec["si_unit"],
            plausible_min=spec.get("plausible_min"),
            plausible_max=spec.get("plausible_max"),
        )

    units: dict[str, UnitDef] = {}
    aliases: list[str] = []
    for entry in data["units"]:
        dimension = entry["dimension"]
        parameter = parameters[dimension]
        unit = UnitDef(
            code=entry["code"],
            factor=float(entry["si_factor"]),
            offset=float(entry.get("si_offset", 0.0)),
            dimension=dimension,
            parameter_slug=parameter.slug,
            parameter_name=parameter.name,
        )
        for name in entry["names"]:
            key = name.lower()
            units[key] = unit
            aliases.append(key)
    aliases.sort(key=len, reverse=True)
    return units, parameters, aliases


UNITS, PARAMETERS, _ALIASES = _load_registry()

_UNIT = "(?:" + "|".join(re.escape(alias) for alias in _ALIASES) + ")"
_UNIT_TAIL = r"(?![0-9A-Za-zА-Яа-яЁё])"
_UNIT_GROUP = rf"({_UNIT}){_UNIT_TAIL}"
_NUM = (
    r"(?:[+-]?\d{1,3}(?:[ ]\d{3})+(?:[.,]\d+)?"
    r"|[+-]?\d+(?:[.,]\d+)?(?:[eE][+-]?\d+)?)"
)

_RANGE_WORDS = re.compile(rf"\bот\s*({_NUM})\s*до\s*({_NUM})\s*{_UNIT_GROUP}", re.IGNORECASE)
_RANGE_SEP = re.compile(rf"({_NUM})\s*(?:[…‥]|\.\.\.?|-)\s*({_NUM})\s*{_UNIT_GROUP}", re.IGNORECASE)
_PM = re.compile(rf"({_NUM})\s*±\s*({_NUM})\s*{_UNIT_GROUP}", re.IGNORECASE)
_APPROX = re.compile(
    rf"(?:≈|~|\bоколо|\bпорядка|\bпримерно|\bориентировочно)\s*({_NUM})\s*{_UNIT_GROUP}", re.IGNORECASE
)
_GTE = re.compile(
    rf"(?:≥|⩾|\bне\s+менее|\bне\s+ниже|\bне\s+меньше|\bминимум|\bот)\s*({_NUM})\s*{_UNIT_GROUP}",
    re.IGNORECASE,
)
_LTE = re.compile(
    rf"(?:≤|⩽|\bне\s+более|\bне\s+выше|\bне\s+больше|\bне\s+превыша\w*|\bмаксимум|\bдо)"
    rf"\s*({_NUM})\s*{_UNIT_GROUP}",
    re.IGNORECASE,
)
_GT = re.compile(rf"(?:>|\bсвыше|\bвыше|\bболее|\bбольше)\s*({_NUM})\s*{_UNIT_GROUP}", re.IGNORECASE)
_LT = re.compile(rf"(?:<|\bменее|\bниже|\bменьше)\s*({_NUM})\s*{_UNIT_GROUP}", re.IGNORECASE)
_PH_PREFIX = re.compile(rf"\b(?:pH|рН)\s*[:=]?\s*({_NUM})(?![0-9A-Za-zА-Яа-яЁё])", re.IGNORECASE)
_EQ = re.compile(rf"({_NUM})\s*{_UNIT_GROUP}", re.IGNORECASE)

_STOP_CONTEXT = re.compile(
    r"(?:\b(?:рис(?:унок|унке|\.)?|табл(?:ица|ице|\.)?|формул\w*|уравнени\w*|гост(?:\s*р)?|ост|ту|"
    r"санпин|снип|iso|мэк|iec|din|astm|патент\w*|patent|doi|стр(?:аница|\.)?|пункт|глава|раздел|"
    r"позици\w*|образец|проба|no)|№|\bп\.)\s*\.?\s*$",
    re.IGNORECASE,
)

_SCIENTIFIC = re.compile(
    r"(\d+(?:[.,]\d+)?)\s*[·*×]\s*10\s*(?:\^|\*\*)?\s*([⁻⁺+-]?)([⁰¹²³⁴⁵⁶⁷⁸⁹]+|\d+)"
)
_SUPERSCRIPT = str.maketrans("⁰¹²³⁴⁵⁶⁷⁸⁹⁻⁺", "0123456789-+")

_SPACE_CHARS = "        "
_DASH_CHARS = "–—−‒―‐‑"


def _scientific(match: re.Match) -> str:
    mantissa = match.group(1)
    sign = match.group(2).translate(_SUPERSCRIPT)
    exponent = match.group(3).translate(_SUPERSCRIPT)
    return f"{mantissa}e{sign}{exponent}"


def normalize(text: str) -> str:
    for char in _SPACE_CHARS:
        text = text.replace(char, " ")
    for char in _DASH_CHARS:
        text = text.replace(char, "-")
    text = text.replace("º", "°")
    text = text.replace("μ", "µ")
    text = _SCIENTIFIC.sub(_scientific, text)
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
    confidence: float = 0.97
    flags: list[str] = field(default_factory=list)
    conditions: dict[str, str] = field(default_factory=dict)


def _num(value: str) -> float:
    return float(value.replace(" ", "").replace(",", "."))


def _si(value: float | None, unit: UnitDef) -> float | None:
    if value is None:
        return None
    return value * unit.factor + unit.offset


def _unit(raw: str) -> UnitDef | None:
    return UNITS.get(raw.lower())


def _quote(text: str, start: int, end: int) -> str:
    left = text.rfind(".", 0, start)
    right = text.find(".", end)
    left = 0 if left < 0 else left + 1
    right = len(text) if right < 0 else right
    return text[left:right].strip()


def _stopped(text: str, start: int) -> bool:
    return _STOP_CONTEXT.search(text[max(0, start - 30) : start]) is not None


def _sanity(vmin_si: float | None, vmax_si: float | None, parameter: ParameterDef) -> bool:
    lo, hi = parameter.plausible_min, parameter.plausible_max
    if lo is None and hi is None:
        return True
    tol = 1e-6
    for value in (vmin_si, vmax_si):
        if value is None:
            continue
        if lo is not None and value < lo - abs(lo) * tol - tol:
            return False
        if hi is not None and value > hi + abs(hi) * tol + tol:
            return False
    return True


def extract_facts(text: str) -> list[Fact]:
    text = normalize(text)
    facts: list[Fact] = []
    spans: list[tuple[int, int]] = []

    def add(match: re.Match, operator: str, vmin, vmax, unit_raw: str) -> None:
        span = (match.start(), match.end())
        for existing in spans:
            if not (span[1] <= existing[0] or span[0] >= existing[1]):
                return
        if _stopped(text, span[0]):
            return
        spans.append(span)
        unit = _unit(unit_raw)
        if unit is None:
            return
        parameter = PARAMETERS[unit.dimension]
        vmin_si = _si(vmin, unit)
        vmax_si = _si(vmax, unit)
        flags: list[str] = []
        confidence = 0.97
        if not _sanity(vmin_si, vmax_si, parameter):
            flags.append("implausible")
            confidence = 0.5
        facts.append(
            Fact(
                operator=operator,
                value_raw=match.group(0).strip(),
                vmin=vmin,
                vmax=vmax,
                unit_orig=unit_raw,
                unit_code=unit.code,
                vmin_si=vmin_si,
                vmax_si=vmax_si,
                parameter_slug=unit.parameter_slug,
                parameter_name=unit.parameter_name,
                quote=_quote(text, span[0], span[1]),
                char_from=span[0],
                char_to=span[1],
                confidence=confidence,
                flags=flags,
            )
        )

    for match in _RANGE_WORDS.finditer(text):
        add(match, "range", _num(match.group(1)), _num(match.group(2)), match.group(3))
    for match in _RANGE_SEP.finditer(text):
        add(match, "range", _num(match.group(1)), _num(match.group(2)), match.group(3))
    for match in _PM.finditer(text):
        center, delta = _num(match.group(1)), _num(match.group(2))
        add(match, "range", center - delta, center + delta, match.group(3))
    for match in _APPROX.finditer(text):
        value = _num(match.group(1))
        add(match, "approx", value, value, match.group(2))
    for match in _GTE.finditer(text):
        add(match, "gte", _num(match.group(1)), None, match.group(2))
    for match in _LTE.finditer(text):
        add(match, "lte", None, _num(match.group(1)), match.group(2))
    for match in _GT.finditer(text):
        add(match, "gt", _num(match.group(1)), None, match.group(2))
    for match in _LT.finditer(text):
        add(match, "lt", None, _num(match.group(1)), match.group(2))
    for match in _PH_PREFIX.finditer(text):
        value = _num(match.group(1))
        add(match, "eq", value, value, "pH")
    for match in _EQ.finditer(text):
        value = _num(match.group(1))
        add(match, "eq", value, value, match.group(2))

    facts.sort(key=lambda fact: fact.char_from)
    return facts
