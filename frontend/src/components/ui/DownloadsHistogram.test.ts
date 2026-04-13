import { describe, expect, it, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import DownloadsHistogram from "./DownloadsHistogram.svelte";
import type { MonthlyDownloads } from "../../types";

const emptyData: MonthlyDownloads[] = [
  { month: "2026-01", count: 0 },
  { month: "2026-02", count: 0 },
  { month: "2026-03", count: 0 },
];

const dataWithDownloads: MonthlyDownloads[] = [
  { month: "2026-01", count: 2 },
  { month: "2026-02", count: 5 },
  { month: "2026-03", count: 1 },
];

describe("DownloadsHistogram", () => {
  afterEach(() => cleanup());

  it("renders the default title", () => {
    render(DownloadsHistogram, { data: dataWithDownloads });
    expect(screen.getByRole("heading", { level: 3 })).toHaveTextContent(
      "Downloads per month",
    );
  });

  it("renders a custom title", () => {
    render(DownloadsHistogram, {
      data: dataWithDownloads,
      title: "My custom title",
    });
    expect(screen.getByRole("heading", { level: 3 })).toHaveTextContent(
      "My custom title",
    );
  });

  it("shows the empty state message when all counts are zero", () => {
    render(DownloadsHistogram, { data: emptyData });
    expect(screen.getByText(/no downloads recorded yet/i)).toBeInTheDocument();
    expect(screen.queryByTestId("histogram-bars")).not.toBeInTheDocument();
  });

  it("renders bars when data has downloads", () => {
    render(DownloadsHistogram, { data: dataWithDownloads });
    expect(
      screen.queryByText(/no downloads recorded yet/i),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("histogram-bars")).toBeInTheDocument();
  });

  it("renders a bar element for each data point", () => {
    render(DownloadsHistogram, { data: dataWithDownloads });
    const bars = screen.getByTestId("histogram-bars");
    // One child group per data point
    expect(bars.children).toHaveLength(dataWithDownloads.length);
  });

  it("applies a high-contrast focus-visible outline and does not suppress the focus ring on listitem elements", () => {
    render(DownloadsHistogram, { data: dataWithDownloads });
    const items = screen.getAllByRole("listitem");
    for (const item of items) {
      // Must have a high-contrast focus-visible outline for WCAG 2.4.11 / 2.4.7
      expect(item.className).toContain("focus-visible:outline-accent-600");
      // Must NOT suppress the outline ring
      expect(item.className).not.toContain("focus-within:outline-none");
    }
  });

  it("links the list aria-labelledby to the heading id", () => {
    render(DownloadsHistogram, { data: dataWithDownloads });
    const heading = screen.getByRole("heading", { level: 3 });
    const list = screen.getByRole("list");
    const headingId = heading.getAttribute("id");
    expect(headingId).toBeTruthy();
    expect(list.getAttribute("aria-labelledby")).toBe(headingId);
  });

  it("generates unique ids for multiple instances", () => {
    render(DownloadsHistogram, { data: dataWithDownloads, title: "First" });
    render(DownloadsHistogram, { data: dataWithDownloads, title: "Second" });
    const headings = screen.getAllByRole("heading", { level: 3 });
    const id1 = headings[0].getAttribute("id");
    const id2 = headings[1].getAttribute("id");
    expect(id1).toBeTruthy();
    expect(id2).toBeTruthy();
    expect(id1).not.toBe(id2);
  });
});
