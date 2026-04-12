import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import { userEvent } from "@testing-library/user-event";
import type { RemoteMetadata } from "../../types";

vi.mock("lucide-svelte", () => ({
  ArrowLeft: () => {},
  Check: () => {},
  X: () => {},
}));

import MetadataComparison from "./MetadataComparison.svelte";

const baseMetadata: RemoteMetadata = {
  id: "m1",
  book_id: "b1",
  status: "pending",
  source: "goodreads",
  title: "New Title",
  description: "New description",
  asin: null,
  isbn10: null,
  isbn13: "9780547928227",
  goodreads_id: "5907",
  hardcover_id: null,
  google_books_id: null,
  publication_date: "1937-09-21",
  publisher: "Allen & Unwin",
  language: "en",
  cover_image_url: null,
  author_name: "J.R.R. Tolkien",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const baseCurrentValues = {
  title: "Old Title",
  description: null,
  publisher: null,
  language: null,
  publication_date: null,
  isbn13: null,
  isbn10: null,
  asin: null,
  goodreads_id: null,
  hardcover_id: null,
  google_books_id: null,
  cover_image_url: null,
};

describe("MetadataComparison", () => {
  afterEach(() => cleanup());

  it("renders the Fetched Metadata heading", () => {
    render(MetadataComparison, {
      metadata: baseMetadata,
      currentValues: baseCurrentValues,
      onApplyField: vi.fn(),
      onApplyAll: vi.fn(),
      onDismiss: vi.fn(),
    });
    expect(
      screen.getByRole("heading", { name: /Fetched Metadata/i }),
    ).toBeInTheDocument();
  });

  it("displays the source label", () => {
    render(MetadataComparison, {
      metadata: baseMetadata,
      currentValues: baseCurrentValues,
      onApplyField: vi.fn(),
      onApplyAll: vi.fn(),
      onDismiss: vi.fn(),
    });
    expect(screen.getByText("Goodreads")).toBeInTheDocument();
  });

  it("only renders rows for non-null metadata fields", () => {
    const metadata: RemoteMetadata = {
      ...baseMetadata,
      title: "Only Title",
      description: null,
      isbn13: null,
      goodreads_id: null,
      publisher: null,
      language: null,
      publication_date: null,
    };
    render(MetadataComparison, {
      metadata,
      currentValues: baseCurrentValues,
      onApplyField: vi.fn(),
      onApplyAll: vi.fn(),
      onDismiss: vi.fn(),
    });
    expect(screen.getByText("Title")).toBeInTheDocument();
    expect(screen.queryByText("Description")).not.toBeInTheDocument();
  });

  it("calls onApplyAll when Apply All button is clicked", async () => {
    const onApplyAll = vi.fn();
    render(MetadataComparison, {
      metadata: baseMetadata,
      currentValues: baseCurrentValues,
      onApplyField: vi.fn(),
      onApplyAll,
      onDismiss: vi.fn(),
    });

    const user = userEvent.setup();
    await user.click(screen.getByText("Apply All"));
    expect(onApplyAll).toHaveBeenCalledOnce();
  });

  it("calls onDismiss when Dismiss button is clicked", async () => {
    const onDismiss = vi.fn();
    render(MetadataComparison, {
      metadata: baseMetadata,
      currentValues: baseCurrentValues,
      onApplyField: vi.fn(),
      onApplyAll: vi.fn(),
      onDismiss,
    });

    const user = userEvent.setup();
    await user.click(screen.getByText("Dismiss"));
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("calls onApplyField when a per-field apply button is clicked", async () => {
    const onApplyField = vi.fn();
    render(MetadataComparison, {
      metadata: baseMetadata,
      currentValues: baseCurrentValues,
      onApplyField,
      onApplyAll: vi.fn(),
      onDismiss: vi.fn(),
    });

    const user = userEvent.setup();
    const applyButtons = screen.getAllByTitle("Use fetched value");
    await user.click(applyButtons[0]);
    expect(onApplyField).toHaveBeenCalledOnce();
  });

  it("does not show apply button when values match", () => {
    const matchingValues = {
      ...baseCurrentValues,
      title: "New Title",
      description: "New description",
      isbn13: "9780547928227",
      goodreads_id: "5907",
      publisher: "Allen & Unwin",
      language: "en",
      publication_date: "1937-09-21",
    };
    render(MetadataComparison, {
      metadata: baseMetadata,
      currentValues: matchingValues,
      onApplyField: vi.fn(),
      onApplyAll: vi.fn(),
      onDismiss: vi.fn(),
    });
    expect(screen.queryByTitle("Use fetched value")).not.toBeInTheDocument();
  });

  it("shows accessible 'Values match' label when a field value matches", () => {
    const matchingValues = {
      ...baseCurrentValues,
      title: "New Title",
    };
    render(MetadataComparison, {
      metadata: baseMetadata,
      currentValues: matchingValues,
      onApplyField: vi.fn(),
      onApplyAll: vi.fn(),
      onDismiss: vi.fn(),
    });
    const matchIndicators = screen.getAllByRole("img", {
      name: "Values match",
    });
    expect(matchIndicators.length).toBeGreaterThan(0);
  });
});
