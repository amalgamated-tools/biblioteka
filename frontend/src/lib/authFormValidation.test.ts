import { describe, expect, it } from "vitest";
import { validateAuthForm } from "./authFormValidation";

describe("validateAuthForm", () => {
  it("returns required error for empty login fields", () => {
    expect(
      validateAuthForm({ isLogin: true, name: "", email: "", password: "" }),
    ).toMatchObject({
      error: "Please fill in all fields",
      loginEmailInvalid: true,
      loginPasswordInvalid: true,
    });
  });

  it("returns required error for missing signup fields", () => {
    expect(
      validateAuthForm({
        isLogin: false,
        name: "Reader",
        email: "",
        password: "",
      }),
    ).toMatchObject({
      error: "Please fill in the email and password fields",
      signupEmailInvalid: true,
      signupPasswordInvalid: true,
    });
  });

  it("returns password validation error when password is too short", () => {
    expect(
      validateAuthForm({
        isLogin: true,
        name: "",
        email: "reader@example.com",
        password: "12345",
      }),
    ).toMatchObject({
      error: "Password must be at least 6 characters",
      loginPasswordInvalid: true,
    });
  });

  it("returns email validation error for invalid email", () => {
    expect(
      validateAuthForm({
        isLogin: false,
        name: "Reader",
        email: "not-an-email",
        password: "securepass",
      }),
    ).toMatchObject({
      error: "Please enter a valid email address",
      signupEmailInvalid: true,
    });
  });

  it("returns no error for valid login fields", () => {
    expect(
      validateAuthForm({
        isLogin: true,
        name: "",
        email: "reader@example.com",
        password: "securepass",
      }),
    ).toEqual({
      error: null,
      loginEmailInvalid: false,
      loginPasswordInvalid: false,
      signupNameInvalid: false,
      signupEmailInvalid: false,
      signupPasswordInvalid: false,
    });
  });
});
