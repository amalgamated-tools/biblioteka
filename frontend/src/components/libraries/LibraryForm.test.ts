import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { render, fireEvent, cleanup } from "@testing-library/svelte";
import { tick } from "svelte";
import {
  LIBRARY_ORGANIZATION_OPTIONS,
  LIBRARY_ORGANIZATION_TYPES,
} from "../../types";

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
  afterEach(() => cleanup());

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
    const nameInput = container.querySelector("#lib-name") as HTMLInputElement;
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
});

describe("LibraryForm monitor toggle switch", () => {
  afterEach(() => cleanup());

  it("associates the switch input with its label via for/id", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const label = container.querySelector('label[for="lib-monitored"]');
    expect(label).toBeInTheDocument();

    const input = container.querySelector("#lib-monitored");
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute("role", "switch");
  });

  it("has aria-checked reflecting the unchecked state by default", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const input = container.querySelector("#lib-monitored") as HTMLInputElement;
    expect(input.checked).toBe(false);
    expect(input).toHaveAttribute("aria-checked", "false");
  });

  it("updates aria-checked when toggled", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const input = container.querySelector("#lib-monitored") as HTMLInputElement;
    expect(input).toHaveAttribute("aria-checked", "false");

    await fireEvent.click(input);
    await tick();

    expect(input.checked).toBe(true);
    expect(input).toHaveAttribute("aria-checked", "true");
  });
});

describe("LibraryForm organization type dropdown", () => {
  afterEach(() => cleanup());

  it("renders the organization type dropdown", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const select = container.querySelector(
      "#lib-org-type",
    ) as HTMLSelectElement;
    expect(select).toBeInTheDocument();
    expect(select.tagName).toBe("SELECT");
  });

  it("has three options with correct values", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const select = container.querySelector(
      "#lib-org-type",
    ) as HTMLSelectElement;
    const options = select.querySelectorAll("option");
    expect(options).toHaveLength(LIBRARY_ORGANIZATION_OPTIONS.length);

    const values = Array.from(options).map((o) => o.value);
    expect(values).toEqual(
      LIBRARY_ORGANIZATION_OPTIONS.map((option) => option.value),
    );
  });

  it("defaults to book_per_folder in create mode", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const select = container.querySelector(
      "#lib-org-type",
    ) as HTMLSelectElement;
    expect(select.value).toBe(LIBRARY_ORGANIZATION_TYPES.BOOK_PER_FOLDER);
  });

  it("has a label with text File Organization", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "create", editId: "" },
    });
    await tick();

    const label = container.querySelector('label[for="lib-org-type"]');
    expect(label).toBeInTheDocument();
    expect(label).toHaveTextContent("File Organization");
  });
});

describe("LibraryForm delete confirmation accessibility", () => {
  const mockLibrary = {
    id: "lib-1",
    name: "My Fiction",
    paths: ["/books"],
    monitored: false,
    organization_type: LIBRARY_ORGANIZATION_TYPES.BOOK_PER_FOLDER,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };

  beforeEach(() => {
    Object.assign(libraryStore, { libraries: [mockLibrary] });
  });

  afterEach(() => {
    Object.assign(libraryStore, { libraries: [] });
    cleanup();
  });

  it("shows the delete trigger button in edit mode", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "edit", editId: "lib-1" },
    });
    await tick();

    const trigger = container.querySelector('[data-delete-trigger="lib-delete"]');
    expect(trigger).toBeInTheDocument();
    expect(trigger).toHaveTextContent("Delete Library");
  });

  it("shows delete confirmation with role=alertdialog when trigger is clicked", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "edit", editId: "lib-1" },
    });
    await tick();

    const trigger = container.querySelector<HTMLButtonElement>(
      '[data-delete-trigger="lib-delete"]',
    )!;
    await fireEvent.click(trigger);
    await tick();

    const dialog = container.querySelector('[role="alertdialog"]');
    expect(dialog).toBeInTheDocument();
  });

  it("delete confirmation has aria-labelledby pointing to label element", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "edit", editId: "lib-1" },
    });
    await tick();

    const trigger = container.querySelector<HTMLButtonElement>(
      '[data-delete-trigger="lib-delete"]',
    )!;
    await fireEvent.click(trigger);
    await tick();

    const dialog = container.querySelector('[role="alertdialog"]')!;
    const labelledBy = dialog.getAttribute("aria-labelledby");
    expect(labelledBy).toBeTruthy();

    const label = container.querySelector(`#${labelledBy}`);
    expect(label).toBeInTheDocument();
    expect(label).toHaveTextContent('Delete "My Fiction"?');
  });

  it("pressing Escape on the confirmation dismisses it and restores focus to the trigger", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "edit", editId: "lib-1" },
    });
    await tick();

    const trigger = container.querySelector<HTMLButtonElement>(
      '[data-delete-trigger="lib-delete"]',
    )!;
    await fireEvent.click(trigger);
    await tick();

    const dialog = container.querySelector('[role="alertdialog"]')!;
    await fireEvent.keyDown(dialog, { key: "Escape" });
    await tick();

    expect(container.querySelector('[role="alertdialog"]')).toBeNull();
    const restoredTrigger = container.querySelector<HTMLButtonElement>(
      '[data-delete-trigger="lib-delete"]',
    )!;
    expect(restoredTrigger).toBeInTheDocument();
    expect(document.activeElement).toBe(restoredTrigger);
  });

  it("clicking the cancel button dismisses the confirmation and restores focus to the trigger", async () => {
    const { container } = render(LibraryForm, {
      props: { mode: "edit", editId: "lib-1" },
    });
    await tick();

    const trigger = container.querySelector<HTMLButtonElement>(
      '[data-delete-trigger="lib-delete"]',
    )!;
    await fireEvent.click(trigger);
    await tick();

    const buttons = container.querySelectorAll('[role="alertdialog"] button');
    const cancelButton = Array.from(buttons).find(
      (b) => b.textContent?.trim() === "Cancel",
    )!;
    await fireEvent.click(cancelButton);
    await tick();

    expect(container.querySelector('[role="alertdialog"]')).toBeNull();
    const restoredTrigger = container.querySelector<HTMLButtonElement>(
      '[data-delete-trigger="lib-delete"]',
    )!;
    expect(restoredTrigger).toBeInTheDocument();
    expect(document.activeElement).toBe(restoredTrigger);
  });
});
