import { authHeaders } from "@/shared/lib/role";

function triggerDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 10_000);
}

export function downloadFile(
  filename: string,
  content: string,
  mime: string,
): void {
  triggerDownload(new Blob([content], { type: `${mime};charset=utf-8` }), filename);
}

function filenameFromDisposition(header: string | null, fallback: string): string {
  if (!header) return fallback;
  const utf8 = /filename\*=UTF-8''([^;]+)/i.exec(header);
  if (utf8?.[1]) {
    try {
      return decodeURIComponent(utf8[1]);
    } catch {
      // fall through to the ascii filename
    }
  }
  const plain = /filename="?([^";]+)"?/i.exec(header);
  return plain?.[1] ?? fallback;
}

export async function downloadDocumentSource(documentId: string): Promise<void> {
  const response = await fetch(
    `/v1/documents/${encodeURIComponent(documentId)}/file`,
    { headers: authHeaders() },
  );
  if (!response.ok) {
    throw new Error(
      response.status === 404
        ? "Оригинал документа недоступен"
        : `Не удалось скачать (${response.status})`,
    );
  }
  const blob = await response.blob();
  const filename = filenameFromDisposition(
    response.headers.get("Content-Disposition"),
    "document",
  );
  triggerDownload(blob, filename);
}
