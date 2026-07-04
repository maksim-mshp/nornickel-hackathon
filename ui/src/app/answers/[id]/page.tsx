import type { Metadata } from "next";
import { SavedAnswerView } from "./saved-answer-view";

export const metadata: Metadata = {
  title: "Сохранённый ответ — kmap",
};

export default async function AnswerPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <SavedAnswerView id={id} />;
}
