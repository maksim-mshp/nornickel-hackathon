"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { askStream } from "@/shared/api/ask-client";
import type {
  AnswerDoc,
  EvidencePack,
  QueryPlan,
} from "@/shared/api/types";

export type AskPhase =
  | "idle"
  | "planning"
  | "retrieving"
  | "streaming"
  | "done"
  | "error";

export type AskState = {
  phase: AskPhase;
  question: string;
  plan: QueryPlan | null;
  pack: EvidencePack | null;
  summaryText: string;
  answer: AnswerDoc | null;
  error: string | null;
};

const INITIAL: AskState = {
  phase: "idle",
  question: "",
  plan: null,
  pack: null,
  summaryText: "",
  answer: null,
  error: null,
};

export function useAsk() {
  const [state, setState] = useState<AskState>(INITIAL);
  const abortRef = useRef<AbortController | null>(null);

  const reset = useCallback(() => {
    abortRef.current?.abort();
    setState(INITIAL);
  }, []);

  const ask = useCallback((question: string) => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setState({ ...INITIAL, phase: "planning", question });

    (async () => {
      try {
        for await (const event of askStream(question, controller.signal)) {
          if (controller.signal.aborted) return;
          setState((prev) => {
            switch (event.type) {
              case "plan":
                return { ...prev, phase: "retrieving", plan: event.plan };
              case "evidence":
                return { ...prev, phase: "streaming", pack: event.pack };
              case "answer.delta":
                return {
                  ...prev,
                  summaryText: prev.summaryText + event.text,
                };
              case "answer.done":
                return { ...prev, phase: "done", answer: event.answer };
              case "error":
                return { ...prev, phase: "error", error: event.message };
              default:
                return prev;
            }
          });
        }
      } catch (err) {
        if ((err as Error).name === "AbortError") return;
        setState((prev) => ({
          ...prev,
          phase: "error",
          error: (err as Error).message,
        }));
      }
    })();
  }, []);

  useEffect(() => () => abortRef.current?.abort(), []);

  return { state, ask, reset };
}
