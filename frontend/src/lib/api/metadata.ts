import type { RemoteMetadata } from "../../types";
import { request } from "./core";

export async function fetchMetadata(
  bookId: string,
): Promise<{ task_id: string }> {
  return request<{ task_id: string }>(
    "POST",
    `/api/books/${bookId}/metadata/fetch`,
  );
}

export async function getMetadata(bookId: string): Promise<RemoteMetadata> {
  return request<RemoteMetadata>("GET", `/api/books/${bookId}/metadata`);
}

export async function applyMetadata(bookId: string): Promise<unknown> {
  return request<unknown>("POST", `/api/books/${bookId}/metadata/apply`);
}

export async function rejectMetadata(bookId: string): Promise<void> {
  await request<void>("POST", `/api/books/${bookId}/metadata/reject`);
}

/**
 * Open an SSE connection for metadata fetch progress events.
 * Uses cookie-based auth since EventSource does not support custom headers.
 */
export function subscribeToMetadataEvents(bookId: string): EventSource {
  return new EventSource(`/api/books/${bookId}/metadata/events`);
}
