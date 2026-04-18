import type { ReadingGroup, ReadingGroupInput } from "../types";
import * as api from "../lib/api";

class GroupStore {
  groups: ReadingGroup[] = $state.raw([]);
  loading = $state(false);
  loaded = $state(false);
  loadError: string | null = $state(null);

  async load(): Promise<void> {
    if (this.loading || this.loaded) return;
    this.loading = true;
    this.loadError = null;
    try {
      this.groups = await api.listGroups();
      this.loaded = true;
    } catch (err) {
      this.loadError =
        err instanceof Error ? err.message : "failed to load groups";
      this.loaded = true;
    } finally {
      this.loading = false;
    }
  }

  /** Force a reload from the server, even if already loaded. */
  async reload(): Promise<void> {
    this.loaded = false;
    this.loading = false;
    this.loadError = null;
    return this.load();
  }

  async create(input: ReadingGroupInput): Promise<ReadingGroup> {
    const created = await api.createGroup(input);
    this.groups = [...this.groups, created].sort((a, b) =>
      a.name.localeCompare(b.name),
    );
    return created;
  }

  async update(id: string, input: ReadingGroupInput): Promise<ReadingGroup> {
    const updated = await api.updateGroup(id, input);
    this.groups = this.groups
      .map((g) => (g.id === id ? updated : g))
      .sort((a, b) => a.name.localeCompare(b.name));
    return updated;
  }

  async remove(id: string): Promise<void> {
    await api.deleteGroup(id);
    this.groups = this.groups.filter((g) => g.id !== id);
  }
}

export const groupStore = new GroupStore();
