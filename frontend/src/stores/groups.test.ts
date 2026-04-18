import { describe, it, expect, vi, beforeEach } from "vitest";
import * as api from "../lib/api";
import type { ReadingGroup } from "../types";

vi.mock("../lib/api", () => ({
  listGroups: vi.fn(),
  createGroup: vi.fn(),
  updateGroup: vi.fn(),
  deleteGroup: vi.fn(),
}));

// Import after mocking
const { groupStore } = await import("./groups.svelte");

const fakeGroup: ReadingGroup = {
  id: "g-1",
  owner_id: "u-1",
  name: "Book Club",
  description: null,
  member_count: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("groupStore", () => {
  beforeEach(() => {
    groupStore.groups = [];
    groupStore.loaded = false;
    groupStore.loading = false;
    groupStore.loadError = null;
    vi.clearAllMocks();
  });

  it("loads groups from the API", async () => {
    vi.mocked(api.listGroups).mockResolvedValue([fakeGroup]);

    await groupStore.load();

    expect(api.listGroups).toHaveBeenCalledOnce();
    expect(groupStore.groups).toEqual([fakeGroup]);
    expect(groupStore.loaded).toBe(true);
    expect(groupStore.loadError).toBeNull();
  });

  it("does not re-load when already loaded", async () => {
    vi.mocked(api.listGroups).mockResolvedValue([fakeGroup]);
    await groupStore.load();

    await groupStore.load();

    expect(api.listGroups).toHaveBeenCalledOnce();
  });

  it("sets loadError on failure", async () => {
    vi.mocked(api.listGroups).mockRejectedValue(new Error("network error"));

    await groupStore.load();

    expect(groupStore.loadError).toBe("network error");
    expect(groupStore.loaded).toBe(true);
  });

  it("creates a group and inserts sorted", async () => {
    groupStore.groups = [fakeGroup];
    groupStore.loaded = true;
    const newGroup: ReadingGroup = {
      ...fakeGroup,
      id: "g-2",
      name: "Another Club",
    };
    vi.mocked(api.createGroup).mockResolvedValue(newGroup);

    const result = await groupStore.create({ name: "Another Club" });

    expect(result).toEqual(newGroup);
    expect(groupStore.groups[0].name).toBe("Another Club");
    expect(groupStore.groups[1].name).toBe("Book Club");
  });

  it("updates a group in the list", async () => {
    groupStore.groups = [fakeGroup];
    groupStore.loaded = true;
    const updated: ReadingGroup = { ...fakeGroup, name: "Updated Club" };
    vi.mocked(api.updateGroup).mockResolvedValue(updated);

    await groupStore.update("g-1", { name: "Updated Club", description: null });

    expect(groupStore.groups[0].name).toBe("Updated Club");
  });

  it("removes a group from the list", async () => {
    groupStore.groups = [fakeGroup];
    groupStore.loaded = true;
    vi.mocked(api.deleteGroup).mockResolvedValue(undefined);

    await groupStore.remove("g-1");

    expect(groupStore.groups).toHaveLength(0);
  });

  it("adjustMemberCount increments count for the given group", () => {
    groupStore.groups = [fakeGroup];

    groupStore.adjustMemberCount("g-1", 1);

    expect(groupStore.groups[0].member_count).toBe(2);
  });

  it("adjustMemberCount decrements count for the given group", () => {
    groupStore.groups = [{ ...fakeGroup, member_count: 3 }];

    groupStore.adjustMemberCount("g-1", -1);

    expect(groupStore.groups[0].member_count).toBe(2);
  });

  it("adjustMemberCount does not modify other groups", () => {
    const other: ReadingGroup = { ...fakeGroup, id: "g-2", name: "Other" };
    groupStore.groups = [fakeGroup, other];

    groupStore.adjustMemberCount("g-2", 1);

    expect(groupStore.groups[0].member_count).toBe(1);
    expect(groupStore.groups[1].member_count).toBe(2);
  });

  it("reload forces a fresh fetch", async () => {
    vi.mocked(api.listGroups).mockResolvedValue([fakeGroup]);
    groupStore.loaded = true;

    await groupStore.reload();

    expect(api.listGroups).toHaveBeenCalledOnce();
    expect(groupStore.groups).toEqual([fakeGroup]);
  });
});
