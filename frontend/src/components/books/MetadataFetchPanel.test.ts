import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import { userEvent } from "@testing-library/user-event";
import type { RemoteMetadata } from "../../types";

vi.mock("lucide-svelte", () => ({
  Search: () => {},
  ArrowLeft: () => {},
  ArrowRight: () => {},
  Check: () => {},
  X: () => {},
}));

const mockFetchMetadata = vi.fn();
const mockGetMetadata = vi.fn();
const mockSubscribeToMetadataEvents = vi.fn();

vi.mock("../../lib/api", () => ({
  fetchMetadata: (...args: unknown[]) => mockFetchMetadata(...args),
  getMetadata: (...args: unknown[]) => mockGetMetadata(...args),
  rejectMetadata: vi.fn(),
  subscribeToMetadataEvents: (...args: unknown[]) =>
    mockSubscribeToMetadataEvents(...args),
}));

import MetadataFetchPanel from "./MetadataFetchPanel.svelte";

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

const fakeMetadata: RemoteMetadata = {
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

function createMockEventSource() {
  return {
    onmessage: null as ((event: { data: string }) => void) | null,
    onerror: null as (() => void) | null,
    close: vi.fn(),
  };
}

function renderPanel(metadata: RemoteMetadata | null = null) {
  return render(MetadataFetchPanel, {
    bookId: "b1",
    saving: false,
    metadata,
    currentValues: baseCurrentValues,
    onApplyField: vi.fn(),
    onApplyAll: vi.fn(),
    onDismiss: vi.fn(),
  });
}

describe("MetadataFetchPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    cleanup();
  });

  it("renders the Remote Metadata heading and Fetch button", () => {
    renderPanel();
    expect(
      screen.getByRole("heading", { name: /Remote Metadata/i }),
    ).toBeInTheDocument();
    expect(screen.getByText("Fetch Metadata")).toBeInTheDocument();
  });

  it("short-circuits on already_exists and loads metadata", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "already_exists" });
    mockGetMetadata.mockResolvedValue(fakeMetadata);

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    expect(mockES.close).toHaveBeenCalled();
    expect(mockGetMetadata).toHaveBeenCalledWith("b1");
  });

  it("handles SSE complete event", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "enqueued" });
    mockGetMetadata.mockResolvedValue(fakeMetadata);

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    // Simulate SSE complete event
    mockES.onmessage!({
      data: JSON.stringify({ event: "complete", message: "Done" }),
    });
    await vi.advanceTimersByTimeAsync(0);

    expect(mockES.close).toHaveBeenCalled();
    expect(mockGetMetadata).toHaveBeenCalledWith("b1");
  });

  it("handles SSE error event and shows error message", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "enqueued" });

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    // Simulate SSE error event
    mockES.onmessage!({
      data: JSON.stringify({
        event: "error",
        message: "Provider unavailable",
      }),
    });
    await vi.advanceTimersByTimeAsync(0);

    expect(mockES.close).toHaveBeenCalled();
    await waitFor(() => {
      expect(screen.getByText("Provider unavailable")).toBeInTheDocument();
    });
  });

  it("handles SSE not_found event", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "enqueued" });

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    mockES.onmessage!({
      data: JSON.stringify({
        event: "not_found",
        message: "No metadata found for this book",
      }),
    });
    await vi.advanceTimersByTimeAsync(0);

    await waitFor(() => {
      expect(
        screen.getByText("No metadata found for this book"),
      ).toBeInTheDocument();
    });
  });

  it("falls back to polling on 60-second timeout", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "enqueued" });
    mockGetMetadata.mockResolvedValue(fakeMetadata);

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    // Advance past the 60-second timeout
    await vi.advanceTimersByTimeAsync(60_000);

    expect(mockES.close).toHaveBeenCalled();
    expect(mockGetMetadata).toHaveBeenCalledWith("b1");
  });

  it("handles SSE connection error and tries loading metadata", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "enqueued" });
    mockGetMetadata.mockRejectedValue(new Error("not found"));

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    // Simulate SSE onerror
    mockES.onerror!();
    await vi.advanceTimersByTimeAsync(0);

    expect(mockES.close).toHaveBeenCalled();
    await waitFor(() => {
      expect(
        screen.getByText(
          "Metadata stream closed unexpectedly. Please try again.",
        ),
      ).toBeInTheDocument();
    });
  });

  it("shows error when fetchMetadata API call fails", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockRejectedValue(new Error("Network error"));

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    await waitFor(() => {
      expect(screen.getByText("Network error")).toBeInTheDocument();
    });
  });

  it("cleans up SSE on unmount", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "enqueued" });

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const { unmount } = renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    unmount();

    expect(mockES.close).toHaveBeenCalled();
  });

  it("shows progress message during fetch", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "enqueued" });

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    expect(screen.getByText("Starting metadata fetch...")).toBeInTheDocument();

    // Simulate progress update
    mockES.onmessage!({
      data: JSON.stringify({ event: "progress", message: "Searching..." }),
    });
    await vi.advanceTimersByTimeAsync(0);

    await waitFor(() => {
      expect(screen.getByText("Searching...")).toBeInTheDocument();
    });
  });

  it("shows already_running progress message", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "already_running" });

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    await waitFor(() => {
      expect(
        screen.getByText("Metadata fetch already in progress..."),
      ).toBeInTheDocument();
    });
  });

  it("closes the previous SSE stream when bookId changes and ignores stale events", async () => {
    const previousES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(previousES);
    mockFetchMetadata.mockResolvedValue({ status: "enqueued" });

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const view = renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    expect(mockSubscribeToMetadataEvents).toHaveBeenCalledTimes(1);
    expect(mockFetchMetadata).toHaveBeenCalledTimes(1);

    // Simulate bookId prop change (navigating to a different book edit page)
    await view.rerender({
      bookId: "book-2",
      saving: false,
      metadata: null,
      currentValues: baseCurrentValues,
      onApplyField: vi.fn(),
      onApplyAll: vi.fn(),
      onDismiss: vi.fn(),
    });
    await vi.advanceTimersByTimeAsync(0);

    expect(previousES.close).toHaveBeenCalled();

    // Stale events from the old stream should not update the UI
    mockGetMetadata.mockClear();
    previousES.onmessage?.({
      data: JSON.stringify({
        event: "progress",
        message: "Old stream should be ignored",
      }),
    });
    previousES.onmessage?.({
      data: JSON.stringify({ event: "complete" }),
    });
    await vi.advanceTimersByTimeAsync(0);

    expect(
      screen.queryByText("Old stream should be ignored"),
    ).not.toBeInTheDocument();
    // Stale complete event must not trigger a metadata fetch for the new bookId
    expect(mockGetMetadata).not.toHaveBeenCalled();
    // No new SSE subscription or fetch was triggered by the bookId change
    expect(mockSubscribeToMetadataEvents).toHaveBeenCalledTimes(1);
    expect(mockFetchMetadata).toHaveBeenCalledTimes(1);
  });

  it("ignores stale metadata fallback results after bookId rerender", async () => {
    const mockES = createMockEventSource();
    mockSubscribeToMetadataEvents.mockReturnValue(mockES);
    mockFetchMetadata.mockResolvedValue({ status: "enqueued" });

    let resolveMetadata!: (value: RemoteMetadata) => void;
    const pendingMetadata = new Promise<RemoteMetadata>((resolve) => {
      resolveMetadata = resolve;
    });
    mockGetMetadata.mockImplementation((id: string) => {
      if (id === "b1") {
        return pendingMetadata;
      }
      return Promise.reject(new Error("not found"));
    });

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const { rerender } = renderPanel();

    await user.click(screen.getByText("Fetch Metadata"));
    await vi.advanceTimersByTimeAsync(0);

    // Trigger the async fallback load for the original book.
    mockES.onerror!();

    await waitFor(() => {
      expect(mockGetMetadata).toHaveBeenCalledWith("b1");
    });

    await rerender({
      bookId: "b2",
      saving: false,
      metadata: null,
      currentValues: baseCurrentValues,
      onApplyField: vi.fn(),
      onApplyAll: vi.fn(),
      onDismiss: vi.fn(),
    });

    // Resolve the old book's metadata request after the component has rerendered.
    resolveMetadata(fakeMetadata);
    await vi.advanceTimersByTimeAsync(0);

    expect(
      screen.queryByText(
        "Metadata stream closed unexpectedly. Please try again.",
      ),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(fakeMetadata.title!)).not.toBeInTheDocument();
  });
});
