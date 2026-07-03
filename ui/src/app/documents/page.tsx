import type { Metadata } from "next";
import { PageStub } from "@/shared/ui/page-stub";

export const metadata: Metadata = {
  title: "Документы — kmap",
};

export default function DocumentsPage() {
  return (
    <PageStub
      title="Реестр документов и конвейер"
      description="Загрузка корпуса и статусы обработки: registered → parsed → extracted → indexed"
      plannedBlocks={[
        "dropzone: множественная загрузка + манифест",
        "таблица: тип · язык · география · access · статус конвейера · факты",
        "паспорт документа: версии, метрики качества экстракции, reindex",
      ]}
    />
  );
}
