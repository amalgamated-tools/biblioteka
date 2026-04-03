import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";

vi.mock("lucide-svelte", () => ({ Library: () => {} }));

import MyLibrary from "./MyLibrary.svelte";

describe("MyLibrary", () => {
  afterEach(() => cleanup());

  it("renders the 'My Library' heading", () => {
    render(MyLibrary);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("My Library");
  });

  it("renders the empty-state message", () => {
    render(MyLibrary);
    expect(
      screen.getByText("Your personal library is empty."),
    ).toBeInTheDocument();
  });

  it("renders the secondary hint text", () => {
    render(MyLibrary);
    expect(
      screen.getByText("Browse All Books to add some to your collection."),
    ).toBeInTheDocument();
  });
});
