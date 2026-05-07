import type { PaginatedBooks } from "../../types";
import { request } from "./core";

type PaginatedListResponse<TItem, TKey extends string> = {
  total: number;
  limit: number;
  offset: number;
} & Record<TKey, TItem[]>;

/**
 * Fetches a paginated list of books for a given entity path.
 *
 * Pass the entity base path (e.g. `/api/authors/${id}`) and the pagination
 * parameters; the `/books` suffix and query string are appended here.
 */
export async function listEntityBooks(
  entityPath: string,
  limit: number,
  offset: number,
): Promise<PaginatedBooks> {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  return request<PaginatedBooks>(
    "GET",
    `${entityPath}/books?${params.toString()}`,
  );
}

export async function listAllPaginated<TItem, TKey extends string>(
  path: string,
  itemsKey: TKey,
  limit = 200,
): Promise<TItem[]> {
  const items: TItem[] = [];
  let offset = 0;

  for (;;) {
    const params = new URLSearchParams({
      limit: String(limit),
      offset: String(offset),
    });
    const response = await request<PaginatedListResponse<TItem, TKey>>(
      "GET",
      `${path}?${params.toString()}`,
    );
    const pageItems = response[itemsKey];

    items.push(...pageItems);
    if (pageItems.length === 0 || items.length >= response.total) {
      return items;
    }
    offset += pageItems.length;
  }
}
