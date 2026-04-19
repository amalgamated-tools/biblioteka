import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../stores/auth.svelte", () => ({
  authStore: {
    user: {
      id: "u1",
      name: "Test User",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    },
  },
}));

vi.mock("../../lib/api", () => ({
  updateProfile: vi.fn().mockResolvedValue({
    id: "u1",
    name: "Test User",
    email: "test@example.com",
    oidc_linked: false,
    is_admin: false,
  }),
}));

vi.mock("lucide-svelte", () => ({
  User: () => {},
}));

import DisplayNameSection from "./DisplayNameSection.svelte";
import { authStore } from "../../stores/auth.svelte";
import { updateProfile } from "../../lib/api";

describe("DisplayNameSection", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("displays the current user name in the name input", () => {
    render(DisplayNameSection);

    const nameInput = screen.getByLabelText("Name");
    expect(nameInput).toHaveValue("Test User");
  });

  it("renders the Save Name button", () => {
    render(DisplayNameSection);

    expect(
      screen.getByRole("button", { name: "Save Name" }),
    ).toBeInTheDocument();
  });

  it("calls updateProfile with new name on valid submission", async () => {
    render(DisplayNameSection);

    const nameInput = screen.getByLabelText("Name");
    await fireEvent.input(nameInput, { target: { value: "New Name" } });

    const form = screen.getByRole("form", { name: "Update display name" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(updateProfile).toHaveBeenCalledWith("New Name");
  });

  it("shows success message after name update", async () => {
    render(DisplayNameSection);

    const nameInput = screen.getByLabelText("Name");
    await fireEvent.input(nameInput, { target: { value: "Updated Name" } });

    const form = screen.getByRole("form", { name: "Update display name" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByText("Display name updated")).toBeInTheDocument();
  });

  it("shows validation error when name is empty on submit", async () => {
    render(DisplayNameSection);

    const nameInput = screen.getByLabelText("Name");
    await fireEvent.input(nameInput, { target: { value: "" } });

    const form = screen.getByRole("form", { name: "Update display name" });
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Display name is required",
    );
    expect(screen.getByLabelText("Name")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(screen.getByLabelText("Name")).toHaveAttribute(
      "aria-describedby",
      "display-name-error",
    );
  });

  it("shows error banner when updateProfile rejects", async () => {
    vi.mocked(updateProfile).mockRejectedValueOnce(
      new Error("Failed to update"),
    );
    render(DisplayNameSection);

    const form = screen.getByRole("form", { name: "Update display name" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent("Failed to update");
  });

  it("updates the authStore user name after successful profile update", async () => {
    vi.mocked(updateProfile).mockResolvedValueOnce({
      id: "u1",
      name: "New Name",
      email: "test@example.com",
      oidc_linked: false,
      is_admin: false,
    });
    render(DisplayNameSection);

    const nameInput = screen.getByLabelText("Name");
    await fireEvent.input(nameInput, { target: { value: "New Name" } });

    const form = screen.getByRole("form", { name: "Update display name" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(vi.mocked(authStore).user?.name).toBe("New Name");
  });

  describe("success message auto-dismiss", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it("auto-dismisses the success message after 3 seconds", async () => {
      render(DisplayNameSection);

      const nameInput = screen.getByLabelText("Name");
      await fireEvent.input(nameInput, { target: { value: "Updated Name" } });

      const form = screen.getByRole("form", { name: "Update display name" });
      await fireEvent.submit(form);
      await tick();
      await tick();

      expect(screen.getByText("Display name updated")).toBeInTheDocument();

      await vi.advanceTimersByTimeAsync(3000);
      await tick();

      expect(screen.queryByText("Display name updated")).toBeNull();
    });
  });
});
