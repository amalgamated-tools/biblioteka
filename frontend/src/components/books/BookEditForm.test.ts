import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import { userEvent } from "@testing-library/user-event";

vi.mock("../../stores/router.svelte", () => ({
  routerStore: { navigate: vi.fn() },
}));

vi.mock("../../lib/api", () => ({
  updateBook: vi.fn().mockResolvedValue({}),
  rejectMetadata: vi.fn().mockResolvedValue(undefined),
}));

import BookEditForm from "./BookEditForm.svelte";
import type { FormFields } from "./BookEditForm.svelte";

function makeFields(overrides: Partial<FormFields> = {}): FormFields {
  return {
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
    ...overrides,
  };
}

function renderForm(overrides: Record<string, unknown> = {}) {
  return render(BookEditForm, {
    bookId: "b1",
    fields: makeFields(),
    saving: false,
    hasPendingMetadata: false,
    onSaved: vi.fn(),
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

  it("displays the formError AlertBanner when title is blank and form is submitted", async () => {
    const user = userEvent.setup();
    renderForm({ fields: makeFields({ title: "" }) });

    await user.click(screen.getByText("Save Changes"));

    const error = screen.getByText("Title is required");
    expect(error).toBeInTheDocument();
    expect(error.closest('[role="alert"]')).toHaveAttribute(
      "id",
      "book-edit-form-error",
    );
    expect(screen.getByLabelText(/Title/i)).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(screen.getByLabelText(/Title/i)).toHaveAttribute(
      "aria-describedby",
      "book-edit-form-error",
    );
  });

  it("does not display an error banner initially", () => {
    renderForm();
    expect(screen.queryByText("Title is required")).not.toBeInTheDocument();
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
