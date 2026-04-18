import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../stores/groups.svelte", () => ({
  groupStore: {
    loaded: false,
    loading: false,
    loadError: null as string | null,
    groups: [],
    load: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  },
}));

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    subPath: "",
    navigate: vi.fn(),
  },
}));

vi.mock("./ui/AlertBanner.svelte", () => ({ default: () => {} }));
vi.mock("./groups/GroupDetail.svelte", () => ({ default: () => {} }));

vi.mock("lucide-svelte", () => ({
  Users: () => {},
  Plus: () => {},
  X: () => {},
  Check: () => {},
}));

import Groups from "./Groups.svelte";
import { groupStore } from "../stores/groups.svelte";
import { routerStore } from "../stores/router.svelte";
import type { ReadingGroup } from "../types";

const fakeGroup: ReadingGroup = {
  id: "g-1",
  owner_id: "u-1",
  name: "Book Club",
  description: null,
  member_count: 2,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("Groups", () => {
  beforeEach(() => {
    vi.mocked(groupStore).loaded = false;
    vi.mocked(groupStore).loadError = null;
    vi.mocked(groupStore).groups = [];
    vi.mocked(routerStore).subPath = "";
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("calls load on mount when not loaded", async () => {
    vi.mocked(groupStore).loaded = false;
    render(Groups);
    await tick();
    expect(vi.mocked(groupStore).load).toHaveBeenCalled();
  });

  it("shows empty state when no groups and loaded", async () => {
    vi.mocked(groupStore).loaded = true;
    vi.mocked(groupStore).loadError = null;
    vi.mocked(groupStore).groups = [];
    render(Groups);
    await tick();

    expect(screen.getByText(/No reading groups yet/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Create Your First Group/i }),
    ).toBeInTheDocument();
  });

  it("does not show empty state when load failed", async () => {
    vi.mocked(groupStore).loaded = true;
    vi.mocked(groupStore).loadError = "network error";
    vi.mocked(groupStore).groups = [];
    render(Groups);
    await tick();

    expect(
      screen.queryByText(/No reading groups yet/i),
    ).not.toBeInTheDocument();
  });

  it("shows group cards when groups exist", async () => {
    vi.mocked(groupStore).loaded = true;
    vi.mocked(groupStore).groups = [fakeGroup];
    render(Groups);
    await tick();

    expect(screen.getByText("Book Club")).toBeInTheDocument();
    expect(screen.getByText("2 members")).toBeInTheDocument();
  });

  it("shows 'New Group' button in header", async () => {
    vi.mocked(groupStore).loaded = true;
    render(Groups);
    await tick();

    expect(
      screen.getByRole("button", { name: /New Group/i }),
    ).toBeInTheDocument();
  });

  it("shows create form after clicking New Group", async () => {
    vi.mocked(groupStore).loaded = true;
    render(Groups);
    await tick();

    fireEvent.click(screen.getByRole("button", { name: /New Group/i }));
    await tick();

    expect(
      screen.getByRole("heading", { name: /Create Reading Group/i }),
    ).toBeInTheDocument();
  });

  it("hides New Group button when form is open", async () => {
    vi.mocked(groupStore).loaded = true;
    render(Groups);
    await tick();

    fireEvent.click(screen.getByRole("button", { name: /New Group/i }));
    await tick();

    expect(
      screen.queryByRole("button", { name: /New Group/i }),
    ).not.toBeInTheDocument();
  });

  it("hides the form after Cancel", async () => {
    vi.mocked(groupStore).loaded = true;
    render(Groups);
    await tick();

    fireEvent.click(screen.getByRole("button", { name: /New Group/i }));
    await tick();
    fireEvent.click(screen.getByRole("button", { name: /Cancel/i }));
    await tick();

    expect(
      screen.queryByRole("heading", { name: /Create Reading Group/i }),
    ).not.toBeInTheDocument();
  });

  it("announces required fields in the create form", async () => {
    vi.mocked(groupStore).loaded = true;
    render(Groups);
    await tick();

    await fireEvent.click(screen.getByRole("button", { name: /New Group/i }));
    await tick();

    const nameInput = screen.getByLabelText(/Name/i);
    expect(nameInput).toHaveAttribute("aria-required", "true");

    const legend = screen.getByText(/Fields marked with/i, { exact: false });
    expect(legend).toBeInTheDocument();
    expect(legend.textContent).toMatch(/are required/i);
  });

  it("renders GroupDetail when subPath is a group ID", async () => {
    vi.mocked(groupStore).loaded = true;
    vi.mocked(routerStore).subPath = "g-1";
    render(Groups);
    await tick();

    // When subPath is set, the component renders GroupDetail (mocked).
    // Confirm the groups list is NOT rendered.
    expect(
      screen.queryByText(/No reading groups yet/i),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: /Reading Groups/i }),
    ).not.toBeInTheDocument();
  });

  it("marks name field invalid after create error", async () => {
    vi.mocked(groupStore).loaded = true;
    vi.mocked(groupStore).create = vi
      .fn()
      .mockRejectedValueOnce(new Error("Name already exists"));
    render(Groups);
    await tick();

    fireEvent.click(screen.getByRole("button", { name: /New Group/i }));
    await tick();

    await fireEvent.input(screen.getByLabelText(/Name/i), {
      target: { value: "Book Club" },
    });
    await fireEvent.click(screen.getByRole("button", { name: /^Create$/i }));
    await tick();

    expect(screen.getByLabelText(/Name/i)).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(screen.getByLabelText(/Name/i)).toHaveAttribute(
      "aria-describedby",
      "create-group-error",
    );
  });
});
