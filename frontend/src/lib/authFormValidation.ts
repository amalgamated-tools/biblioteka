import {
  email as emailRule,
  minLength,
  required,
  validate,
} from "./validation";
import {
  getLoginRequiredFieldError,
  getSignupRequiredFieldError,
} from "./authRequiredErrors";

export interface AuthFormValidationInput {
  isLogin: boolean;
  name: string;
  email: string;
  password: string;
}

export interface AuthFormValidationResult {
  error: string | null;
  loginEmailInvalid: boolean;
  loginPasswordInvalid: boolean;
  signupNameInvalid: boolean;
  signupEmailInvalid: boolean;
  signupPasswordInvalid: boolean;
}

export function validateAuthForm(
  input: AuthFormValidationInput,
): AuthFormValidationResult {
  let loginEmailInvalid = false;
  let loginPasswordInvalid = false;
  let signupNameInvalid = false;
  let signupEmailInvalid = false;
  let signupPasswordInvalid = false;

  if (input.isLogin) {
    loginEmailInvalid = required()(input.email) !== null;
    loginPasswordInvalid = required()(input.password) !== null;
    if (loginEmailInvalid || loginPasswordInvalid) {
      return {
        error: getLoginRequiredFieldError(
          loginEmailInvalid,
          loginPasswordInvalid,
        ),
        loginEmailInvalid,
        loginPasswordInvalid,
        signupNameInvalid,
        signupEmailInvalid,
        signupPasswordInvalid,
      };
    }
  } else {
    signupNameInvalid = required()(input.name) !== null;
    signupEmailInvalid = required()(input.email) !== null;
    signupPasswordInvalid = required()(input.password) !== null;
    if (signupNameInvalid || signupEmailInvalid || signupPasswordInvalid) {
      return {
        error: getSignupRequiredFieldError(
          signupNameInvalid,
          signupEmailInvalid,
          signupPasswordInvalid,
        ),
        loginEmailInvalid,
        loginPasswordInvalid,
        signupNameInvalid,
        signupEmailInvalid,
        signupPasswordInvalid,
      };
    }
  }

  const pwdError = validate(input.password, [
    minLength(6, "Password must be at least 6 characters"),
  ]);
  if (pwdError) {
    if (input.isLogin) {
      loginPasswordInvalid = true;
    } else {
      signupPasswordInvalid = true;
    }
    return {
      error: pwdError,
      loginEmailInvalid,
      loginPasswordInvalid,
      signupNameInvalid,
      signupEmailInvalid,
      signupPasswordInvalid,
    };
  }

  const invalidEmailError = validate(input.email, [
    emailRule("Please enter a valid email address"),
  ]);
  if (invalidEmailError) {
    if (input.isLogin) {
      loginEmailInvalid = true;
    } else {
      signupEmailInvalid = true;
    }
    return {
      error: invalidEmailError,
      loginEmailInvalid,
      loginPasswordInvalid,
      signupNameInvalid,
      signupEmailInvalid,
      signupPasswordInvalid,
    };
  }

  return {
    error: null,
    loginEmailInvalid,
    loginPasswordInvalid,
    signupNameInvalid,
    signupEmailInvalid,
    signupPasswordInvalid,
  };
}
