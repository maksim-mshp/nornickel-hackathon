import type { Metadata } from "next";
import { PageStub } from "@/shared/ui/page-stub";

export const metadata: Metadata = {
  title: "Эксперты — kmap",
};

export default function ExpertsPage() {
  return (
    <PageStub
      title="Институциональная память"
      description="Поиск экспертов по теме с доказательной цепочкой: документы → эксперименты → годы"
      plannedBlocks={[
        "поиск по теме или сущности",
        "карточки: лаборатория, вес по теме, спарклайн активности",
        "evidence-цепочка (раскрывается)",
        "режим «карта»: биграф человек—тема",
      ]}
    />
  );
}
