import test from 'node:test';
import assert from 'node:assert/strict';
import { AUTH_ERROR_TEST_ID, getAuthErrorBanner } from './shared.mjs';

test('auth screenshot flow watches the stable auth error test id', () => {
    let receivedTestId;
    const locator = { marker: 'locator' };
    const page = {
        getByTestId(testId) {
            receivedTestId = testId;
            return locator;
        },
    };

    const result = getAuthErrorBanner(page);

    assert.equal(receivedTestId, AUTH_ERROR_TEST_ID);
    assert.equal(result, locator);
});
