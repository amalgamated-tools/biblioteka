import type { Tag, TagInput } from "../types";
import * as api from "../lib/api";
import { CrudStore } from "./crudStore.svelte";

class TagStore extends CrudStore<Tag, TagInput> {
  constructor() {
    super({
      list: api.listTags,
      create: api.createTag,
      update: api.updateTag,
      delete: api.deleteTag,
    });
  }

  get tags(): Tag[] {
    return this.items;
  }

  set tags(v: Tag[]) {
    this.items = v;
  }
}

export const tagStore = new TagStore();
