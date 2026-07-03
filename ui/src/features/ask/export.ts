import { formatFactValue } from "@/entities/fact/fact-value";
import type { AnswerDoc, EvidencePack } from "@/shared/api/types";

function download(filename: string, mime: string, content: string) {
  const blob = new Blob([content], { type: `${mime};charset=utf-8` });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

export function exportMarkdown(
  question: string,
  answer: AnswerDoc,
  pack: EvidencePack,
) {
  const lines = [
    `# ${question}`,
    "",
    answer.summary,
    "",
    `> guard: ${answer.guard.numbersChecked}/${answer.guard.numbersChecked} чисел сверены · confidence ${Math.round(answer.confidence * 100)}%`,
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
  download("kmap-answer.md", "text/markdown", lines.join("\n"));
}

export function exportCsv(pack: EvidencePack) {
  const header =
    "ref;subject;parameter;operator;vmin;vmax;unit;geography;document;page;year;validation;score";
  const rows = pack.facts.map((fact) =>
    [
      fact.ref,
      fact.subject.name,
      fact.parameter.name,
      fact.value.operator,
      fact.value.vmin ?? "",
      fact.value.vmax ?? "",
      fact.value.unit,
      fact.geography,
      `"${fact.provenance.title}"`,
      fact.provenance.page,
      fact.provenance.year,
      fact.validationStatus,
      fact.score,
    ].join(";"),
  );
  download("kmap-evidence.csv", "text/csv", [header, ...rows].join("\n"));
}

export async function copyShareLink(question: string): Promise<boolean> {
  const url = `${window.location.origin}/?q=${encodeURIComponent(question)}`;
  try {
    await navigator.clipboard.writeText(url);
    return true;
  } catch {
    return false;
  }
}
