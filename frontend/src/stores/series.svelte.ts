import type { Series, SeriesInput } from "../types";
import * as api from "../lib/api";

class SeriesStore {
  series: Series[] = $state.raw([]);
  loading = $state(false);
  loaded = $state(false);

  async load(): Promise<void> {
    if (this.loading || this.loaded) return;
    this.loading = true;
    try {
      const data = await api.listSeries();
      this.series = data;
      this.loaded = true;
    } catch {
      // Silently fail — individual pages can handle errors
    } finally {
      this.loading = false;
    }
  }

  async add(input: SeriesInput): Promise<Series> {
    const created = await api.createSeries(input);
    this.series = [...this.series, created];
    return created;
  }

  async edit(id: string, input: SeriesInput): Promise<Series> {
    const updated = await api.updateSeries(id, input);
    this.series = this.series.map((s) => (s.id === id ? updated : s));
    return updated;
  }

  async remove(id: string): Promise<void> {
    await api.deleteSeries(id);
    this.series = this.series.filter((s) => s.id !== id);
  }
}

export const seriesStore = new SeriesStore();
