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

  it("hides the visual chart from assistive technology", () => {
    render(DownloadsHistogram, { data: dataWithDownloads });
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
    expect(screen.queryByRole("listitem")).not.toBeInTheDocument();
    expect(screen.getByTestId("histogram-bars")).toHaveAttribute(
      "aria-hidden",
      "true",
    );
  });

  it("renders an accessible data table with month and download counts", () => {
    render(DownloadsHistogram, { data: dataWithDownloads });
    const table = screen.getByRole("table", { name: "Downloads per month" });
    expect(table).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Month" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Downloads" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("rowheader", { name: "January 2026" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("rowheader", { name: "February 2026" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("rowheader", { name: "March 2026" }),
    ).toBeInTheDocument();
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
