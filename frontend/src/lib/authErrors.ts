export function getLoginFieldInvalidState(message: string): {
  email: boolean;
  password: boolean;
} {
  const loweredError = message.toLowerCase();

  const ambiguous = [
    /\bemail\s+or\s+password\b/,
    /\bemail\s+and\s+password\b/,
  ].some((pattern) => pattern.test(loweredError));
  if (ambiguous) {
    return { email: false, password: false };
  }

  const mentionsEmail = [
    /\binvalid email\b/,
    /\bemail is invalid\b/,
    /\bemail is not valid\b/,
    /\bunknown account\b/,
    /\baccount not found\b/,
    /\buser not found\b/,
  ].some((pattern) => pattern.test(loweredError));
  const mentionsPassword = [
    /\bpassword must\b/,
    /\binvalid password\b/,
    /\bincorrect password\b/,
    /\bwrong password\b/,
  ].some((pattern) => pattern.test(loweredError));
  const mentionsCredentials = [
    /\binvalid credentials\b/,
    /\bincorrect credentials\b/,
    /\bwrong credentials\b/,
  ].some((pattern) => pattern.test(loweredError));

  if (mentionsEmail && mentionsPassword) {
    return { email: true, password: true };
  }
  if (mentionsEmail) {
    return { email: true, password: false };
  }
  if (mentionsPassword) {
    return { email: false, password: true };
  }
  if (mentionsCredentials) {
    return { email: true, password: true };
  }

  return { email: false, password: false };
}

export function getSignupFieldInvalidState(message: string): {
  name: boolean;
  email: boolean;
  password: boolean;
} {
  const loweredError = message.toLowerCase();

  const mentionsName = [
    /\bname is required\b/,
    /\binvalid name\b/,
    /\bname must\b/,
    /\bdisplay name\b/,
    /\bfull name\b/,
  ].some((pattern) => pattern.test(loweredError));
  const mentionsEmail = [
    /\binvalid email\b/,
    /\bemail is invalid\b/,
    /\bemail is not valid\b/,
    /\bplease enter a valid email\b/,
    /\binvalid email address\b/,
    /\bemail already exists\b/,
    /\bemail .* already exists\b/,
    /\bemail already registered\b/,
    /\bemail .* already registered\b/,
    /\bemail already taken\b/,
    /\bemail .* already taken\b/,
    /\bemail is already in use\b/,
    /\bemail .* is already in use\b/,
  ].some((pattern) => pattern.test(loweredError));
  const mentionsPassword = [
    /\bpassword must\b/,
    /\binvalid password\b/,
    /\bpassword is invalid\b/,
    /\bpassphrase\b/,
  ].some((pattern) => pattern.test(loweredError));

  if (!mentionsName && !mentionsEmail && !mentionsPassword) {
    return { name: false, email: false, password: false };
  }

  return {
    name: mentionsName,
    email: mentionsEmail,
    password: mentionsPassword,
  };
}
