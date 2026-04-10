import type { Author, AuthorInput } from "../types";
import * as api from "../lib/api";
import { CrudStore } from "./crudStore.svelte";

class AuthorStore extends CrudStore<Author, AuthorInput> {
  constructor() {
    super({
      list: api.listAuthors,
      create: api.createAuthor,
      update: api.updateAuthor,
      delete: api.deleteAuthor,
    });
  }

  get authors(): Author[] {
    return this.items;
  }

  set authors(v: Author[]) {
    this.items = v;
  }
}

export const authorStore = new AuthorStore();
