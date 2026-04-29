import { describe, expect, it } from "vitest";
import {
  getLoginRequiredFieldError,
  getSignupRequiredFieldError,
} from "./authRequiredErrors";

describe("getLoginRequiredFieldError", () => {
  it("returns all-fields message when both email and password are missing", () => {
    expect(getLoginRequiredFieldError(true, true)).toBe(
      "Please fill in all fields",
    );
  });

  it("returns email-field message when only email is missing", () => {
    expect(getLoginRequiredFieldError(true, false)).toBe(
      "Please fill in the email field",
    );
  });

  it("returns password-field message when only password is missing", () => {
    expect(getLoginRequiredFieldError(false, true)).toBe(
      "Please fill in the password field",
    );
  });

  it("throws when no fields are missing (programming error guard)", () => {
    expect(() => getLoginRequiredFieldError(false, false)).toThrow(
      "getLoginRequiredFieldError called with no missing fields",
    );
  });
});

describe("getSignupRequiredFieldError", () => {
  const cases: {
    name: string;
    nameMissing: boolean;
    emailMissing: boolean;
    passwordMissing: boolean;
    expected: string;
  }[] = [
    {
      name: "all three fields",
      nameMissing: true,
      emailMissing: true,
      passwordMissing: true,
      expected: "Please fill in all fields",
    },
    {
      name: "name and email",
      nameMissing: true,
      emailMissing: true,
      passwordMissing: false,
      expected: "Please fill in the name and email fields",
    },
    {
      name: "name and password",
      nameMissing: true,
      emailMissing: false,
      passwordMissing: true,
      expected: "Please fill in the name and password fields",
    },
    {
      name: "email and password",
      nameMissing: false,
      emailMissing: true,
      passwordMissing: true,
      expected: "Please fill in the email and password fields",
    },
    {
      name: "name only",
      nameMissing: true,
      emailMissing: false,
      passwordMissing: false,
      expected: "Please fill in the name field",
    },
    {
      name: "email only",
      nameMissing: false,
      emailMissing: true,
      passwordMissing: false,
      expected: "Please fill in the email field",
    },
    {
      name: "password only",
      nameMissing: false,
      emailMissing: false,
      passwordMissing: true,
      expected: "Please fill in the password field",
    },
  ];

  for (const {
    name,
    nameMissing,
    emailMissing,
    passwordMissing,
    expected,
  } of cases) {
    it(`returns correct message when ${name} missing`, () => {
      expect(
        getSignupRequiredFieldError(nameMissing, emailMissing, passwordMissing),
      ).toBe(expected);
    });
  }

  it("throws when no fields are missing (programming error guard)", () => {
    expect(() => getSignupRequiredFieldError(false, false, false)).toThrow(
      "getSignupRequiredFieldError called with no missing fields",
    );
  });
});
