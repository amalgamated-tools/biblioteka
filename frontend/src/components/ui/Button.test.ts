import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { createRawSnippet } from "svelte";
import Button from "./Button.svelte";

function makeChildren(text: string) {
  return createRawSnippet(() => ({
    render: () => `<span>${text}</span>`,
  }));
}

describe("Button", () => {
  afterEach(() => cleanup());

  it("renders a button element", () => {
    render(Button, { children: makeChildren("Click me") });
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("renders children content", () => {
    render(Button, { children: makeChildren("My Label") });
    expect(screen.getByRole("button")).toHaveTextContent("My Label");
  });

  it("defaults to type='button'", () => {
    render(Button, { children: makeChildren("Click") });
    expect(screen.getByRole("button")).toHaveAttribute("type", "button");
  });

  it("forwards the type prop to the button element", () => {
    render(Button, { children: makeChildren("Submit"), type: "submit" });
    expect(screen.getByRole("button")).toHaveAttribute("type", "submit");
  });

  it("applies primary variant gradient classes by default", () => {
    render(Button, { children: makeChildren("Primary") });
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("bg-gradient-to-r");
    expect(btn.className).toContain("from-accent-600");
  });

  it("applies secondary variant classes", () => {
    render(Button, { children: makeChildren("Secondary"), variant: "secondary" });
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("border");
    expect(btn.className).not.toContain("bg-gradient-to-r");
  });

  it("applies danger variant classes", () => {
    render(Button, { children: makeChildren("Danger"), variant: "danger" });
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("bg-danger-600");
  });

  it("is not disabled by default", () => {
    render(Button, { children: makeChildren("Click") });
    expect(screen.getByRole("button")).not.toBeDisabled();
  });

  it("disables the button when disabled prop is true", () => {
    render(Button, { children: makeChildren("Disabled"), disabled: true });
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("includes the disabled cursor Tailwind variant class", () => {
    render(Button, { children: makeChildren("Disabled") });
    expect(screen.getByRole("button").className).toContain("disabled:cursor-not-allowed");
  });

  it("calls onclick handler when clicked", async () => {
    const onclick = vi.fn();
    render(Button, { children: makeChildren("Click"), onclick });
    await fireEvent.click(screen.getByRole("button"));
    expect(onclick).toHaveBeenCalledTimes(1);
  });

  it("does not fire onclick when button is disabled (userEvent respects disabled)", async () => {
    const onclick = vi.fn();
    render(Button, { children: makeChildren("Disabled"), disabled: true, onclick });
    await userEvent.setup().click(screen.getByRole("button"));
    expect(onclick).not.toHaveBeenCalled();
  });

  it("appends the extra class string to the button", () => {
    render(Button, { children: makeChildren("Styled"), class: "my-extra-class" });
    expect(screen.getByRole("button").className).toContain("my-extra-class");
  });
});
