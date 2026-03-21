import { describe, expect, it, vi, afterEach } from "vitest";
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
