import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
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
  ]),
  previewCalibreImport: vi.fn().mockResolvedValue({
    total: 3,
    books: [
      {
        calibre_id: 1,
        title: "Dune",
        authors: ["Frank Herbert"],
        series: [{ name: "Dune", position: 1 }],
        publisher: "Chilton Books",
        publication_date: "1965-08-01",
        isbn13: "9780441013593",
        isbn10: "",
        formats: ["epub"],
      },
      {
        calibre_id: 2,
        title: "Foundation",
        authors: ["Isaac Asimov"],
        series: [],
        publisher: "",
        publication_date: "",
        isbn13: "",
        isbn10: "",
        formats: ["epub", "mobi"],
      },
    ],
  }),
  confirmCalibreImport: vi.fn().mockResolvedValue({
    total: 3,
    imported: 3,
    skipped: 0,
    errors: 0,
  }),
  previewCalibreImportFromPath: vi.fn().mockResolvedValue({
    total: 3,
    books: [
      {
        calibre_id: 1,
        title: "Dune",
        authors: ["Frank Herbert"],
        series: [{ name: "Dune", position: 1 }],
        publisher: "Chilton Books",
        formats: ["epub"],
      },
      {
        calibre_id: 2,
        title: "Foundation",
        authors: ["Isaac Asimov"],
        series: [],
        formats: ["epub", "mobi"],
      },
    ],
  }),
  confirmCalibreImportFromPath: vi.fn().mockResolvedValue({
    total: 3,
    imported: 3,
    skipped: 0,
    errors: 0,
  }),
}));

vi.mock("lucide-svelte", () => ({
  DatabaseZap: () => {},
}));

import CalibreImportTab from "./CalibreImportTab.svelte";
import {
  previewCalibreImport,
  confirmCalibreImport,
  previewCalibreImportFromPath,
  confirmCalibreImportFromPath,
} from "../../lib/api";

describe("CalibreImportTab rendering", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the heading", async () => {
    render(CalibreImportTab);
    await tick();

    expect(
      screen.getByRole("heading", { name: /Import from Calibre/i }),
    ).toBeInTheDocument();
  });

  it("renders the file input and preview button", async () => {
    render(CalibreImportTab);
    await tick();

    expect(screen.getByLabelText(/Calibre metadata\.db/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Preview Import/i }),
    ).toBeInTheDocument();
  });

  it("shows step indicators", async () => {
    render(CalibreImportTab);
    await tick();

    expect(screen.getByText("Upload")).toBeInTheDocument();
    expect(screen.getByText("Preview")).toBeInTheDocument();
    expect(screen.getByText("Done")).toBeInTheDocument();
  });

  it("shows error when submitting without a file", async () => {
    render(CalibreImportTab);
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      /Please select a Calibre metadata\.db file/i,
    );
  });

  it("shows error when previewCalibreImport rejects", async () => {
    vi.mocked(previewCalibreImport).mockRejectedValueOnce(
      new Error("invalid database file"),
    );

    render(CalibreImportTab);
    await tick();

    // Simulate a file selection by creating a fake File.
    const fileInput = screen.getByLabelText(
      /Calibre metadata\.db/i,
    ) as HTMLInputElement;
    const fakeFile = new File(["data"], "metadata.db", {
      type: "application/octet-stream",
    });
    Object.defineProperty(fileInput, "files", { value: [fakeFile] });
    await fireEvent.change(fileInput);

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      /invalid database file/i,
    );
  });

  it("advances to preview step after successful preview", async () => {
    render(CalibreImportTab);
    await tick();

    const fileInput = screen.getByLabelText(
      /Calibre metadata\.db/i,
    ) as HTMLInputElement;
    const fakeFile = new File(["data"], "metadata.db", {
      type: "application/octet-stream",
    });
    Object.defineProperty(fileInput, "files", { value: [fakeFile] });
    await fireEvent.change(fileInput);

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    // Preview step should show the total count and a book title.
    expect(
      screen.getByText(/books in your Calibre library/),
    ).toBeInTheDocument();
    expect(screen.getByText("Dune")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Import 3 books/i }),
    ).toBeInTheDocument();
  });

  it("shows result step after confirming import", async () => {
    render(CalibreImportTab);
    await tick();

    // Advance to preview step.
    const fileInput = screen.getByLabelText(
      /Calibre metadata\.db/i,
    ) as HTMLInputElement;
    const fakeFile = new File(["data"], "metadata.db", {
      type: "application/octet-stream",
    });
    Object.defineProperty(fileInput, "files", { value: [fakeFile] });
    await fireEvent.change(fileInput);

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    // Confirm the import.
    const confirmBtn = screen.getByRole("button", { name: /Import 3 books/i });
    await fireEvent.click(confirmBtn);
    await tick();
    await tick();

    expect(confirmCalibreImport).toHaveBeenCalledOnce();
    expect(screen.getByText(/Successfully imported/i)).toBeInTheDocument();
  });

  it("Start Over button returns to upload step", async () => {
    render(CalibreImportTab);
    await tick();

    // Advance to preview step.
    const fileInput = screen.getByLabelText(
      /Calibre metadata\.db/i,
    ) as HTMLInputElement;
    const fakeFile = new File(["data"], "metadata.db", {
      type: "application/octet-stream",
    });
    Object.defineProperty(fileInput, "files", { value: [fakeFile] });
    await fireEvent.change(fileInput);

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    const startOver = screen.getByRole("button", { name: /Start Over/i });
    await fireEvent.click(startOver);
    await tick();

    // Back to upload — file input visible again.
    expect(screen.getByLabelText(/Calibre metadata\.db/i)).toBeInTheDocument();
  });
});

