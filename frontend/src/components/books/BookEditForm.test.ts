import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import { userEvent } from "@testing-library/user-event";

import BookEditForm from "./BookEditForm.svelte";

function renderForm(overrides: Record<string, unknown> = {}) {
  return render(BookEditForm, {
    title: "The Hobbit",
    description: "A fantasy novel",
    publisher: "Allen & Unwin",
    language: "en",
    publicationDate: "1937-09-21",
    isbn13: "9780547928227",
    isbn10: "0547928227",
    asin: "B007978NPG",
    goodreadsId: "5907",
    hardcoverId: "hc-123",
    googleBooksId: "gb-456",
    coverImageUrl: "https://example.com/cover.jpg",
    saving: false,
    formError: null,
    onsubmit: vi.fn(),
    oncancel: vi.fn(),
    ...overrides,
  });
}

describe("BookEditForm", () => {
  afterEach(() => cleanup());

  it("renders all 12 form fields with correct values", () => {
    renderForm();

    const fields: Array<[string, string]> = [
      ["Book title", "The Hobbit"],
      ["Book description", "A fantasy novel"],
      ["Publisher", "Allen & Unwin"],
      ["Language", "en"],
      ["YYYY-MM-DD", "1937-09-21"],
      ["ISBN-13", "9780547928227"],
      ["ISBN-10", "0547928227"],
      ["ASIN", "B007978NPG"],
      ["Goodreads ID", "5907"],
      ["Hardcover ID", "hc-123"],
      ["Google Books ID", "gb-456"],
      ["https://...", "https://example.com/cover.jpg"],
    ];

    for (const [placeholder, value] of fields) {
      const input = screen.getByPlaceholderText(placeholder) as
        | HTMLInputElement
        | HTMLTextAreaElement;
      expect(input.value).toBe(value);
    }
  });

  it("displays the formError AlertBanner when formError is set", () => {
    renderForm({ formError: "Title is required" });
    expect(screen.getByText("Title is required")).toBeInTheDocument();
  });

  it("does not display an error banner when formError is null", () => {
    renderForm({ formError: null });
    expect(screen.queryByText("Title is required")).not.toBeInTheDocument();
  });

  it("calls onsubmit when the form is submitted", async () => {
    const onsubmit = vi.fn((e: SubmitEvent) => e.preventDefault());
    renderForm({ onsubmit });

    const user = userEvent.setup();
    await user.click(screen.getByText("Save Changes"));

    expect(onsubmit).toHaveBeenCalledOnce();
  });

  it("calls oncancel when the cancel button is clicked", async () => {
    const oncancel = vi.fn();
    renderForm({ oncancel });

    const user = userEvent.setup();
    await user.click(screen.getByText("Cancel"));

    expect(oncancel).toHaveBeenCalledOnce();
  });

  it("disables all inputs and buttons when saving is true", () => {
    renderForm({ saving: true });

    const inputs = screen.getAllByRole("textbox");
    for (const input of inputs) {
      expect(input).toBeDisabled();
    }

    expect(screen.getByText("Saving...")).toBeInTheDocument();
    expect(screen.getByText("Saving...").closest("button")).toBeDisabled();
    expect(screen.getByText("Cancel").closest("button")).toBeDisabled();
  });
});
