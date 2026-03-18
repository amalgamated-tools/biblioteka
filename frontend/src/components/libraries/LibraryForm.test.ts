import { describe, expect, it, vi, afterEach } from "vitest";
import {
  render,
  fireEvent,
  cleanup,
  screen,
  within,
} from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../stores/libraries.svelte", () => ({
  libraryStore: {
    libraries: [],
    add: vi.fn(),
    edit: vi.fn(),
    remove: vi.fn(),
  },
}));

vi.mock("../../stores/router.svelte", () => ({
  routerStore: {
    navigate: vi.fn(),
  },
}));

vi.mock("lucide-svelte", () => ({
  Plus: () => {},
  FolderOpen: () => {},
  Trash2: () => {},
  X: () => {},
}));

import LibraryForm from "./LibraryForm.svelte";
import { libraryStore } from "../../stores/libraries.svelte";

describe("LibraryForm accessibility", () => {
  afterEach(() => {
    cleanup();
    libraryStore.libraries = [];
  });

  it("marks the name input as aria-required", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const nameInput = container.querySelector("#lib-name");
    expect(nameInput).toHaveAttribute("aria-required", "true");
  });

  it("marks folder path inputs as aria-required", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const folderInput = container.querySelector(
      'input[aria-label="Folder path"]',
    );
    expect(folderInput).toHaveAttribute("aria-required", "true");
  });

  it("shows required indicator (*) on Name and Folders labels", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const nameLabel = container.querySelector('label[for="lib-name"]');
    expect(
      nameLabel?.querySelector('span[aria-hidden="true"]'),
    ).toHaveTextContent("*");

    const foldersLabel = container.querySelector("fieldset legend");
    expect(
      foldersLabel?.querySelector('span[aria-hidden="true"]'),
    ).toHaveTextContent("*");
  });

  it("shows inline name error with aria-invalid when submitting empty name", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const form = container.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    const nameInput = container.querySelector("#lib-name");
    expect(nameInput).toHaveAttribute("aria-invalid", "true");
    expect(nameInput).toHaveAttribute("aria-describedby", "lib-name-error");

    const errorMessage = container.querySelector("#lib-name-error");
    expect(errorMessage).toBeInTheDocument();
    expect(errorMessage).toHaveTextContent("Name is required");
    expect(errorMessage).toHaveAttribute("role", "alert");
  });

  it("shows inline folder error with aria-invalid when submitting empty paths", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    // Fill in the name so we get past the name validation
    const nameInput = container.querySelector(
      "#lib-name",
    ) as HTMLInputElement;
    await fireEvent.input(nameInput, { target: { value: "My Library" } });

    const form = container.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    const folderInput = container.querySelector(
      'input[aria-label="Folder path"]',
    );
    expect(folderInput).toHaveAttribute("aria-invalid", "true");
    expect(folderInput).toHaveAttribute(
      "aria-describedby",
      "lib-folders-error",
    );

    const errorMessage = container.querySelector("#lib-folders-error");
    expect(errorMessage).toBeInTheDocument();
    expect(errorMessage).toHaveTextContent("At least one folder is required");
    expect(errorMessage).toHaveAttribute("role", "alert");
  });

  it("does not show aria-invalid or error messages before submission", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const nameInput = container.querySelector("#lib-name");
    expect(nameInput).not.toHaveAttribute("aria-invalid");
    expect(nameInput).not.toHaveAttribute("aria-describedby");

    expect(container.querySelector("#lib-name-error")).toBeNull();
    expect(container.querySelector("#lib-folders-error")).toBeNull();
  });

  it("moves focus into delete confirmation and uses descriptive controls", async () => {
    libraryStore.libraries = [
      {
        id: "lib-1",
        name: "Library",
        paths: ["/books"],
        organization_type: "book_per_folder",
        monitored: false,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ];

    render(LibraryForm, {
      props: { mode: "edit", editId: "lib-1" },
    });
    await tick();

    await fireEvent.click(screen.getByRole("button", { name: "Delete Library" }));
    await tick();

    const deleteConfirm = screen.getByRole("alertdialog", {
      name: "Confirm library deletion",
    });
    expect(deleteConfirm).toHaveAttribute("aria-describedby", "delete-confirm-msg");
    expect(screen.getByText("Delete this library?")).toHaveAttribute(
      "id",
      "delete-confirm-msg",
    );

    const confirmButton = within(deleteConfirm).getByRole("button", {
      name: "Yes, delete library",
    });
    expect(confirmButton).toHaveFocus();

    expect(
      within(deleteConfirm).getByRole("button", { name: "Cancel" }),
    ).toBeInTheDocument();
  });

  it("traps focus inside the delete confirmation controls", async () => {
    libraryStore.libraries = [
      {
        id: "lib-1",
        name: "Library",
        paths: ["/books"],
        organization_type: "book_per_folder",
        monitored: false,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ];

    render(LibraryForm, {
      props: { mode: "edit", editId: "lib-1" },
    });
    await tick();

    await fireEvent.click(screen.getByRole("button", { name: "Delete Library" }));
    await tick();

    const deleteConfirm = screen.getByRole("alertdialog", {
      name: "Confirm library deletion",
    });
    const confirmButton = within(deleteConfirm).getByRole("button", {
      name: "Yes, delete library",
    });
    const cancelButton = within(deleteConfirm).getByRole("button", {
      name: "Cancel",
    });

    expect(confirmButton).toHaveFocus();

    await fireEvent.keyDown(deleteConfirm, { key: "Tab" });
    expect(cancelButton).toHaveFocus();

    await fireEvent.keyDown(deleteConfirm, { key: "Tab", shiftKey: true });
    expect(confirmButton).toHaveFocus();
  });
});
