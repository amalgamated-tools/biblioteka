/**
 * Lightweight form validation utility.
 *
 * Rules are functions that accept a string value and return an error message
 * string when the value is invalid, or null when it passes.
 *
 * @example
 * const error = validate(password, [
 *   required("Password is required"),
 *   minLength(6, "Password must be at least 6 characters"),
 * ]);
 */

/** A function that validates a string value and returns an error or null. */
export type ValidationRule = (value: string) => string | null;

/**
 * Returns a rule that fails when the trimmed value is empty.
 * @param message - Error message (default: "This field is required")
 */
export function required(message = "This field is required"): ValidationRule {
  return (value) => (value.trim() === "" ? message : null);
}

/**
 * Returns a rule that fails when the value is shorter than `min` characters.
 * @param min - Minimum length
 * @param message - Error message (default: "Must be at least N characters")
 */
export function minLength(
  min: number,
  message = `Must be at least ${min} characters`,
): ValidationRule {
  return (value) => (value.length < min ? message : null);
}

/**
 * Returns a rule that fails when the value does not equal `other`.
 * @param other - The string the value must match
 * @param message - Error message (default: "Values do not match")
 */
export function matches(
  other: string,
  message = "Values do not match",
): ValidationRule {
  return (value) => (value !== other ? message : null);
}

/**
 * Runs each rule in order and returns the first error message, or null if all
 * rules pass.
 * @param value - The string to validate
 * @param rules - Ordered list of validation rules
 */
export function validate(
  value: string,
  rules: ValidationRule[],
): string | null {
  for (const rule of rules) {
    const error = rule(value);
    if (error !== null) return error;
  }
  return null;
}
