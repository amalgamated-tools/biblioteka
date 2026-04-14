import type { CalibrePreview, CalibreImportResult } from "../../types";
import { requestFormData } from "./core";

export async function previewCalibreImport(
  file: File,
): Promise<CalibrePreview> {
  const form = new FormData();
  form.append("metadata_db", file);
  return requestFormData<CalibrePreview>(
    "POST",
    "/api/calibre-import/preview",
    form,
  );
}

export async function confirmCalibreImport(
  file: File,
  libraryId: string,
): Promise<CalibreImportResult> {
  const form = new FormData();
  form.append("metadata_db", file);
  if (libraryId) {
    form.append("library_id", libraryId);
  }
  return requestFormData<CalibreImportResult>(
    "POST",
    "/api/calibre-import/confirm",
    form,
  );
}
