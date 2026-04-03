export interface CrudOps<T, TInput> {
  list: () => Promise<T[]>;
  create: (input: TInput) => Promise<T>;
  update: (id: string, input: TInput) => Promise<T>;
  delete: (id: string) => Promise<void>;
}

export class CrudStore<T extends { id: string }, TInput> {
  items: T[] = $state.raw([]);
  loading = $state(false);
  loaded = $state(false);

  private readonly ops: CrudOps<T, TInput>;

  constructor(ops: CrudOps<T, TInput>) {
    this.ops = ops;
  }

  async load(): Promise<void> {
    if (this.loading || this.loaded) return;
    this.loading = true;
    try {
      const data = await this.ops.list();
      this.items = data;
      this.loaded = true;
    } catch {
      // Silently fail — individual pages can handle errors
    } finally {
      this.loading = false;
    }
  }

  async add(input: TInput): Promise<T> {
    const created = await this.ops.create(input);
    this.items = [...this.items, created];
    return created;
  }

  async edit(id: string, input: TInput): Promise<T> {
    const updated = await this.ops.update(id, input);
    this.items = this.items.map((item) => (item.id === id ? updated : item));
    return updated;
  }

  async remove(id: string): Promise<void> {
    await this.ops.delete(id);
    this.items = this.items.filter((item) => item.id !== id);
  }
}
