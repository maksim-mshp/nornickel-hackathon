import type { AskEvent } from "@/shared/api/types";
import {
  CATHOLYTE_ANSWER,
  CATHOLYTE_PACK,
  CATHOLYTE_PLAN,
  CATHOLYTE_SUMMARY,
} from "@/shared/api/mock/catholyte-scenario";

const PLAN_DELAY_MS = 600;
const EVIDENCE_DELAY_MS = 900;
const DELTA_DELAY_MS = 90;
const DELTA_WORDS = 8;
const DONE_DELAY_MS = 400;

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(resolve, ms);
    signal?.addEventListener("abort", () => {
      clearTimeout(timer);
      reject(new DOMException("aborted", "AbortError"));
    });
  });
}

function splitDeltas(text: string): string[] {
  const words = text.match(/\S+\s*/g) ?? [];
  const chunks: string[] = [];
  for (let i = 0; i < words.length; i += DELTA_WORDS) {
    chunks.push(words.slice(i, i + DELTA_WORDS).join(""));
  }
  return chunks;
}

export async function* askStream(
  question: string,
  signal?: AbortSignal,
): AsyncGenerator<AskEvent> {
  void question;
  await sleep(PLAN_DELAY_MS, signal);
  yield { type: "plan", plan: CATHOLYTE_PLAN };
  await sleep(EVIDENCE_DELAY_MS, signal);
  yield { type: "evidence", pack: CATHOLYTE_PACK };
  for (const delta of splitDeltas(CATHOLYTE_SUMMARY)) {
    await sleep(DELTA_DELAY_MS, signal);
    yield { type: "answer.delta", text: delta };
  }
  await sleep(DONE_DELAY_MS, signal);
  yield { type: "answer.done", answer: CATHOLYTE_ANSWER };
}
