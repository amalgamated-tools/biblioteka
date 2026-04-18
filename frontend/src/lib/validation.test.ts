import { describe, it, expect } from "vitest";
import { required, minLength, matches, email, validate } from "./validation";

describe("required", () => {
  it("returns null for a non-empty value", () => {
    expect(required()("hello")).toBeNull();
  });

  it("returns an error for an empty string", () => {
    expect(required()("")).not.toBeNull();
  });

  it("returns an error for a whitespace-only string", () => {
    expect(required()("   ")).not.toBeNull();
  });

  it("uses the default message when none is supplied", () => {
    expect(required()("")).toBe("This field is required");
  });

  it("uses the custom message when supplied", () => {
    expect(required("Name is required")("")).toBe("Name is required");
  });

  it("passes a value that is non-empty after trimming", () => {
    expect(required()("  hello  ")).toBeNull();
  });
});

describe("minLength", () => {
  it("returns null when the value meets the minimum length", () => {
    expect(minLength(6)("abcdef")).toBeNull();
  });

  it("returns null when the value exceeds the minimum length", () => {
    expect(minLength(6)("abcdefg")).toBeNull();
  });

  it("returns an error when the value is shorter than the minimum", () => {
    expect(minLength(6)("abc")).not.toBeNull();
  });

  it("uses the default message when none is supplied", () => {
    expect(minLength(6)("abc")).toBe("Must be at least 6 characters");
  });

  it("uses the custom message when supplied", () => {
    expect(minLength(6, "Password must be at least 6 characters")("abc")).toBe(
      "Password must be at least 6 characters",
    );
  });

  it("returns an error for an empty string", () => {
    expect(minLength(1)("")).not.toBeNull();
  });
});

describe("matches", () => {
  it("returns null when the values are equal", () => {
    expect(matches("secret")("secret")).toBeNull();
  });

  it("returns an error when the values differ", () => {
    expect(matches("secret")("wrong")).not.toBeNull();
  });

  it("uses the default message when none is supplied", () => {
    expect(matches("a")("b")).toBe("Values do not match");
  });

  it("uses the custom message when supplied", () => {
    expect(matches("a", "Passwords do not match")("b")).toBe(
      "Passwords do not match",
    );
  });

  it("is case-sensitive", () => {
    expect(matches("Secret")("secret")).not.toBeNull();
  });
});

describe("validate", () => {
  it("returns null when all rules pass", () => {
    expect(validate("hello", [required(), minLength(3)])).toBeNull();
  });

  it("returns the first error when the first rule fails", () => {
    expect(validate("", [required("Required"), minLength(3)])).toBe("Required");
  });

  it("returns the error from the first failing rule in order", () => {
    expect(validate("ab", [required(), minLength(3, "Too short")])).toBe(
      "Too short",
    );
  });

  it("returns null for an empty rules array", () => {
    expect(validate("", [])).toBeNull();
  });

  it("stops at the first failing rule and does not evaluate later rules", () => {
    let secondRuleCalled = false;
    const secondRule = (v: string) => {
      secondRuleCalled = true;
      return v === "bad" ? "bad value" : null;
    };
    validate("", [required("Required"), secondRule]);
    expect(secondRuleCalled).toBe(false);
  });
});

describe("email", () => {
  it("returns null for a valid email address", () => {
    expect(email()("user@example.com")).toBeNull();
  });

  it("returns null for a valid email with subdomains", () => {
    expect(email()("user@mail.example.co.uk")).toBeNull();
  });

  it("returns an error for a value with no @", () => {
    expect(email()("notanemail")).not.toBeNull();
  });

  it("returns an error for a value missing the domain part", () => {
    expect(email()("user@")).not.toBeNull();
  });

  it("returns an error for a value missing the TLD", () => {
    expect(email()("user@example")).not.toBeNull();
  });

  it("returns an error for an empty string", () => {
    expect(email()("")).not.toBeNull();
  });

  it("uses the default message when none is supplied", () => {
    expect(email()("bad")).toBe("Please enter a valid email address");
  });

  it("uses a custom message when supplied", () => {
    expect(email("Invalid email")("bad")).toBe("Invalid email");
  });
});
