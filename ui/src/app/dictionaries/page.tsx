import type { Metadata } from "next";
import { PageStub } from "@/shared/ui/page-stub";

export const metadata: Metadata = {
  title: "Словари — kmap",
};

export default function DictionariesPage() {
  return (
    <PageStub
      title="Словари"
      description="Синонимы сущностей и реестр единиц измерения с живым тестом парсинга"
      plannedBlocks={[
        "синонимы: canonical ← алиасы (язык, источник, статус)",
        "единицы: код, написания, размерность, SI-фактор",
        "тест-строка «распарсить пример» через numcore",
        "версии правок с автором",
      ]}
    />
  );
}
