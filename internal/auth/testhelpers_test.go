package auth

// tokenCookieName mirrors TokenCookieName() as a constant for tests that need
// to set cookies directly (httptest.Cookie.Name).
const tokenCookieName = "biblioteka_token"

// testConfig returns a Config suitable for middleware tests that verify both
// JWT + API key flows (APIKeyPrefix is populated).
func testConfig() Config {
	return Config{CookieName: tokenCookieName, APIKeyPrefix: APIKeyPrefix}
}

// testJWTOnlyConfig returns a Config suitable for middleware tests that only
// verify JWT flows (APIKeyPrefix is empty so API key handling is disabled).
func testJWTOnlyConfig() Config {
	return Config{CookieName: tokenCookieName}
}
