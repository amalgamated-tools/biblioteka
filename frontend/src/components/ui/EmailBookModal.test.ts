import {
  describe,
  expect,
  it,
  vi,
  afterEach,
  beforeEach,
  type Mock,
} from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import { userEvent } from "@testing-library/user-event";
import type { Book } from "../../types";

vi.mock("lucide-svelte", () => ({ Mail: () => {}, X: () => {} }));

const fakeBook: Book = {
  id: "b1",
  title: "The Hobbit",
  description: null,
  asin: null,
  isbn10: null,
  isbn13: null,
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  publication_date: null,
  publisher: null,
  language: null,
  cover_image_url: null,
  authors: [],
  series: [],
  files: [
    {
      id: "f1",
      book_id: "b1",
      file_type: "epub",
      file_name: "hobbit.epub",
      file_size: 1024,
      file_hash: null,
      file_path: "/books/hobbit.epub",
      download_count: 0,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  ],
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const fakeBookNoFiles: Book = { ...fakeBook, files: [] };

let fetchMock: Mock;

beforeEach(() => {
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

// Helper to mock a successful fetch response.
function mockFetch(body: unknown, status = 200) {
  fetchMock.mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    statusText: "OK",
    headers: new Headers({ "content-type": "application/json" }),
    json: vi.fn().mockResolvedValue(body),
    text: vi.fn().mockResolvedValue(JSON.stringify(body)),
  } as unknown as Response);
}

import EmailBookModal from "./EmailBookModal.svelte";

describe("EmailBookModal", () => {
  it("shows loading state initially", () => {
    fetchMock.mockReturnValue(new Promise(() => {})); // never resolves
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });
    expect(screen.getByRole("status")).toHaveTextContent(/loading/i);
  });

  it("moves focus to the dialog on mount", async () => {
    fetchMock.mockReturnValue(new Promise(() => {}));
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });
    const dialog = screen.getByRole("dialog");
    await waitFor(() => expect(dialog).toHaveFocus());
  });

  it("renders the dialog title", async () => {
    mockFetch(fakeBook);
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });
    await waitFor(() =>
      expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent(
        "Email Book",
      ),
    );
  });

  it("shows file name and email input after loading", async () => {
    mockFetch(fakeBook);
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });
    await waitFor(() =>
      expect(screen.getByText(/hobbit\.epub/)).toBeInTheDocument(),
    );
    expect(screen.getByLabelText("To")).toBeInTheDocument();
  });

  it("shows 'no files' message when book has no files", async () => {
    mockFetch(fakeBookNoFiles);
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });
    await waitFor(() =>
      expect(screen.getByText(/no files available/i)).toBeInTheDocument(),
    );
  });

  it("shows an error message when book load fails", async () => {
    fetchMock.mockRejectedValue(new Error("Network error"));
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });
    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("Network error"),
    );
  });

  it("calls onClose when the close button is clicked", async () => {
    mockFetch(fakeBook);
    const onClose = vi.fn();
    render(EmailBookModal, { bookId: "b1", onClose });
    await waitFor(() => screen.getByLabelText("Close"));
    await userEvent.click(screen.getByLabelText("Close"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("shows success message after sending", async () => {
    // First call returns book details, second call returns email success.
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: vi.fn(),
        text: vi.fn().mockResolvedValue(JSON.stringify(fakeBook)),
      } as unknown as Response)
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: vi.fn(),
        text: vi
          .fn()
          .mockResolvedValue(
            JSON.stringify({ message: "Email sent successfully" }),
          ),
      } as unknown as Response);

    const user = userEvent.setup();
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });

    await waitFor(() => screen.getByLabelText("To"));
    await user.type(screen.getByLabelText("To"), "reader@example.com");
    await user.click(screen.getByRole("button", { name: /send/i }));

    await waitFor(() =>
      expect(screen.getByRole("status")).toHaveTextContent(
        "Email sent successfully",
      ),
    );
  });

  it("shows error message when send fails", async () => {
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: vi.fn(),
        text: vi.fn().mockResolvedValue(JSON.stringify(fakeBook)),
      } as unknown as Response)
      .mockResolvedValueOnce({
        ok: false,
        status: 502,
        statusText: "Bad Gateway",
        headers: new Headers({ "content-type": "application/json" }),
        json: vi.fn(),
        text: vi
          .fn()
          .mockResolvedValue(JSON.stringify({ error: "failed to send email" })),
      } as unknown as Response);

    const user = userEvent.setup();
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });

    await waitFor(() => screen.getByLabelText("To"));
    await user.type(screen.getByLabelText("To"), "reader@example.com");
    await user.click(screen.getByRole("button", { name: /send/i }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "failed to send email",
      ),
    );
  });

  it("traps Tab focus within the dialog: Tab from last element wraps to first", async () => {
    mockFetch(fakeBook);
    const user = userEvent.setup();
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });

    await waitFor(() => screen.getByLabelText("To"));

    // Focus the Cancel button (last enabled focusable element; Send is disabled when input is empty)
    const cancelButton = screen.getByRole("button", { name: /cancel/i });
    cancelButton.focus();
    expect(cancelButton).toHaveFocus();

    // Tab from the last element should wrap back to the first focusable element
    await user.keyboard("{Tab}");

    // First focusable element is the Close button
    const closeButton = screen.getByRole("button", { name: /close/i });
    expect(closeButton).toHaveFocus();
  });

  it("traps Shift+Tab focus within the dialog: Shift+Tab from first element wraps to last", async () => {
    mockFetch(fakeBook);
    const user = userEvent.setup();
    render(EmailBookModal, { bookId: "b1", onClose: vi.fn() });

    await waitFor(() => screen.getByLabelText("To"));

    // Focus the Close button (first focusable element)
    const closeButton = screen.getByRole("button", { name: /close/i });
    closeButton.focus();
    expect(closeButton).toHaveFocus();

    // Shift+Tab from the first element should wrap to the last focusable element
    await user.keyboard("{Shift>}{Tab}{/Shift}");

    // Last enabled focusable element is the Cancel button (Send is disabled when input is empty)
    const cancelButton = screen.getByRole("button", { name: /cancel/i });
    expect(cancelButton).toHaveFocus();
  });
});
