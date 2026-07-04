import { formatFactValue } from "@/entities/fact/fact-value";
import type { AnswerDoc, EvidencePack } from "@/shared/api/types";
import { toCsv } from "@/shared/lib/csv";
import { downloadFile } from "@/shared/lib/download";

export function exportMarkdown(
  question: string,
  answer: AnswerDoc,
  pack: EvidencePack,
) {
  const total = Math.max(0, answer.guard.numbersChecked);
  const violations = Math.min(Math.max(0, answer.guard.violations), total);
  const verified = total - violations;
  const lines = [
    `# ${question}`,
    "",
    answer.summary,
    "",
    `> guard: ${verified}/${total} чисел сверены · confidence ${Math.round(answer.confidence * 100)}%`,
    "",
    "## Методы",
    ...answer.methods.map(
      (method) =>
        `- **${method.name}** — ${method.applicability} [${method.citations.join(", ")}]`,
    ),
    "",
    "## Evidence",
    "| Ref | Параметр | Значение | Источник | Стр. | Год | Статус |",
    "|---|---|---|---|---|---|---|",
    ...pack.facts.map(
      (fact) =>
        `| ${fact.ref} | ${fact.parameter.name} | ${formatFactValue(fact.value)} ${fact.value.unit} | ${fact.provenance.title} | ${fact.provenance.page} | ${fact.provenance.year} | ${fact.validationStatus} |`,
    ),
  ];
  downloadFile("kmap-answer.md", lines.join("\n"), "text/markdown");
}

export function exportCsv(pack: EvidencePack) {
  const header = [
    "ref",
    "subject",
    "parameter",
    "operator",
    "vmin",
    "vmax",
    "unit",
    "geography",
    "document",
    "page",
    "year",
    "validation",
    "score",
  ];
  const body = pack.facts.map((fact) => [
    fact.ref,
    fact.subject.name,
    fact.parameter.name,
    fact.value.operator,
    fact.value.vmin ?? "",
    fact.value.vmax ?? "",
    fact.value.unit,
    fact.geography,
    fact.provenance.title,
    fact.provenance.page,
    fact.provenance.year,
    fact.validationStatus,
    fact.score,
  ]);
  downloadFile("kmap-evidence.csv", toCsv([header, ...body]), "text/csv");
}

function legacyCopy(text: string): boolean {
  try {
    const area = document.createElement("textarea");
    area.value = text;
    area.style.position = "fixed";
    area.style.opacity = "0";
    document.body.appendChild(area);
    area.focus();
    area.select();
    const ok = document.execCommand("copy");
    area.remove();
    return ok;
  } catch {
    return false;
  }
}

export async function copyShareLink(question: string): Promise<boolean> {
  const url = `${window.location.origin}/?q=${encodeURIComponent(question)}`;
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url);
      return true;
    }
  } catch {
    return legacyCopy(url);
  }
  return legacyCopy(url);
}
