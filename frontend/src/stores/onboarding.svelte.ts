const STORAGE_KEY_PREFIX = "biblioteka_onboarding_skipped_";

class OnboardingStore {
  #skippedForUser = $state<string | undefined>(undefined);

  /**
   * Returns true if the first-library setup wizard has been dismissed for
   * the given user.  Returns false when `userId` is undefined (e.g. while
   * the auth store is still initialising).
   */
  isSkipped(userId: string | undefined): boolean {
    if (!userId) return false;
    // Depend on reactive state so callers re-evaluate when skip changes.
    void this.#skippedForUser;
    try {
      return localStorage.getItem(STORAGE_KEY_PREFIX + userId) === "1";
    } catch {
      return false;
    }
  }

  /** Persist the "skip" flag for this user so the wizard is not shown again. */
  skip(userId: string | undefined): void {
    if (!userId) return;
    try {
      localStorage.setItem(STORAGE_KEY_PREFIX + userId, "1");
      this.#skippedForUser = userId;
    } catch {
      // Ignore storage errors (e.g. private-browsing quotas).
    }
  }

  /**
   * Remove the "skip" flag, e.g. after the user creates their first library.
   * This keeps localStorage tidy and allows the wizard to resurface if the
   * user later removes all their libraries.
   */
  clearSkip(userId: string | undefined): void {
    if (!userId) return;
    try {
      localStorage.removeItem(STORAGE_KEY_PREFIX + userId);
      this.#skippedForUser = undefined;
    } catch {
      // Ignore storage errors.
    }
  }
}

export const onboardingStore = new OnboardingStore();
