export function getLoginRequiredFieldError(
  emailMissing: boolean,
  passwordMissing: boolean,
): string {
  if (emailMissing && passwordMissing) {
    return "Please fill in all fields";
  }
  if (emailMissing) {
    return "Please fill in the email field";
  }
  return "Please fill in the password field";
}

export function getSignupRequiredFieldError(
  nameMissing: boolean,
  emailMissing: boolean,
  passwordMissing: boolean,
): string {
  if (nameMissing && emailMissing && passwordMissing) {
    return "Please fill in all fields";
  }
  if (nameMissing && emailMissing) {
    return "Please fill in the name and email fields";
  }
  if (nameMissing && passwordMissing) {
    return "Please fill in the name and password fields";
  }
  if (emailMissing && passwordMissing) {
    return "Please fill in the email and password fields";
  }
  if (nameMissing) {
    return "Please fill in the name field";
  }
  if (emailMissing) {
    return "Please fill in the email field";
  }
  return "Please fill in the password field";
}
