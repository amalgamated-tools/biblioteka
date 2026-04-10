// Package handlers contains all HTTP handler structs and their request/response
// processing logic. Each domain (books, authors, series, libraries, users,
// Kobo, OPDS, KOSync, OIDC, admin, config) has a dedicated handler struct that
// holds the database connection and other dependencies. Shared utilities for
// JSON responses, pagination, audit logging, and generic CRUD patterns live in
// helpers.go and the named_entity.go / book_subresource.go / credentials.go
// / tokens.go files.
package handlers
