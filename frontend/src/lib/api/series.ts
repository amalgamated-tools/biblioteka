import type { Series, SeriesInput, PaginatedBooks } from "../../types";
import { request } from "./core";
import { listEntityBooks } from "./pagination";

interface SeriesListResponse {
  series: Series[];
  total: number;
  limit: number;
  offset: number;
}

export async function listSeries(): Promise<Series[]> {
  const result = await request<SeriesListResponse>("GET", "/api/series");
  return result.series;
}

export async function getSeries(id: string): Promise<Series> {
  return request<Series>("GET", `/api/series/${id}`);
}

export async function createSeries(input: SeriesInput): Promise<Series> {
  return request<Series>("POST", "/api/series", input);
}

export async function updateSeries(
  id: string,
  input: SeriesInput,
): Promise<Series> {
  return request<Series>("PUT", `/api/series/${id}`, input);
}

export async function deleteSeries(id: string): Promise<void> {
  await request<void>("DELETE", `/api/series/${id}`);
}

export async function listSeriesBooks(
  seriesId: string,
  limit = 50,
  offset = 0,
): Promise<PaginatedBooks> {
  return listEntityBooks(`/api/series/${seriesId}`, limit, offset);
}
