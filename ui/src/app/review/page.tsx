import type { Metadata } from "next";
import { PageStub } from "@/shared/ui/page-stub";

export const metadata: Metadata = {
  title: "Ревью — kmap",
};

export default function ReviewPage() {
  return (
    <PageStub
      title="Очередь ревью"
      description="Три стопки для эксперта-валидатора с работой с клавиатуры и undo"
      plannedBlocks={[
        "сущности pending: merge / approve с кандидатами",
        "числа-сироты: цитата + кандидаты привязки",
        "противоречия suspected: вердикт судьи → подтвердить / отклонить / разрешить",
      ]}
    />
  );
}
