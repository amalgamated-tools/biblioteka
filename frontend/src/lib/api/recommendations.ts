import type { BookSummary } from "../../types";
import { request } from "./core";

/**
 * Returns scored book recommendations for the authenticated user.
 * Books are ranked by author overlap, series continuation, publisher match,
 * and download popularity. When no reading history exists the most-recently-
 * added books are returned as a fallback.
 * @param limit - Max number of recommendations (default 10, max 50).
 * @param signal - Optional AbortSignal to cancel the in-flight request.
 */
export async function getRecommendations(
  limit = 10,
  signal?: AbortSignal,
): Promise<BookSummary[]> {
  const params = new URLSearchParams({ limit: String(limit) });
  return request<BookSummary[]>(
    "GET",
    `/api/recommendations?${params.toString()}`,
    undefined,
    signal,
  );
}
