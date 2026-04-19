import type {
  AIEnrichment,
  MetadataFetchResponse,
  RemoteMetadata,
} from "../../types";
import { request } from "./core";

export async function fetchMetadata(
  bookId: string,
): Promise<MetadataFetchResponse> {
  return request<MetadataFetchResponse>(
    "POST",
    `/api/books/${bookId}/metadata/fetch`,
  );
}

export async function getMetadata(bookId: string): Promise<RemoteMetadata> {
  return request<RemoteMetadata>("GET", `/api/books/${bookId}/metadata`);
}

export async function rejectMetadata(bookId: string): Promise<void> {
  await request<void>("POST", `/api/books/${bookId}/metadata/reject`);
}

/**
 * Return the most recent pending AI enrichment for a book.
 */
export async function getPendingAIEnrichment(
  bookId: string,
): Promise<AIEnrichment> {
  return request<AIEnrichment>("GET", `/api/books/${bookId}/metadata/ai`);
}

/**
 * Enqueue an AI enrichment job for the given book.
 */
export async function fetchAIEnrichment(
  bookId: string,
): Promise<MetadataFetchResponse> {
  return request<MetadataFetchResponse>(
    "POST",
    `/api/books/${bookId}/metadata/ai-fetch`,
  );
}

/**
 * Apply the pending AI enrichment to the book (merges tags, optionally sets description).
 */
export async function applyAIEnrichment(bookId: string): Promise<AIEnrichment> {
  return request<AIEnrichment>(
    "POST",
    `/api/books/${bookId}/metadata/ai-apply`,
  );
}

/**
 * Reject the pending AI enrichment for the book.
 */
export async function rejectAIEnrichment(bookId: string): Promise<void> {
  await request<void>("POST", `/api/books/${bookId}/metadata/ai-reject`);
}

/**
 * Open an SSE connection for metadata fetch progress events.
 * Uses cookie-based auth since EventSource does not support custom headers.
 */
export function subscribeToMetadataEvents(bookId: string): EventSource {
  return new EventSource(`/api/books/${bookId}/metadata/events`);
}
