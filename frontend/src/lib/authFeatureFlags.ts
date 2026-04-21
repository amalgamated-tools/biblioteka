import { getOidcEnabled, getPasskeyEnabled, getSignupEnabled } from "./api";

export interface AuthFeatureFlags {
  oidcEnabled: boolean;
  signupEnabled: boolean;
  passkeyEnabled: boolean;
  initError: string | null;
}

export async function fetchAuthFeatureFlags(
  signal: AbortSignal,
): Promise<AuthFeatureFlags> {
  const [oidcResult, signupResult, passkeyResult] = await Promise.allSettled([
    getOidcEnabled(signal),
    getSignupEnabled(signal),
    getPasskeyEnabled(signal),
  ]);

  return {
    oidcEnabled: oidcResult.status === "fulfilled" ? oidcResult.value : false,
    signupEnabled:
      signupResult.status === "fulfilled" ? signupResult.value : true,
    passkeyEnabled:
      passkeyResult.status === "fulfilled" ? passkeyResult.value : false,
    initError:
      oidcResult.status === "rejected" || signupResult.status === "rejected"
        ? "Unable to reach the server to load auth settings"
        : null,
  };
}
