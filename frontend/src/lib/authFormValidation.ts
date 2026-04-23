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
  const signupNameInvalid = false;
  let signupEmailInvalid = false;
  let signupPasswordInvalid = false;

  if (input.isLogin) {
    const loginEmailMissing = required()(input.email) !== null;
    const loginPasswordMissing = required()(input.password) !== null;
    if (loginEmailMissing || loginPasswordMissing) {
      return {
        error: getLoginRequiredFieldError(
          loginEmailMissing,
          loginPasswordMissing,
        ),
        loginEmailInvalid: loginEmailMissing,
        loginPasswordInvalid: loginPasswordMissing,
        signupNameInvalid,
        signupEmailInvalid,
        signupPasswordInvalid,
      };
    }
  } else {
    const signupNameMissing = required()(input.name) !== null;
    const signupEmailMissing = required()(input.email) !== null;
    const signupPasswordMissing = required()(input.password) !== null;
    if (signupNameMissing || signupEmailMissing || signupPasswordMissing) {
      return {
        error: getSignupRequiredFieldError(
          signupNameMissing,
          signupEmailMissing,
          signupPasswordMissing,
        ),
        loginEmailInvalid,
        loginPasswordInvalid,
        signupNameInvalid: signupNameMissing,
        signupEmailInvalid: signupEmailMissing,
        signupPasswordInvalid: signupPasswordMissing,
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
