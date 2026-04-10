import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  getWatchFolderConfig: vi.fn().mockResolvedValue({
    path: "",
    library_id: "",
  }),
  setWatchFolderConfig: vi.fn().mockResolvedValue({
    message: "Watch folder configuration saved successfully",
  }),
  listLibraries: vi.fn().mockResolvedValue([
    {
      id: "lib-1",
      name: "My Library",
      paths: ["/books"],
      organization_type: "none",
      monitored: true,
      created_at: "",
      updated_at: "",
    },
    {
      id: "lib-2",
      name: "Archive",
      paths: ["/archive"],
      organization_type: "none",
      monitored: false,
      created_at: "",
      updated_at: "",
    },
  ]),
}));

vi.mock("lucide-svelte", () => ({
  FolderSearch: () => {},
}));

import WatchFolderTab from "./WatchFolderTab.svelte";
import {
  getWatchFolderConfig,
  setWatchFolderConfig,
  listLibraries,
} from "../../lib/api";

describe("WatchFolderTab rendering", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the Watch Folder heading", async () => {
    render(WatchFolderTab);
    await tick();

    expect(
      screen.getByRole("heading", { name: /Watch Folder/i }),
    ).toBeInTheDocument();
  });

  it("loads watch folder config on mount", async () => {
    render(WatchFolderTab);
    await tick();
    await tick();

    expect(getWatchFolderConfig).toHaveBeenCalled();
    expect(listLibraries).toHaveBeenCalled();
  });

  it("shows 'Not configured' status when path is empty", async () => {
    render(WatchFolderTab);
    await tick();

    expect(screen.getByText("Not configured")).toBeInTheDocument();
  });

  it("shows 'Configured' status when path is set", async () => {
    vi.mocked(getWatchFolderConfig).mockResolvedValueOnce({
      path: "/incoming",
      library_id: "lib-1",
    });
    render(WatchFolderTab);
    await tick();
    await tick();

    expect(screen.getByText("Configured")).toBeInTheDocument();
  });

  it("shows 'Save Configuration' button when not configured", async () => {
    render(WatchFolderTab);
    await tick();
    await tick();

    expect(
      screen.getByRole("button", { name: "Save Configuration" }),
    ).toBeInTheDocument();
  });

  it("populates library dropdown options", async () => {
    render(WatchFolderTab);
    await tick();
    await tick();

    const select = screen.getByLabelText("Target Library") as HTMLSelectElement;
    const options = Array.from(select.options);
    expect(options).toHaveLength(3); // placeholder + 2 libraries
    expect(options[1].textContent).toBe("My Library");
    expect(options[2].textContent).toBe("Archive");
  });
});

describe("WatchFolderTab form validation", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows error when path is set but library is not selected", async () => {
    render(WatchFolderTab);
    await tick();
    await tick();

    // Type a path
    const pathInput = screen.getByLabelText("Folder Path") as HTMLInputElement;
    await fireEvent.input(pathInput, { target: { value: "/incoming" } });
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Target library is required",
    );
  });

  it("calls setWatchFolderConfig with correct values on valid submission", async () => {
    vi.mocked(getWatchFolderConfig).mockResolvedValueOnce({
      path: "/incoming",
      library_id: "lib-1",
    });
    render(WatchFolderTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(setWatchFolderConfig).toHaveBeenCalledWith({
      path: "/incoming",
      library_id: "lib-1",
    });
  });

  it("shows success message after saving", async () => {
    vi.mocked(getWatchFolderConfig).mockResolvedValueOnce({
      path: "/incoming",
      library_id: "lib-1",
    });
    render(WatchFolderTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(
      screen.getByText("Watch folder configuration saved successfully"),
    ).toBeInTheDocument();
  });

  it("shows error banner when setWatchFolderConfig rejects", async () => {
    vi.mocked(getWatchFolderConfig).mockResolvedValueOnce({
      path: "/incoming",
      library_id: "lib-1",
    });
    vi.mocked(setWatchFolderConfig).mockRejectedValueOnce(
      new Error("folder not found"),
    );
    render(WatchFolderTab);
    await tick();
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("folder not found");
  });

  it("can clear the configuration by submitting empty path", async () => {
    render(WatchFolderTab);
    await tick();
    await tick();

    // Path is already empty, just submit
    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(setWatchFolderConfig).toHaveBeenCalledWith({
      path: "",
      library_id: "",
    });
  });
});
