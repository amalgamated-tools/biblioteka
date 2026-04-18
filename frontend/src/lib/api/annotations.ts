import type { BookAnnotation, BookAnnotationInput } from "../../types";
import { request } from "./core";

export async function listBookAnnotations(
  bookId: string,
): Promise<BookAnnotation[]> {
  return request<BookAnnotation[]>("GET", `/api/books/${bookId}/annotations`);
}

export async function createAnnotation(
  bookId: string,
  input: BookAnnotationInput,
): Promise<BookAnnotation> {
  return request<BookAnnotation>(
    "POST",
    `/api/books/${bookId}/annotations`,
    input,
  );
}

export async function getAnnotation(id: string): Promise<BookAnnotation> {
  return request<BookAnnotation>("GET", `/api/annotations/${id}`);
}

export async function updateAnnotation(
  id: string,
  input: BookAnnotationInput,
): Promise<BookAnnotation> {
  return request<BookAnnotation>("PUT", `/api/annotations/${id}`, input);
}

export async function deleteAnnotation(id: string): Promise<void> {
  await request<void>("DELETE", `/api/annotations/${id}`);
}
