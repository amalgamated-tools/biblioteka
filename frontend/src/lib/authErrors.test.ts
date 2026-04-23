import { describe, expect, it } from "vitest";
import {
  getLoginFieldInvalidState,
  getSignupFieldInvalidState,
} from "./authErrors";

describe("getLoginFieldInvalidState", () => {
  it("does not mark fields for ambiguous credential errors", () => {
    expect(getLoginFieldInvalidState("Invalid email or password")).toEqual({
      email: false,
      password: false,
    });
  });

  it("marks only email for email-specific errors", () => {
    expect(getLoginFieldInvalidState("Email is not valid")).toEqual({
      email: true,
      password: false,
    });
  });

  it("marks only password for password-specific errors", () => {
    expect(getLoginFieldInvalidState("Incorrect password")).toEqual({
      email: false,
      password: true,
    });
  });

  it("marks both fields for credential-level errors", () => {
    expect(getLoginFieldInvalidState("Invalid credentials")).toEqual({
      email: true,
      password: true,
    });
  });

  it("does not mark fields for generic errors", () => {
    expect(getLoginFieldInvalidState("Authentication failed")).toEqual({
      email: false,
      password: false,
    });
  });
});

describe("getSignupFieldInvalidState", () => {
  it("marks only email for duplicate-email errors", () => {
    expect(getSignupFieldInvalidState("email already exists")).toEqual({
      name: false,
      email: true,
      password: false,
    });
  });

  it("marks only password for password-specific errors", () => {
    expect(getSignupFieldInvalidState("password must contain symbols")).toEqual(
      {
        name: false,
        email: false,
        password: true,
      },
    );
  });

  it("marks only name for name-specific errors", () => {
    expect(getSignupFieldInvalidState("display name is required")).toEqual({
      name: true,
      email: false,
      password: false,
    });
  });

  it("marks all mentioned fields", () => {
    expect(
      getSignupFieldInvalidState(
        "full name is invalid and email is invalid and password is invalid",
      ),
    ).toEqual({
      name: true,
      email: true,
      password: true,
    });
  });

  it("does not mark fields for generic errors", () => {
    expect(getSignupFieldInvalidState("Signup failed")).toEqual({
      name: false,
      email: false,
      password: false,
    });
  });
});
