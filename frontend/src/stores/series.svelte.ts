import type { Series, SeriesInput } from "../types";
import * as api from "../lib/api";
import { CrudStore } from "./crudStore.svelte";

class SeriesStore extends CrudStore<Series, SeriesInput> {
  constructor() {
    super({
      list: api.listSeries,
      create: api.createSeries,
      update: api.updateSeries,
      delete: api.deleteSeries,
    });
  }

  get series(): Series[] {
    return this.items;
  }

  set series(v: Series[]) {
    this.items = v;
  }
}

export const seriesStore = new SeriesStore();
