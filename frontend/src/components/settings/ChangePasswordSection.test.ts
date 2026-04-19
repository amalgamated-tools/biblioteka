import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import { cleanup, render, screen, fireEvent } from "@testing-library/svelte";
import { tick } from "svelte";

vi.mock("../../lib/api", () => ({
  changePassword: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("lucide-svelte", () => ({
  Lock: () => {},
}));

import ChangePasswordSection from "./ChangePasswordSection.svelte";
import { changePassword } from "../../lib/api";

describe("ChangePasswordSection", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders all three password fields", () => {
    render(ChangePasswordSection);

    expect(screen.getByLabelText("Current Password")).toBeInTheDocument();
    expect(screen.getByLabelText("New Password")).toBeInTheDocument();
    expect(screen.getByLabelText("Confirm New Password")).toBeInTheDocument();
  });

  it("renders the Update Password button", () => {
    render(ChangePasswordSection);

    expect(
      screen.getByRole("button", { name: "Update Password" }),
    ).toBeInTheDocument();
  });

  it("shows validation error when current password is empty on submit", async () => {
    render(ChangePasswordSection);

    const form = screen.getByRole("form", { name: "Change password" });
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Current password is required",
    );
    expect(screen.getByLabelText("Current Password")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(screen.getByLabelText("Current Password")).toHaveAttribute(
      "aria-describedby",
      "password-change-error",
    );
  });

  it("shows validation error when new password is empty on submit", async () => {
    render(ChangePasswordSection);

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });

    const form = screen.getByRole("form", { name: "Change password" });
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "New password is required",
    );
    expect(screen.getByLabelText("New Password")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
  });

  it("shows validation error when new password is shorter than 6 characters", async () => {
    render(ChangePasswordSection);

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "abc" },
    });

    const form = screen.getByRole("form", { name: "Change password" });
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "New password must be at least 6 characters",
    );
  });

  it("shows validation error when confirm password does not match", async () => {
    render(ChangePasswordSection);

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "newpassword" },
    });
    await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
      target: { value: "different" },
    });

    const form = screen.getByRole("form", { name: "Change password" });
    await fireEvent.submit(form);
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Passwords do not match",
    );
    expect(screen.getByLabelText("Confirm New Password")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
  });

  it("calls changePassword with correct values on valid submission", async () => {
    render(ChangePasswordSection);

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "newpassword" },
    });
    await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
      target: { value: "newpassword" },
    });

    const form = screen.getByRole("form", { name: "Change password" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(changePassword).toHaveBeenCalledWith("old-pass", "newpassword");
  });

  it("shows success message after password change", async () => {
    render(ChangePasswordSection);

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "newpassword" },
    });
    await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
      target: { value: "newpassword" },
    });

    const form = screen.getByRole("form", { name: "Change password" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(
      screen.getByText("Password updated successfully"),
    ).toBeInTheDocument();
  });

  it("clears password fields after successful change", async () => {
    render(ChangePasswordSection);

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "old-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "newpassword" },
    });
    await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
      target: { value: "newpassword" },
    });

    const form = screen.getByRole("form", { name: "Change password" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByLabelText("Current Password")).toHaveValue("");
    expect(screen.getByLabelText("New Password")).toHaveValue("");
    expect(screen.getByLabelText("Confirm New Password")).toHaveValue("");
  });

  it("shows error banner when changePassword rejects", async () => {
    vi.mocked(changePassword).mockRejectedValueOnce(
      new Error("Current password incorrect"),
    );
    render(ChangePasswordSection);

    await fireEvent.input(screen.getByLabelText("Current Password"), {
      target: { value: "wrong-pass" },
    });
    await fireEvent.input(screen.getByLabelText("New Password"), {
      target: { value: "newpassword" },
    });
    await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
      target: { value: "newpassword" },
    });

    const form = screen.getByRole("form", { name: "Change password" });
    await fireEvent.submit(form);
    await tick();
    await tick();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Current password incorrect",
    );
  });

  describe("success message auto-dismiss", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it("auto-dismisses the success message after 3 seconds", async () => {
      render(ChangePasswordSection);

      await fireEvent.input(screen.getByLabelText("Current Password"), {
        target: { value: "old-pass" },
      });
      await fireEvent.input(screen.getByLabelText("New Password"), {
        target: { value: "newpassword" },
      });
      await fireEvent.input(screen.getByLabelText("Confirm New Password"), {
        target: { value: "newpassword" },
      });

      const form = screen.getByRole("form", { name: "Change password" });
      await fireEvent.submit(form);
      await tick();
      await tick();

      expect(
        screen.getByText("Password updated successfully"),
      ).toBeInTheDocument();

      await vi.advanceTimersByTimeAsync(3000);
      await tick();

      expect(screen.queryByText("Password updated successfully")).toBeNull();
    });
  });
});
