import type { Metadata } from "next";
import { PageStub } from "@/shared/ui/page-stub";

export const metadata: Metadata = {
  title: "Эксперименты — kmap",
};

export default function ExperimentsPage() {
  return (
    <PageStub
      title="Каталог экспериментов"
      description="Плотная таблица экспериментов с фасетами, режимом сравнения и экспортом CSV"
      plannedBlocks={[
        "таблица: ID · материал · процесс · условия · результат · источник · confidence",
        "фасетные фильтры слева",
        "режим сравнения выбранных (до 4, различия подсвечены)",
        "экспорт CSV",
      ]}
    />
  );
}
