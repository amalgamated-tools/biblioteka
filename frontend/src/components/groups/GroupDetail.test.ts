import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  listGroupMembers: vi.fn(),
  addGroupMember: vi.fn(),
  removeGroupMember: vi.fn(),
  listGroupReadingLists: vi.fn(),
  shareListWithGroup: vi.fn(),
  unshareListFromGroup: vi.fn(),
}));

vi.mock("../../stores/groups.svelte", () => ({
  groupStore: {
    loaded: true,
    loading: false,
    loadError: null as string | null,
    groups: [] as ReadingGroup[],
    load: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
    setMemberCount: vi.fn(),
    adjustMemberCount: vi.fn(),
  },
}));

vi.mock("../../stores/reading-lists.svelte", () => ({
  readingListStore: {
    loaded: true,
    loading: false,
    lists: [] as ReadingList[],
    load: vi.fn(),
  },
}));

vi.mock("../../stores/auth.svelte", () => ({
  authStore: {
    user: null as { id: string } | null,
  },
}));

vi.mock("../../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

vi.mock("lucide-svelte", () => ({
  ArrowLeft: () => null,
  Pencil: () => null,
  Trash2: () => null,
  Users: () => null,
  BookMarked: () => null,
  UserPlus: () => null,
  UserMinus: () => null,
  X: () => null,
  Check: () => null,
  Share2: () => null,
}));

import type {
  ReadingGroup,
  ReadingGroupMember,
  ReadingList,
} from "../../types";
import {
  listGroupMembers,
  addGroupMember,
  removeGroupMember,
  listGroupReadingLists,
  shareListWithGroup,
  unshareListFromGroup,
} from "../../lib/api";
import { groupStore } from "../../stores/groups.svelte";
import { readingListStore } from "../../stores/reading-lists.svelte";
import { authStore } from "../../stores/auth.svelte";
import { routerStore } from "../../stores/router.svelte";
import GroupDetail from "./GroupDetail.svelte";

const OWNER_ID = "u-owner";

const fakeGroup: ReadingGroup = {
  id: "g-1",
  owner_id: OWNER_ID,
  name: "Book Club",
  description: "A great club",
  member_count: 2,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeMember: ReadingGroupMember = {
  group_id: "g-1",
  user_id: "u-2",
  user_name: "alice",
  role: "member",
  joined_at: "2026-01-02T00:00:00Z",
};

const fakeList: ReadingList = {
  id: "rl-1",
  name: "To Read",
  description: null,
  book_count: 5,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

function setCurrentUser(id: string) {
  (authStore as unknown as { user: { id: string } }).user = { id };
}

describe("GroupDetail", () => {
  beforeEach(() => {
    vi.mocked(groupStore).loaded = true;
    vi.mocked(groupStore).loading = false;
    vi.mocked(groupStore).loadError = null;
    (vi.mocked(groupStore) as unknown as { groups: ReadingGroup[] }).groups = [
      fakeGroup,
    ];
    vi.mocked(groupStore).load = vi.fn();
    vi.mocked(groupStore).update = vi
      .fn()
      .mockResolvedValue({ ...fakeGroup, name: "Updated" });
    vi.mocked(groupStore).remove = vi.fn().mockResolvedValue(undefined);
    vi.mocked(groupStore).setMemberCount = vi.fn();

    (
      vi.mocked(readingListStore) as unknown as {
        loaded: boolean;
        loading: boolean;
        lists: ReadingList[];
      }
    ).loaded = true;
    (
      vi.mocked(readingListStore) as unknown as {
        loaded: boolean;
        loading: boolean;
        lists: ReadingList[];
      }
    ).loading = false;
    (
      vi.mocked(readingListStore) as unknown as {
        loaded: boolean;
        loading: boolean;
        lists: ReadingList[];
      }
    ).lists = [];
    vi.mocked(readingListStore).load = vi.fn();

    setCurrentUser(OWNER_ID);

    vi.mocked(listGroupMembers).mockResolvedValue([fakeMember]);
    vi.mocked(listGroupReadingLists).mockResolvedValue([]);
    vi.mocked(addGroupMember).mockResolvedValue(undefined);
    vi.mocked(removeGroupMember).mockResolvedValue(undefined);
    vi.mocked(shareListWithGroup).mockResolvedValue(undefined);
    vi.mocked(unshareListFromGroup).mockResolvedValue(undefined);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  // ── Rendering ─────────────────────────────────────────────────────────────

  it("renders the group name and description", async () => {
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    expect(
      screen.getByRole("heading", { level: 1, name: "Book Club" }),
    ).toBeInTheDocument();
    expect(screen.getByText("A great club")).toBeInTheDocument();
  });

  it("shows an error when the group is not in the store", async () => {
    (vi.mocked(groupStore) as unknown as { groups: ReadingGroup[] }).groups =
      [];

    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    expect(await screen.findByText(/Group not found/)).toBeInTheDocument();
  });

  it("triggers groupStore.load when not already loaded", async () => {
    vi.mocked(groupStore).loaded = false;
    vi.mocked(groupStore).loading = false;

    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    expect(groupStore.load).toHaveBeenCalledOnce();
  });

  it("fetches members on mount", async () => {
    render(GroupDetail, { props: { groupId: "g-1" } });

    await waitFor(() => {
      expect(listGroupMembers).toHaveBeenCalledWith("g-1");
    });
    expect(await screen.findByText("alice")).toBeInTheDocument();
  });

  // ── Edit form ──────────────────────────────────────────────────────────────

  it("shows Edit and Delete buttons only for the owner", async () => {
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    expect(screen.getByRole("button", { name: /^Edit$/i })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /^Delete$/i }),
    ).toBeInTheDocument();
  });

  it("hides Edit and Delete buttons for non-owner", async () => {
    setCurrentUser("u-other");

    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    expect(
      screen.queryByRole("button", { name: /^Edit$/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /^Delete$/i }),
    ).not.toBeInTheDocument();
  });

  it("opens edit form pre-filled with current values", async () => {
    const user = userEvent.setup();
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /^Edit$/i }));

    expect(
      screen.getByRole("heading", { name: /Edit Group/i }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/Name/i)).toHaveValue("Book Club");
    expect(screen.getByLabelText(/Description/i)).toHaveValue("A great club");
  });

  it("saves edits via the group store", async () => {
    const user = userEvent.setup();
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /^Edit$/i }));

    const nameInput = screen.getByLabelText(/Name/i);
    await user.clear(nameInput);
    await user.type(nameInput, "New Name");
    await user.click(screen.getByRole("button", { name: /^Save$/i }));

    await waitFor(() => {
      expect(groupStore.update).toHaveBeenCalledWith("g-1", {
        name: "New Name",
        description: "A great club",
      });
    });
    expect(
      screen.queryByRole("heading", { name: /Edit Group/i }),
    ).not.toBeInTheDocument();
  });

  it("surfaces save errors in the edit form", async () => {
    const user = userEvent.setup();
    vi.mocked(groupStore).update = vi
      .fn()
      .mockRejectedValueOnce(new Error("Name taken"));

    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /^Edit$/i }));
    await user.click(screen.getByRole("button", { name: /^Save$/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Name taken");
    expect(
      screen.getByRole("heading", { name: /Edit Group/i }),
    ).toBeInTheDocument();
  });

  it("cancels editing without saving", async () => {
    const user = userEvent.setup();
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /^Edit$/i }));
    await user.click(screen.getByRole("button", { name: /Cancel/i }));

    expect(
      screen.queryByRole("heading", { name: /Edit Group/i }),
    ).not.toBeInTheDocument();
    expect(groupStore.update).not.toHaveBeenCalled();
  });

  // ── Delete flow ────────────────────────────────────────────────────────────

  it("shows delete confirmation prompt on Delete click", async () => {
    const user = userEvent.setup();
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /^Delete$/i }));

    expect(screen.getByText(/Delete this group\?/)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Yes, delete/i }),
    ).toBeInTheDocument();
  });

  it("deletes the group and navigates away on confirm", async () => {
    const user = userEvent.setup();
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /^Delete$/i }));
    await user.click(screen.getByRole("button", { name: /Yes, delete/i }));

    await waitFor(() => {
      expect(groupStore.remove).toHaveBeenCalledWith("g-1");
      expect(routerStore.navigate).toHaveBeenCalledWith("groups");
    });
  });

  it("cancels delete without removing the group", async () => {
    const user = userEvent.setup();
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /^Delete$/i }));
    await user.click(screen.getByRole("button", { name: /^Cancel$/i }));

    expect(screen.queryByText(/Delete this group\?/)).not.toBeInTheDocument();
    expect(groupStore.remove).not.toHaveBeenCalled();
  });

  // ── Add member ─────────────────────────────────────────────────────────────

  it("shows Add Member button only for the owner", async () => {
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    expect(
      screen.getByRole("button", { name: /Add Member/i }),
    ).toBeInTheDocument();
  });

  it("hides Add Member button for non-owner", async () => {
    setCurrentUser("u-other");
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    expect(
      screen.queryByRole("button", { name: /Add Member/i }),
    ).not.toBeInTheDocument();
  });

  it("shows add-member form when Add Member is clicked", async () => {
    const user = userEvent.setup();
    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /Add Member/i }));

    expect(screen.getByLabelText(/User ID to add/i)).toBeInTheDocument();
  });

  it("adds a member and syncs member count from refreshed list", async () => {
    const user = userEvent.setup();
    const newMember: ReadingGroupMember = {
      ...fakeMember,
      user_id: "u-new",
      user_name: "bob",
    };
    vi.mocked(listGroupMembers)
      .mockResolvedValueOnce([fakeMember])
      .mockResolvedValueOnce([fakeMember, newMember]);

    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /Add Member/i }));
    await user.type(screen.getByLabelText(/User ID to add/i), "u-new");
    await user.click(screen.getByRole("button", { name: /^Add$/i }));

    await waitFor(() => {
      expect(addGroupMember).toHaveBeenCalledWith("g-1", "u-new");
      expect(groupStore.setMemberCount).toHaveBeenCalledWith("g-1", 2);
    });
  });

  it("shows error and keeps form open when adding a member fails", async () => {
    const user = userEvent.setup();
    vi.mocked(addGroupMember).mockRejectedValueOnce(
      new Error("User not found"),
    );

    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await user.click(screen.getByRole("button", { name: /Add Member/i }));
    await user.type(screen.getByLabelText(/User ID to add/i), "u-unknown");
    await user.click(screen.getByRole("button", { name: /^Add$/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "User not found",
    );
    expect(screen.getByLabelText(/User ID to add/i)).toBeInTheDocument();
  });

  // ── Remove member ──────────────────────────────────────────────────────────

  it("shows remove button for non-self members (owner view)", async () => {
    render(GroupDetail, { props: { groupId: "g-1" } });

    await waitFor(() => screen.getByText("alice"));

    expect(
      screen.getByLabelText(/Remove alice from group/i),
    ).toBeInTheDocument();
  });

  it("shows remove confirmation after clicking the remove button", async () => {
    const user = userEvent.setup();
    render(GroupDetail, { props: { groupId: "g-1" } });
    await waitFor(() => screen.getByText("alice"));

    await user.click(screen.getByLabelText(/Remove alice from group/i));

    expect(screen.getByText("Remove?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Yes$/i })).toBeInTheDocument();
  });

  it("removes a member and syncs member count from refreshed list", async () => {
    const user = userEvent.setup();
    vi.mocked(listGroupMembers)
      .mockResolvedValueOnce([fakeMember])
      .mockResolvedValueOnce([]);

    render(GroupDetail, { props: { groupId: "g-1" } });
    await waitFor(() => screen.getByText("alice"));

    await user.click(screen.getByLabelText(/Remove alice from group/i));
    await user.click(screen.getByRole("button", { name: /^Yes$/i }));

    await waitFor(() => {
      expect(removeGroupMember).toHaveBeenCalledWith("g-1", "u-2");
      expect(groupStore.setMemberCount).toHaveBeenCalledWith("g-1", 0);
    });
  });

  // ── Share list dropdown ────────────────────────────────────────────────────

  it("shows the share dropdown when the user has unshared lists", async () => {
    (vi.mocked(readingListStore) as unknown as { lists: ReadingList[] }).lists =
      [fakeList];
    vi.mocked(listGroupReadingLists).mockResolvedValue([]);

    render(GroupDetail, { props: { groupId: "g-1" } });

    expect(
      await screen.findByLabelText(/Select a reading list to share/i),
    ).toBeInTheDocument();
  });

  it("hides the share dropdown when all owned lists are already shared", async () => {
    (vi.mocked(readingListStore) as unknown as { lists: ReadingList[] }).lists =
      [fakeList];
    vi.mocked(listGroupReadingLists).mockResolvedValue([fakeList]);

    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await waitFor(() => {
      expect(
        screen.queryByLabelText(/Select a reading list to share/i),
      ).not.toBeInTheDocument();
    });
  });

  // ── Unshare button guard ───────────────────────────────────────────────────

  it("shows unshare button to the group owner for any shared list", async () => {
    vi.mocked(listGroupReadingLists).mockResolvedValue([fakeList]);

    render(GroupDetail, { props: { groupId: "g-1" } });

    await waitFor(() => {
      expect(
        screen.getByLabelText(/Unshare To Read from group/i),
      ).toBeInTheDocument();
    });
  });

  it("shows unshare button to a non-owner for lists they own", async () => {
    setCurrentUser("u-other");
    (vi.mocked(readingListStore) as unknown as { lists: ReadingList[] }).lists =
      [fakeList];
    vi.mocked(listGroupReadingLists).mockResolvedValue([fakeList]);

    render(GroupDetail, { props: { groupId: "g-1" } });

    await waitFor(() => {
      expect(
        screen.getByLabelText(/Unshare To Read from group/i),
      ).toBeInTheDocument();
    });
  });

  it("hides unshare button from non-owner for lists they do not own", async () => {
    setCurrentUser("u-other");
    (vi.mocked(readingListStore) as unknown as { lists: ReadingList[] }).lists =
      [];
    vi.mocked(listGroupReadingLists).mockResolvedValue([fakeList]);

    render(GroupDetail, { props: { groupId: "g-1" } });
    await tick();

    await waitFor(() => {
      expect(
        screen.queryByLabelText(/Unshare To Read from group/i),
      ).not.toBeInTheDocument();
    });
  });
});
