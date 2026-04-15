import type { CalibrePreview, CalibreImportResult } from "../../types";
import { request, requestFormData } from "./core";

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

export async function previewCalibreImportFromPath(
  path: string,
): Promise<CalibrePreview> {
  return request<CalibrePreview>("POST", "/api/calibre-import/preview", {
    path,
  });
}

export async function confirmCalibreImportFromPath(
  path: string,
  libraryId: string,
): Promise<CalibreImportResult> {
  return request<CalibreImportResult>("POST", "/api/calibre-import/confirm", {
    path,
    ...(libraryId ? { library_id: libraryId } : {}),
  });
}
