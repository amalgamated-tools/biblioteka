import { describe, expect, it, afterEach } from "vitest";
import { cleanup, render, screen } from "@testing-library/svelte";
import TextInput from "./TextInput.svelte";

describe("TextInput", () => {
  afterEach(() => cleanup());

  it("renders an input element", () => {
    render(TextInput);
    expect(screen.getByRole("textbox")).toBeInTheDocument();
  });

  it("defaults to type='text'", () => {
    render(TextInput);
    expect(screen.getByRole("textbox")).toHaveAttribute("type", "text");
  });

  it("forwards the type prop (email)", () => {
    render(TextInput, { type: "email" });
    expect(screen.getByRole("textbox")).toHaveAttribute("type", "email");
  });

  it("forwards the type prop (password) – not role=textbox", () => {
    const { container } = render(TextInput, { type: "password" });
    expect(container.querySelector("input")).toHaveAttribute(
      "type",
      "password",
    );
  });

  it("is not disabled by default", () => {
    render(TextInput);
    expect(screen.getByRole("textbox")).not.toBeDisabled();
  });

  it("disables the input when the disabled prop is true", () => {
    render(TextInput, { disabled: true });
    expect(screen.getByRole("textbox")).toBeDisabled();
  });

  it("applies cursor-not-allowed class when disabled", () => {
    const { container } = render(TextInput, { disabled: true });
    expect(container.querySelector("input")!.className).toContain(
      "cursor-not-allowed",
    );
  });

  it("applies focus-ring class when enabled", () => {
    const { container } = render(TextInput);
    expect(container.querySelector("input")!.className).toContain(
      "focus:ring-2",
    );
  });

  it("forwards placeholder attribute", () => {
    render(TextInput, { placeholder: "Search..." });
    expect(screen.getByRole("textbox")).toHaveAttribute(
      "placeholder",
      "Search...",
    );
  });

  it("forwards aria-describedby attribute", () => {
    render(TextInput, { "aria-describedby": "hint-text" });
    expect(screen.getByRole("textbox")).toHaveAttribute(
      "aria-describedby",
      "hint-text",
    );
  });

  it("forwards aria-invalid attribute", () => {
    render(TextInput, { "aria-invalid": true });
    expect(screen.getByRole("textbox")).toHaveAttribute("aria-invalid", "true");
  });

  it("forwards id attribute", () => {
    render(TextInput, { id: "my-input" });
    expect(screen.getByRole("textbox")).toHaveAttribute("id", "my-input");
  });

  it("uses ink-400 border for sufficient contrast (WCAG 1.4.11)", () => {
    const { container } = render(TextInput);
    const classes = container.querySelector("input")!.className;
    expect(classes).toContain("border-ink-400");
    expect(classes).not.toContain("border-ink-200");
  });

  it("uses ink-400 dark border for sufficient contrast (WCAG 1.4.11)", () => {
    const { container } = render(TextInput);
    const classes = container.querySelector("input")!.className;
    expect(classes).toContain("dark:border-ink-400");
    expect(classes).not.toContain("dark:border-ink-700");
  });

  it("includes invalid-state styles for aria-invalid inputs", () => {
    const { container } = render(TextInput);
    const classes = container.querySelector("input")!.className;
    expect(classes).toContain("aria-invalid:border-danger-500");
    expect(classes).toContain("aria-invalid:focus:ring-danger-500");
    expect(classes).toContain("aria-invalid:focus:border-danger-500");
  });

  it("uses ink-300 for dark-mode placeholder contrast (WCAG 1.4.3)", () => {
    const { container } = render(TextInput);
    const classes = container.querySelector("input")!.className;
    expect(classes).toContain("dark:placeholder:text-ink-300");
    expect(classes).not.toContain("dark:placeholder:text-ink-500");
  });

  it("appends extra class to the input element", () => {
    const { container } = render(TextInput, { class: "w-full py-2" });
    expect(container.querySelector("input")!.className).toContain(
      "w-full py-2",
    );
  });
});
