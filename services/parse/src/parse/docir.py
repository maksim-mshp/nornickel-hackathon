import io
import re


def detect_format(raw: bytes) -> str:
    if raw[:4] == b"%PDF":
        return "pdf"
    if raw[:4] == b"PK\x03\x04":
        return "docx"
    return "text"


def extract_text(raw: bytes) -> tuple[str, str, int]:
    fmt = detect_format(raw)
    if fmt == "pdf":
        return _pdf_text(raw)
    if fmt == "docx":
        return _docx_text(raw), "docx", 1
    return raw.decode("utf-8", errors="ignore"), "text", 1


def _pdf_text(raw: bytes) -> tuple[str, str, int]:
    from pypdf import PdfReader

    reader = PdfReader(io.BytesIO(raw))
    parts = [page.extract_text() or "" for page in reader.pages]
    return "\n\n".join(parts), "pdf", len(reader.pages)


def _docx_text(raw: bytes) -> str:
    import docx

    document = docx.Document(io.BytesIO(raw))
    return "\n\n".join(paragraph.text for paragraph in document.paragraphs if paragraph.text.strip())


def detect_lang(text: str) -> str:
    cyrillic = len(re.findall(r"[а-яё]", text, re.IGNORECASE))
    latin = len(re.findall(r"[a-z]", text, re.IGNORECASE))
    if cyrillic == 0 and latin == 0:
        return "unknown"
    if cyrillic >= latin:
        return "ru"
    return "en"


def build_docir(document_id: str, text: str, source_format: str, pages: int) -> dict:
    blocks = []
    offset = 0
    for ordinal, paragraph in enumerate(_paragraphs(text)):
        start = text.find(paragraph, offset)
        if start < 0:
            start = offset
        end = start + len(paragraph)
        offset = end
        blocks.append(
            {
                "id": f"b{ordinal}",
                "kind": "paragraph",
                "page": 1,
                "section_path": [],
                "text": paragraph,
                "char_from": start,
                "char_to": end,
            }
        )
    return {
        "schema": "docir/1",
        "document_id": document_id,
        "version": 1,
        "lang": detect_lang(text),
        "source_format": source_format,
        "pages": pages,
        "blocks": blocks,
        "full_text": text,
    }


def _paragraphs(text: str) -> list[str]:
    chunks = [chunk.strip() for chunk in re.split(r"\n\s*\n", text)]
    result = [chunk for chunk in chunks if chunk]
    return result or ([text.strip()] if text.strip() else [])
