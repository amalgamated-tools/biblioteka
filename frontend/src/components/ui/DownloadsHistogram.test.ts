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
    expect(
      screen.getByRole("heading", { level: 3 }),
    ).toHaveTextContent("Downloads per month");
  });

  it("renders a custom title", () => {
    render(DownloadsHistogram, {
      data: dataWithDownloads,
      title: "My custom title",
    });
    expect(
      screen.getByRole("heading", { level: 3 }),
    ).toHaveTextContent("My custom title");
  });

  it("shows the empty state message when all counts are zero", () => {
    render(DownloadsHistogram, { data: emptyData });
    expect(
      screen.getByText(/no downloads recorded yet/i),
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId("histogram-bars"),
    ).not.toBeInTheDocument();
  });

  it("renders bars when data has downloads", () => {
    render(DownloadsHistogram, { data: dataWithDownloads });
    expect(screen.queryByText(/no downloads recorded yet/i)).not.toBeInTheDocument();
    expect(screen.getByTestId("histogram-bars")).toBeInTheDocument();
  });

  it("renders a bar element for each data point", () => {
    render(DownloadsHistogram, { data: dataWithDownloads });
    const bars = screen.getByTestId("histogram-bars");
    // One child group per data point
    expect(bars.children).toHaveLength(dataWithDownloads.length);
  });
});
