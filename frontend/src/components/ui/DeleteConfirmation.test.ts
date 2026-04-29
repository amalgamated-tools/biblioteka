import { describe, expect, it, vi, afterEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import DeleteConfirmation from "./DeleteConfirmation.svelte";

vi.mock("../../lib/actions", () => ({
  autofocusFirstButton: () => ({ destroy: () => {} }),
}));

describe("DeleteConfirmation", () => {
  afterEach(() => cleanup());

  it("renders the item name in the confirmation label", () => {
    render(DeleteConfirmation, {
      itemId: "book-1",
      itemName: "My Book",
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    });

    expect(screen.getByText(`Delete "My Book"?`)).toBeInTheDocument();
  });

  it("has role=group on the root element", () => {
    render(DeleteConfirmation, {
      itemId: "book-1",
      itemName: "My Book",
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    });

    expect(screen.getByRole("group")).toBeInTheDocument();
  });

  it("root element aria-labelledby matches the item label id", () => {
    render(DeleteConfirmation, {
      itemId: "book-1",
      itemName: "My Book",
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
    });

    const group = screen.getByRole("group");
    expect(group).toHaveAttribute(
      "aria-labelledby",
      "delete-confirm-label-book-1",
    );

    const label = document.getElementById("delete-confirm-label-book-1");
    expect(label).toBeInTheDocument();
    expect(label).toHaveTextContent(`Delete "My Book"?`);
  });

  it("calls onConfirm when the Delete button is clicked", async () => {
    const onConfirm = vi.fn();
    render(DeleteConfirmation, {
      itemId: "key-1",
      itemName: "CI Pipeline",
      onConfirm,
      onCancel: vi.fn(),
    });

    await fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel when the Cancel button is clicked", async () => {
    const onCancel = vi.fn();
    render(DeleteConfirmation, {
      itemId: "key-1",
      itemName: "CI Pipeline",
      onConfirm: vi.fn(),
      onCancel,
    });

    await fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("applies the class prop to the root element", () => {
    render(DeleteConfirmation, {
      itemId: "book-1",
      itemName: "My Book",
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
      class: "custom-class",
    });

    expect(screen.getByRole("group")).toHaveClass("custom-class");
  });
});
