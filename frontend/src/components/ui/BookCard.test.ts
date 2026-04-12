import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import { userEvent } from "@testing-library/user-event";
import type { BookSummary } from "../../types";

vi.mock("lucide-svelte", () => ({ BookOpen: () => {}, Mail: () => {} }));

let emailModalRendered = false;
let emailModalOnClose: (() => void) | null = null;
vi.mock("./EmailBookModal.svelte", () => ({
  default: (_anchor: unknown, props: { onClose?: () => void }) => {
    emailModalRendered = true;
    emailModalOnClose = props?.onClose ?? null;
    return {};
  },
}));

import BookCard from "./BookCard.svelte";

const baseBook: BookSummary = {
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
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("BookCard", () => {
  afterEach(() => {
    cleanup();
    emailModalRendered = false;
    emailModalOnClose = null;
  });

  it("renders the book title in a heading", () => {
    render(BookCard, { book: baseBook });
    expect(screen.getByRole("heading", { level: 3 })).toHaveTextContent(
      "The Hobbit",
    );
  });

  it("sets title attribute on the heading for truncation tooltip", () => {
    render(BookCard, { book: baseBook });
    expect(screen.getByRole("heading", { level: 3 })).toHaveAttribute(
      "title",
      "The Hobbit",
    );
  });

  it("renders the cover image when cover_image_url is set", () => {
    const book = {
      ...baseBook,
      cover_image_url: "https://example.com/cover.jpg",
    };
    render(BookCard, { book });
    const img = screen.getByRole("img");
    expect(img).toHaveAttribute("src", "https://example.com/cover.jpg");
    expect(img).toHaveAttribute("alt", "The Hobbit");
  });

  it("sets loading='lazy' on the cover image", () => {
    const book = {
      ...baseBook,
      cover_image_url: "https://example.com/cover.jpg",
    };
    render(BookCard, { book });
    expect(screen.getByRole("img")).toHaveAttribute("loading", "lazy");
  });

  it("does not render an img element when cover_image_url is null", () => {
    render(BookCard, { book: baseBook });
    expect(screen.queryByRole("img")).toBeNull();
  });

  it("renders the publisher name when provided", () => {
    const book = { ...baseBook, publisher: "Allen & Unwin" };
    render(BookCard, { book });
    expect(screen.getByText("Allen & Unwin")).toBeInTheDocument();
  });

  it("does not render a publisher element when publisher is null", () => {
    render(BookCard, { book: baseBook });
    expect(screen.queryByText("Allen & Unwin")).toBeNull();
  });

  it("renders as a link to the book detail view", () => {
    render(BookCard, { book: baseBook });
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "#books/b1");
  });

  it("renders an email button with accessible label", () => {
    render(BookCard, { book: baseBook });
    expect(
      screen.getByRole("button", { name: /email the hobbit/i }),
    ).toBeInTheDocument();
  });

  it("email button has title attribute", () => {
    render(BookCard, { book: baseBook });
    expect(
      screen.getByRole("button", { name: /email the hobbit/i }),
    ).toHaveAttribute("title", "Email this book");
  });

  it("clicking the email button renders the email modal", async () => {
    const user = userEvent.setup();
    render(BookCard, { book: baseBook });
    expect(emailModalRendered).toBe(false);
    const btn = screen.getByRole("button", { name: /email the hobbit/i });
    await user.click(btn);
    expect(emailModalRendered).toBe(true);
  });

  it("returns focus to the email button when the modal closes", async () => {
    const user = userEvent.setup();
    render(BookCard, { book: baseBook });
    const btn = screen.getByRole("button", { name: /email the hobbit/i });
    await user.click(btn);
    expect(emailModalOnClose).toBeTypeOf("function");

    // Move focus away so we can verify it returns to the button
    const outsideFocusTarget = document.createElement("button");
    outsideFocusTarget.type = "button";
    document.body.appendChild(outsideFocusTarget);
    outsideFocusTarget.focus();
    expect(outsideFocusTarget).toHaveFocus();

    // Simulate the modal calling onClose
    await emailModalOnClose!();
    expect(btn).toHaveFocus();

    outsideFocusTarget.remove();
  });
});
