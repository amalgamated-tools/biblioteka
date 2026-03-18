import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../stores/auth.svelte", () => ({
  authStore: {
    user: { email: "reader@example.com" },
    signOut: vi.fn(),
  },
}));

vi.mock("../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

vi.mock("../stores/libraries.svelte", () => ({
  libraryStore: {
    loaded: true,
    libraries: [],
    load: vi.fn(),
  },
}));

vi.mock("../lib/api", () => ({
  getVersion: vi.fn().mockResolvedValue("0.0.0-test"),
}));

vi.mock("lucide-svelte", () => ({
  LayoutDashboard: () => {},
  BookOpen: () => {},
  Library: () => {},
  Plus: () => {},
  Settings: () => {},
  LogOut: () => {},
  BookCheck: () => {},
  Settings2: () => {},
}));

import Sidebar from "./Sidebar.svelte";

describe("Sidebar heading semantics", () => {
  afterEach(() => cleanup());

  it("does not render the brand as a top-level heading", async () => {
    const { container } = render(Sidebar, {
      props: {
        currentView: "dashboard",
        onNavigate: vi.fn(),
        open: true,
        onClose: vi.fn(),
      },
    });

    await tick();

    expect(container.querySelectorAll("h1")).toHaveLength(0);
    expect(container).toHaveTextContent("biblioteka");
  });
});
