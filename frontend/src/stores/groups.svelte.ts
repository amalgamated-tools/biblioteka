import type {
  ReadingGroup,
  ReadingGroupInput,
  ReadingGroupUpdateInput,
} from "../types";
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

  async update(
    id: string,
    input: ReadingGroupUpdateInput,
  ): Promise<ReadingGroup> {
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

  /** Sync the cached member_count for a group to an exact value. */
  setMemberCount(id: string, count: number): void {
    this.groups = this.groups.map((g) =>
      g.id === id ? { ...g, member_count: count } : g,
    );
  }

  /** Adjust the cached member_count for a group by a delta (+1 or -1).
   *  Used to avoid a full reload after adding or removing a single member.
   */
  adjustMemberCount(id: string, delta: number): void {
    this.groups = this.groups.map((g) =>
      g.id === id
        ? { ...g, member_count: Math.max(0, g.member_count + delta) }
        : g,
    );
  }
}

export const groupStore = new GroupStore();