describe("CalibreImportTab server-path mode", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders source toggle with Upload File and Server Path buttons", async () => {
    render(CalibreImportTab);
    await tick();

    expect(
      screen.getByRole("button", { name: /Upload File/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Server Path/i }),
    ).toBeInTheDocument();
  });

  it("shows text input when Server Path is selected", async () => {
    render(CalibreImportTab);
    await tick();

    const serverPathBtn = screen.getByRole("button", {
      name: /Server Path/i,
    });
    await fireEvent.click(serverPathBtn);
    await tick();

    expect(
      screen.getByLabelText(/Server path to metadata\.db/i),
    ).toBeInTheDocument();
    // File input should not be visible.
    expect(
      screen.queryByLabelText(/^Calibre metadata\.db$/i),
    ).not.toBeInTheDocument();
  });

  it("shows error when submitting server path mode without a path", async () => {
    render(CalibreImportTab);
    await tick();

    const serverPathBtn = screen.getByRole("button", {
      name: /Server Path/i,
    });
    await fireEvent.click(serverPathBtn);
    await tick();

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      /Please enter a server path/i,
    );
  });

  it("advances to preview step with server path", async () => {
    render(CalibreImportTab);
    await tick();

    const serverPathBtn = screen.getByRole("button", {
      name: /Server Path/i,
    });
    await fireEvent.click(serverPathBtn);
    await tick();

    const pathInput = screen.getByLabelText(/Server path to metadata\.db/i);
    await fireEvent.input(pathInput, {
      target: { value: "/home/user/Calibre Library/metadata.db" },
    });

    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(previewCalibreImportFromPath).toHaveBeenCalledWith(
      "/home/user/Calibre Library/metadata.db",
    );
    expect(
      screen.getByText(/books in your Calibre library/),
    ).toBeInTheDocument();
  });

  it("calls confirmCalibreImportFromPath in server path mode", async () => {
    render(CalibreImportTab);
    await tick();

    // Switch to path mode.
    const serverPathBtn = screen.getByRole("button", {
      name: /Server Path/i,
    });
    await fireEvent.click(serverPathBtn);
    await tick();

    // Enter a path.
    const pathInput = screen.getByLabelText(/Server path to metadata\.db/i);
    await fireEvent.input(pathInput, {
      target: { value: "/home/user/Calibre Library/metadata.db" },
    });

    // Submit preview.
    const form = document.querySelector("form")!;
    await fireEvent.submit(form);
    await tick();
    await tick();

    // Confirm import.
    const confirmBtn = screen.getByRole("button", { name: /Import 3 books/i });
    await fireEvent.click(confirmBtn);
    await tick();
    await tick();

    expect(confirmCalibreImportFromPath).toHaveBeenCalledOnce();
    expect(screen.getByText(/Successfully imported/i)).toBeInTheDocument();
  });
});
