import { AUTH_ERROR_TEST_ID, getAuthErrorBanner } from './shared.mjs';

/**
 * Verifies that getAuthErrorBanner queries the expected stable auth error test ID
 * and returns the locator from the underlying page implementation.
 *
 * Throws an Error if the behaviour does not match the expectation.
 */
export function verifyAuthErrorBanner() {
    let receivedTestId;
    const locator = { marker: 'locator' };
    const page = {
        getByTestId(testId) {
            receivedTestId = testId;
            return locator;
        },
    };

    const result = getAuthErrorBanner(page);

    if (receivedTestId !== AUTH_ERROR_TEST_ID) {
        throw new Error(
            `Expected getAuthErrorBanner to query test id "${AUTH_ERROR_TEST_ID}", but got "${receivedTestId}".`,
        );
    }

    if (result !== locator) {
        throw new Error(
            'Expected getAuthErrorBanner to return the locator from page.getByTestId().',
        );
    }
}

// Allow this module to be used as a standalone self-check script.
if (import.meta.url === `file://${process.argv[1]}`) {
    try {
        verifyAuthErrorBanner();
    } catch (err) {
        // eslint-disable-next-line no-console
        console.error(err);
        process.exitCode = 1;
    }
}
