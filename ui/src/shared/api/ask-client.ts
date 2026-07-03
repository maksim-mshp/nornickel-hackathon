import type { AnswerDoc, AskEvent, EvidencePack, QueryPlan } from "@/shared/api/types";
import {
  CATHOLYTE_ANSWER,
  CATHOLYTE_PACK,
  CATHOLYTE_PLAN,
  CATHOLYTE_SUMMARY,
} from "@/shared/api/mock/catholyte-scenario";
import { authHeaders } from "@/shared/lib/role";

const ASK_ENDPOINT = "/v1/ask";
const MAX_CONNECT_RETRIES = 2;
const RETRY_BASE_MS = 400;

async function connectAsk(question: string, signal?: AbortSignal): Promise<Response | null> {
  for (let attempt = 0; ; attempt++) {
    try {
      const response = await fetch(ASK_ENDPOINT, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "text/event-stream",
          ...authHeaders(),
        },
        body: JSON.stringify({ question }),
        signal,
      });
      if (response.ok && response.body) return response;
      if (attempt >= MAX_CONNECT_RETRIES) return null;
    } catch (error) {
      if ((error as Error).name === "AbortError") throw error;
      if (attempt >= MAX_CONNECT_RETRIES) return null;
    }
    await sleep(RETRY_BASE_MS * 2 ** attempt, signal);
  }
}

export async function* askStream(
  question: string,
  signal?: AbortSignal,
): AsyncGenerator<AskEvent> {
  const response = await connectAsk(question, signal);
  if (!response?.body) {
    yield* mockAskStream(question, signal);
    return;
  }
  yield* parseSSE(response.body, signal);
}

async function* parseSSE(
  body: ReadableStream<Uint8Array>,
  signal?: AbortSignal,
): AsyncGenerator<AskEvent> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });

      let boundary = buffer.indexOf("\n\n");
      while (boundary !== -1) {
        const frame = buffer.slice(0, boundary);
        buffer = buffer.slice(boundary + 2);
        const event = parseFrame(frame);
        if (event) yield event;
        boundary = buffer.indexOf("\n\n");
      }
    }
  } finally {
    if (signal?.aborted) await reader.cancel().catch(() => {});
  }
}

function parseFrame(frame: string): AskEvent | null {
  let event = "";
  const dataLines: string[] = [];
  for (const line of frame.split("\n")) {
    if (line.startsWith("event:")) {
      event = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trimStart());
    }
  }
  if (!event || dataLines.length === 0) return null;

  const data = JSON.parse(dataLines.join("\n"));
  switch (event) {
    case "plan":
      return { type: "plan", plan: data as QueryPlan };
    case "evidence":
      return { type: "evidence", pack: data as EvidencePack };
    case "answer.delta":
      return { type: "answer.delta", text: String(data.text ?? "") };
    case "answer.done":
      return { type: "answer.done", answer: data as AnswerDoc };
    case "error":
      return {
        type: "error",
        message: String(data.detail ?? data.title ?? data.message ?? "Ошибка ответа"),
      };
    default:
      return null;
  }
}

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

export async function* mockAskStream(
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
