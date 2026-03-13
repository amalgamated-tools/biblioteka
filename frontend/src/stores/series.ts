import { writable } from "svelte/store";
import type { Series, SeriesInput } from "../types";
import * as api from "../lib/api";

export const series = writable<Series[]>([]);
export const seriesLoading = writable(false);
export const seriesLoaded = writable(false);

export async function loadSeries(): Promise<void> {
  seriesLoading.set(true);
  try {
    const data = await api.listSeries();
    series.set(data);
    seriesLoaded.set(true);
  } catch {
    // Silently fail — individual pages can handle errors
  } finally {
    seriesLoading.set(false);
  }
}

export async function addSeries(input: SeriesInput): Promise<Series> {
  const created = await api.createSeries(input);
  series.update((list) => [...list, created]);
  return created;
}

export async function editSeries(id: string, input: SeriesInput): Promise<Series> {
  const updated = await api.updateSeries(id, input);
  series.update((list) => list.map((s) => (s.id === id ? updated : s)));
  return updated;
}

export async function removeSeries(id: string): Promise<void> {
  await api.deleteSeries(id);
  series.update((list) => list.filter((s) => s.id !== id));
}
