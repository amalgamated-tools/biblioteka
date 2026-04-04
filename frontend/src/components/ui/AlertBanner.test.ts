import { describe, expect, it, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import { createRawSnippet } from "svelte";
import AlertBanner from "./AlertBanner.svelte";

function makeChildren(text: string) {
  return createRawSnippet(() => ({
    render: () => `<span>${text}</span>`,
  }));
}

describe("AlertBanner", () => {
  afterEach(() => cleanup());

  it("renders with role='alert' for error variant", () => {
    render(AlertBanner, { variant: "error", children: makeChildren("Error!") });
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("renders with role='status' for success variant", () => {
    render(AlertBanner, {
      variant: "success",
      children: makeChildren("Done!"),
    });
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("sets aria-live='assertive' for error variant", () => {
    render(AlertBanner, { variant: "error", children: makeChildren("Error!") });
    expect(screen.getByRole("alert")).toHaveAttribute("aria-live", "assertive");
  });

  it("sets aria-live='polite' for success variant", () => {
    render(AlertBanner, { variant: "success", children: makeChildren("OK") });
    expect(screen.getByRole("status")).toHaveAttribute("aria-live", "polite");
  });

  it("overrides role with the role prop", () => {
    render(AlertBanner, {
      variant: "error",
      role: "status",
      children: makeChildren("Custom"),
    });
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("overriding role to alert on success sets aria-live='assertive'", () => {
    render(AlertBanner, {
      variant: "success",
      role: "alert",
      children: makeChildren("Custom"),
    });
    expect(screen.getByRole("alert")).toHaveAttribute("aria-live", "assertive");
  });

  it("sets data-testid when testId prop is provided", () => {
    const { container } = render(AlertBanner, {
      variant: "success",
      testId: "my-banner",
      children: makeChildren("Success"),
    });
    expect(
      container.querySelector('[data-testid="my-banner"]'),
    ).toBeInTheDocument();
  });

  it("renders children content", () => {
    render(AlertBanner, {
      variant: "error",
      children: makeChildren("Something went wrong"),
    });
    expect(screen.getByRole("alert")).toHaveTextContent("Something went wrong");
  });

  it("applies error styling classes", () => {
    render(AlertBanner, { variant: "error", children: makeChildren("Error") });
    expect(screen.getByRole("alert").className).toContain("bg-danger-50");
  });

  it("applies success styling classes", () => {
    render(AlertBanner, { variant: "success", children: makeChildren("OK") });
    expect(screen.getByRole("status").className).toContain("bg-success-50");
  });

  it("appends the extra class string", () => {
    render(AlertBanner, {
      variant: "error",
      class: "extra-class",
      children: makeChildren("Error"),
    });
    expect(screen.getByRole("alert").className).toContain("extra-class");
  });
});
