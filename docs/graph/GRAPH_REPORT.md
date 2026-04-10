# Graph Report (2026-04-09)

## Corpus Check
- Large corpus: 461 files · ~2,500,881 words. Semantic extraction will be expensive (many Claude tokens). Consider running on a subfolder, or use --no-semantic to run AST-only.

## Summary
- 2890 nodes · 3697 edges · 194 communities detected
- Extraction: 99% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS · INFERRED: 16 edges (avg confidence: 0.84)
- Token cost: 0 input · 0 output

## God Nodes (most connected - your core abstractions)
1. `DB` - 95 edges
2. `newAuthHandler()` - 28 edges
3. `setupBookHandler()` - 24 edges
4. `makeTestNamedEntityOps()` - 23 edges
5. `setupKOSyncHandler()` - 23 edges
6. `koboDeviceHandler()` - 21 edges
7. `createTestKoboToken()` - 20 edges
8. `requireExifTool()` - 18 edges
9. `OPDSHandler` - 17 edges
10. `BookHandler` - 16 edges

## Surprising Connections (you probably didn't know these)
- `Agent Instructions (AGENTS.md)` --semantically_similar_to--> `Claude Instructions (CLAUDE.md)`  [INFERRED] [semantically similar]
  AGENTS.md → CLAUDE.md
- `Agent Instructions (AGENTS.md)` --semantically_similar_to--> `Gemini Instructions (GEMINI.md)`  [INFERRED] [semantically similar]
  AGENTS.md → GEMINI.md
- `Claude Instructions (CLAUDE.md)` --semantically_similar_to--> `Gemini Instructions (GEMINI.md)`  [INFERRED] [semantically similar]
  CLAUDE.md → GEMINI.md
- `File Organization (book_per_folder/book_per_file/none)` --conceptually_related_to--> `process:file Job`  [INFERRED]
  README.md → docs/background-jobs.md
- `Biblioteka Project` --references--> `Background Jobs (asynq/Redis)`  [EXTRACTED]
  README.md → docs/background-jobs.md

## Hyperedges (group relationships)
- **Authentication Methods** — auth_jwt, auth_oidc, auth_api_keys, opds_credentials, kosync_credentials, kobo_tokens, auth_rate_limiting [EXTRACTED 0.90]
- **Book Import Pipeline** — job_scan_libraries, job_scan_library, job_scan_path, job_process_file, metadata_extraction, path_based_metadata, metadata_sidecar, file_organization [EXTRACTED 0.95]
- **E-Reader Sync Protocols** — opds_catalog, kobo_sync, koreader_sync [EXTRACTED 0.90]
- **Settings Tabbed Architecture** — screenshots/settings-light.png, screenshots/settings-api-keys-dark.png, screenshots/settings-preferences-dark.png, screenshots/settings-users-light.png, screenshots/settings-oidc-light-mobile.png, screenshots/settings-nonadmin-dark.png, screenshots/settings-nonadmin-api-keys-light.png, screenshots/settings-nonadmin-preferences-light.png, screenshots/settings-nonadmin-kobo-light-mobile.png [EXTRACTED 0.95]
- **Mobile Responsive Pattern** — screenshots/my-library-dark-mobile.png, screenshots/my-library-light-mobile.png, screenshots/libraries-light-mobile.png, screenshots/settings-light-mobile.png, screenshots/settings-api-keys-light-mobile.png, screenshots/settings-nonadmin-kobo-light-mobile.png, screenshots/settings-oidc-light-mobile.png [INFERRED 0.85]
- **Authentication Flow** — screenshots/login-dark.png, screenshots/login-light.png, screenshots/signup-dark.png [EXTRACTED 0.95]
- **Dark Theme Mobile Settings Screens** — screenshot:settings-nonadmin-dark-mobile, screenshot:settings-users-dark-mobile, screenshot:settings-api-keys-dark-mobile, screenshot:settings-nonadmin-api-keys-dark-mobile, screenshot:settings-kobo-dark-mobile, screenshot:settings-smtp-dark-mobile, screenshot:settings-nonadmin-preferences-dark-mobile [EXTRACTED 0.75]
- **Admin-Only Feature Screens** — screenshot:settings-users-dark-mobile, screenshot:settings-users-light-mobile, screenshot:settings-oidc-light, screenshot:settings-smtp-dark-mobile, feature:oidc, feature:smtp, concept:rbac [EXTRACTED 0.75]
- **Unauthenticated Auth Pages** — screenshot:login-light-mobile, screenshot:signup-dark-mobile [EXTRACTED 0.75]
- **Responsive Mobile UI Screens** — books-dark, books-light-mobile, books-dark-mobile, settings-smtp-dark, settings-smtp-light-mobile, settings-oidc-dark, settings-oidc-dark-mobile, settings-dark-mobile, settings-preferences-dark-mobile, settings-preferences-light-mobile, settings-nonadmin-kobo-light, settings-nonadmin-kobo-dark-mobile, settings-nonadmin-api-keys-light-mobile, login-dark-mobile, dashboard-dark-mobile, responsive-design [INFERRED 0.75]
- **Admin and Non-Admin Settings Navigation** — settings-smtp-dark, settings-oidc-dark, settings-api-keys-light, settings-dark-mobile, settings-nonadmin-light, settings-nonadmin-preferences-dark, settings-nonadmin-kobo-light, settings-nonadmin-kobo-dark-mobile, settings-nonadmin-api-keys-light-mobile, admin-settings, non-admin-settings, settings-sidebar [INFERRED 0.75]
- **Dark and Light Theme Screen Variants** — books-dark, books-light-mobile, books-dark-mobile, settings-smtp-dark, settings-smtp-light-mobile, settings-preferences-dark-mobile, settings-preferences-light-mobile, settings-nonadmin-kobo-light, settings-nonadmin-kobo-dark-mobile, settings-nonadmin-api-keys-light-mobile, settings-api-keys-light, settings-nonadmin-light, settings-nonadmin-preferences-dark, signup-light, login-dark-mobile, dark-theme, light-theme [INFERRED 0.75]
- **Dark Light Theme System** — screenshot__dashboard_dark, screenshot__dashboard_light_mobile, screenshot__settings_dark, screenshot__settings_kobo_dark, screenshot__settings_smtp_light, screenshot__my_library_dark, screenshot__signup_light_mobile [EXTRACTED 0.90]
- **Brand Icon Family** — icon__android_chrome_512, icon__android_chrome_192, icon__apple_touch, icon__favicon_32, icon__favicon_16 [EXTRACTED 0.95]
- **Settings Tabbed Architecture** — screenshot__settings_dark, screenshot__settings_kobo_dark, screenshot__settings_smtp_light, screenshot__settings_users_dark, screenshot__settings_nonadmin_kobo_dark [EXTRACTED 0.90]

## Communities

### Community 0 - "Admin & Auth Handlers"
Cohesion: 0.01
Nodes (45): getFeatureEnabled(), getOidcEnabled(), getSignupEnabled(), openAuthPage(), openLoginForm(), openSignupForm(), signUp(), putBookSubResource() (+37 more)

### Community 1 - "Book Metadata Pipeline"
Cohesion: 0.02
Nodes (55): checkDuplicate(), linkExistingBookAndSkip(), reorganizedCandidatePaths(), resolveSourcePath(), validateField(), validatePayload(), isBookFilePathAllowed(), isPathUnderRoot() (+47 more)

### Community 2 - "Kobo API Generated Code"
Cohesion: 0.01
Nodes (45): __marshalSearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchResultEdge(), __unmarshalSearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchResultEdge(), GetBookByAsinGetBookByAsinBook, GetBookByAsinGetBookByAsinBookWork, GetBookByAsinGetBookByAsinBookWorkBestBook, GetBookByAsinGetBookByAsinBookWorkBestBookDetails, GetBookByAsinGetBookByAsinBookWorkBestBookDetailsLanguage, GetBookByAsinGetBookByAsinBookWorkBestBookPrimaryContributorEdgeBookContributorEdge (+37 more)

### Community 3 - "DB Models & CRUD"
Cohesion: 0.02
Nodes (29): scanAPIKey(), scanAuditLog(), toAuditLogDTO(), scanAuthor(), bookFileColumnsWithPrefix(), scanBookFile(), bookColumnsWithPrefix(), dollarN() (+21 more)

### Community 4 - "Metadata Extraction Tests"
Cohesion: 0.02
Nodes (42): requireExifToolExtractor(), TestEPUB3CoverImport_EndToEnd(), fetchMetadataResponse, metadataDTO, MetadataHandler, ProcessFilePayload, ActivePeriod, Bookmark (+34 more)

### Community 5 - "Frontend & Linting"
Cohesion: 0.03
Nodes (22): isSlogAny(), run(), AuthStore, AuthorStore, CrudStore, ExifToolOutput, GuideReference, Identifier (+14 more)

### Community 6 - "OPDS Feed Protocol"
Cohesion: 0.03
Nodes (7): writeOPDSError(), writeOPDSFeed(), padInt(), TestAllBooks_Pagination(), setupOPDSHandler(), TestHandleOPDS_MethodNotAllowed(), TestHandleOPDS_UnknownPath()

### Community 7 - "Author Entity CRUD"
Cohesion: 0.04
Nodes (15): NormalizeAuthorName(), normalizeName(), ApiError, getToken(), request(), Author, authorListQuery, Series (+7 more)

### Community 8 - "Goodreads Enrichment"
Cohesion: 0.04
Nodes (35): createGoodreadsMetadataFromResult(), enrichGoodreads(), lookupGoodreads(), NewEnrichGoodreadsHandler(), publishEvent(), publishProgress(), titleSimilar(), EnrichGoodreadsPayload (+27 more)

### Community 9 - "Documentation Hub"
Cohesion: 0.06
Nodes (44): Audit Logging, Administration Guide, REST API Overview, API Reference, API Keys (bib_ prefix), JWT Authentication, OIDC/SSO Authentication, OIDC Account Linking (+36 more)

### Community 10 - "Book Handler Tests"
Cohesion: 0.07
Nodes (24): setupBookHandler(), TestBookAuthors_Handler(), TestBookFiles_Handler(), TestBookSeries_Handler(), TestCreateBook_EnqueueFailureDoesNotFailRequest(), TestCreateBook_EnqueuesGoodreadsJob(), TestCreateBook_Handler(), TestCreateBook_MissingTitle() (+16 more)

### Community 11 - "KOSync Protocol Tests"
Cohesion: 0.1
Nodes (32): createTestUserForKOSync(), setupKOSyncHandler(), TestKOSyncCredential_Delete(), TestKOSyncCredential_Upsert_UpdatesExisting(), TestKOSyncCredential_UpsertAndGet(), TestKOSyncCredential_UsernameConflict(), TestKOSyncCredentials_Delete(), TestKOSyncCredentials_Delete_NotFound() (+24 more)

### Community 12 - "Library Management"
Cohesion: 0.08
Nodes (13): Library, libraryListQuery, libraryDTO, LibraryHandler, libraryRequest, pathValidationError, isColumnUniqueViolation(), isUniqueViolation() (+5 more)

### Community 13 - "Metadata Handler Tests"
Cohesion: 0.09
Nodes (20): mockSubscriber, createTestBook(), MockEventSource, setupMetadataHandler(), strPtr(), TestApplyMetadata(), TestApplyMetadata_NoPending(), TestBookMetadata_Language_Set() (+12 more)

### Community 14 - "Series Handler Tests"
Cohesion: 0.08
Nodes (14): setupSeriesHandler(), TestCreateSeries_Duplicate(), TestCreateSeries_Handler(), TestCreateSeries_MissingName(), TestCreateSeries_WhitespaceOnlyName(), TestDeleteSeries_Handler(), TestDeleteSeries_NotFound(), TestGetSeries_Handler() (+6 more)

### Community 15 - "Author Handler Tests"
Cohesion: 0.08
Nodes (14): setupAuthorHandler(), TestCreateAuthor_Duplicate(), TestCreateAuthor_Handler(), TestCreateAuthor_MissingName(), TestCreateAuthor_WhitespaceOnlyName(), TestDeleteAuthor_Handler(), TestDeleteAuthor_NotFound(), TestGetAuthor_Handler() (+6 more)

### Community 16 - "OIDC Config Tests"
Cohesion: 0.06
Nodes (0): 

### Community 17 - "HTTP Middleware"
Cohesion: 0.09
Nodes (19): flusherRecorder, plainResponseWriter, assertJSONError(), TestLoggingMiddleware_CallsNext(), TestLoggingMiddleware_ImplicitOKOnWrite(), TestLoggingMiddleware_PreservesStatus(), TestMiddleware_InvalidAuthorizationFormat(), TestMiddleware_InvalidToken() (+11 more)

### Community 18 - "Auth Flow Tests"
Cohesion: 0.14
Nodes (30): assertAuthCookie(), mustSignup(), newAuthHandler(), TestChangePassword_Invalid(), TestChangePassword_MethodNotAllowed(), TestChangePassword_Success(), TestLogin_MethodNotAllowed(), TestLogin_MissingFields() (+22 more)

### Community 19 - "ExifTool Extractor"
Cohesion: 0.11
Nodes (19): normalizeExifDate(), requireExifTool(), TestExtractMetadata_EPUB(), TestExtractMetadata_EPUB2And3ProduceSameMetadata(), TestExtractMetadata_EPUB2CoverViaMeta(), TestExtractMetadata_EPUB2Metadata(), TestExtractMetadata_EPUB3CoverViaProperties(), TestExtractMetadata_EPUB3Metadata() (+11 more)

### Community 20 - "Auth (30 nodes)"
Cohesion: 0.12
Nodes (23): Claims, JWTManager, testOIDCProvider, callbackRequest(), newTestOIDCHandler(), newTestOIDCProvider(), requireOIDCCookies(), seedLinkNonce() (+15 more)

### Community 21 - "Tsv (29 nodes)"
Cohesion: 0.07
Nodes (0): 

### Community 22 - "Config (29 nodes)"
Cohesion: 0.09
Nodes (11): IsLoopbackHost(), isValidSMTPHostForStatus(), ValidateForSend(), ValidateHost(), validationErr(), ConfigHandler, configStatusResponse, Config (+3 more)

### Community 23 - "Config (29 nodes)"
Cohesion: 0.07
Nodes (0): 

### Community 24 - "Handlers (28 nodes)"
Cohesion: 0.13
Nodes (26): testEntity, testEntityDTO, testEntityRequest, makeTestNamedEntityOps(), TestCreateNamedEntity_EmptyName(), TestCreateNamedEntity_ErrInvalidName(), TestCreateNamedEntity_ErrNameExists(), TestCreateNamedEntity_GenericCreateError() (+18 more)

### Community 25 - "Goodreads (27 nodes)"
Cohesion: 0.1
Nodes (11): mockGraphQLClient, mockHTTPClient, noResponseClient(), TestParseISBNSearchResponse_CancelledContext(), TestParseISBNSearchResponse_EmptyArray(), TestParseISBNSearchResponse_InvalidBookID(), TestParseISBNSearchResponse_InvalidJSON(), TestParseISBNSearchResponse_InvalidWorkID() (+3 more)

### Community 26 - "Api (26 nodes)"
Cohesion: 0.14
Nodes (21): createTestUser(), setupAPIKeyHandler(), TestCreateAPIKey(), TestCreateAPIKey_AuditLog(), TestCreateAPIKey_EmptyName(), TestCreateAPIKey_NameTooLong(), TestCreateAPIKey_Success(), TestDeleteAPIKey() (+13 more)

### Community 27 - "Enrich (25 nodes)"
Cohesion: 0.16
Nodes (15): createTestBookWithFields(), createTestUser(), TestEnrichGoodreads_ASINLookup(), TestEnrichGoodreads_GoodreadsIDLookup(), TestEnrichGoodreads_ISBN10Lookup(), TestEnrichGoodreads_ISBNFailsFallsToTitle(), TestEnrichGoodreads_ISBNLookup(), TestEnrichGoodreads_ISBNPreferredOverTitle() (+7 more)

### Community 28 - "Handlers (25 nodes)"
Cohesion: 0.21
Nodes (23): testKoboTokenChecker, createTestKoboToken(), koboDeviceHandler(), TestHandleKobo_AnalyticsGetTests(), TestHandleKobo_Auth_Stub(), TestHandleKobo_BookMetadata_NonGET(), TestHandleKobo_BookMetadata_NotFound(), TestHandleKobo_BookMetadata_Success() (+15 more)

### Community 29 - "Finish (24 nodes)"
Cohesion: 0.08
Nodes (0): 

### Community 30 - "Ratelimit (22 nodes)"
Cohesion: 0.14
Nodes (13): mustParseCIDR(), TestIpFromRequestTrusted_AllTrusted(), TestIpFromRequestTrusted_IPv6(), TestIpFromRequestTrusted_IPv6_AllTrusted(), TestIpFromRequestTrusted_MultipleProxies(), TestIpFromRequestTrusted_NoXFF(), TestIpFromRequestTrusted_PortSuffixedXFF(), TestIpFromRequestTrusted_RemoteNotTrusted() (+5 more)

### Community 31 - "Server (22 nodes)"
Cohesion: 0.1
Nodes (4): newTestDB(), TestNewServer_DefaultPort(), TestNewServer_WithDB(), TestNewServer_WithPort()

### Community 32 - "Config (22 nodes)"
Cohesion: 0.11
Nodes (5): setupConfigHandler(), TestHandleConfigStatus_MethodNotAllowed(), TestHandleConfigStatus_RegularUser(), TestHandleConfigStatus_Success(), TestHandleConfigStatus_WhenConfigured()

### Community 33 - "Autodismisstimer (21 nodes)"
Cohesion: 0.1
Nodes (4): AutoDismissTimer, CopyTimeoutState, TimeoutState, TestTimeoutState

### Community 34 - "Db (21 nodes)"
Cohesion: 0.15
Nodes (15): fakeRow, sample, memDB(), scanSampleAndTotal(), TestCollectRows_AlwaysClosesRows(), TestCollectRows_ClosesRowsOnError(), TestCollectRows_EmptyResult(), TestCollectRows_HappyPath() (+7 more)

### Community 35 - "Users (20 nodes)"
Cohesion: 0.1
Nodes (0): 

### Community 36 - "Audit (20 nodes)"
Cohesion: 0.16
Nodes (12): setupAuditLogHandler(), TestHandleAuditLogs_AdminSuccess(), TestHandleAuditLogs_CustomPagination(), TestHandleAuditLogs_DefaultPagination(), TestHandleAuditLogs_EmptyList(), TestHandleAuditLogs_InvalidLimit(), TestHandleAuditLogs_InvalidOffset(), TestHandleAuditLogs_LimitCappedAtMax() (+4 more)

### Community 37 - "Auth (19 nodes)"
Cohesion: 0.16
Nodes (14): adminCacheEntry, AdminChecker, APIKeyValidator, cachingAdminChecker, contextKey, tokenSource, AdminMiddleware(), extractToken() (+6 more)

### Community 38 - "Cover (19 nodes)"
Cohesion: 0.13
Nodes (6): addZipEntry(), makeEPUBWithCover(), TestExtractEPUBCoverDataURL_EPUB3Cover(), TestExtractEPUBCoverDataURL_MissingArchiveFile(), TestParseTSV_EPUB2CoverExtractionE2E(), TestParseTSV_EPUB3CoverExtractionE2E()

### Community 39 - "Oidc (19 nodes)"
Cohesion: 0.11
Nodes (2): newTestOIDCHandlerWithTokenResponse(), TestOIDCCallback_MissingIDToken()

### Community 40 - "Handlers (19 nodes)"
Cohesion: 0.13
Nodes (3): KoboHandler, koboRandomUUID(), writeKoboJSON()

### Community 41 - "Book (18 nodes)"
Cohesion: 0.14
Nodes (7): loadBookResult(), autocompleteEntry, Client, HttpClient, workData, buildFallbackResult(), parseAutocompleteEntries()

### Community 42 - "Kobo (18 nodes)"
Cohesion: 0.21
Nodes (14): createTestBookForKobo(), koboTestFutureTime(), koboTestPastTime(), TestGetKoboReadingState_NotFound(), TestGetKoboReadingState_UserIsolation(), TestGetReadingStatesForBooks_ReturnsMap(), TestGetReadingStatesForBooks_UserIsolation(), TestGetReadingStatesForBooks_WithSinceFilter() (+6 more)

### Community 43 - "Routes (17 nodes)"
Cohesion: 0.19
Nodes (7): checkSystemEndpointMethod(), swaggerSecurityHeaders(), writeSystemJSON(), enabledResponse, healthResponse, Server, versionResponse

### Community 44 - "Db (17 nodes)"
Cohesion: 0.12
Nodes (1): invalidListQuery

### Community 45 - "Worker (17 nodes)"
Cohesion: 0.2
Nodes (12): newTestWorker(), TestEnqueue_NonMarshalablePayload(), TestNew_ValidURL(), TestRedisConnOpt(), TestRegister(), TestRegister_HandlerError(), TestRegister_NilPayload(), TestRegisterSchedule() (+4 more)

### Community 46 - "Handlers (17 nodes)"
Cohesion: 0.19
Nodes (1): OPDSHandler

### Community 47 - "Goodreads (16 nodes)"
Cohesion: 0.18
Nodes (8): networkError, TestSearchByISBN_EmptyISBN(), TestSearchByISBN_HTTPFailure(), TestSearchByISBN_InvalidISBN10CheckDigit(), TestSearchByISBN_InvalidISBN13CheckDigit(), TestSearchByISBN_InvalidLength(), TestSearchByISBN_NonOKStatus(), TestSearchByISBN_ResponseTooLarge()

### Community 48 - "Handlers (16 nodes)"
Cohesion: 0.13
Nodes (1): BookHandler

### Community 49 - "Goodreads (16 nodes)"
Cohesion: 0.12
Nodes (0): 

### Community 50 - "Screenshots/Login-Dark.Png (16 nodes)"
Cohesion: 0.12
Nodes (16): Login Dark Theme, Login Light Theme, My Library Dark Mobile, My Library Light Mobile, My Library Light Desktop, Settings API Keys Dark, Settings API Keys Light Mobile, Settings Light Mobile (+8 more)

### Community 51 - "Timestamp (15 nodes)"
Cohesion: 0.13
Nodes (0): 

### Community 52 - "Admin (15 nodes)"
Cohesion: 0.25
Nodes (13): setupAdminHandler(), TestHandleListUsers_AdminSuccess(), TestHandleListUsers_MethodNotAllowed(), TestHandleListUsers_NonAdminForbidden(), TestHandleListUsers_ResponseContainsAdminFlag(), TestHandleSetAdmin_AdminDemotesUser(), TestHandleSetAdmin_AdminPromotesUser(), TestHandleSetAdmin_CannotChangeSelf() (+5 more)

### Community 53 - "Books (15 nodes)"
Cohesion: 0.18
Nodes (6): createTestLibrary(), registerTestLibrary(), TestPostBookFiles_AuditLog(), TestPostBookFiles_PathOutsideLibrary(), TestPostBookFiles_PathTraversal(), TestPostBookFiles_Success()

### Community 54 - "Exif (14 nodes)"
Cohesion: 0.2
Nodes (4): Exiftool, FileMetadata, handleWriteMetadataResponse(), toString()

### Community 55 - "Db (14 nodes)"
Cohesion: 0.2
Nodes (5): ReadingProgress, KOSyncHandler, kosyncProgressRequest, kosyncProgressResponse, toKOSyncProgressResponse()

### Community 56 - "Kobo (14 nodes)"
Cohesion: 0.24
Nodes (9): createTestKoboTokenID(), setupKoboHandler(), TestKoboTokenCollection_MethodNotAllowed(), TestKoboTokenCreate_EmptyName(), TestKoboTokenCreate_Success(), TestKoboTokenDelete_NotFound(), TestKoboTokenDelete_Success(), TestKoboTokenList_Empty() (+1 more)

### Community 57 - "Admin-Settings (14 nodes)"
Cohesion: 0.18
Nodes (14): Admin settings section, Authentication flow, Login page (dark, mobile), Non-admin settings section, OIDC/SSO configuration, Settings Account tab (dark, mobile), Non-admin settings (light, desktop), OIDC/SSO settings (dark, desktop) (+6 more)

### Community 58 - "Book (13 nodes)"
Cohesion: 0.15
Nodes (0): 

### Community 59 - "Jobs (13 nodes)"
Cohesion: 0.15
Nodes (1): errEnqueuer

### Community 60 - "Types (12 nodes)"
Cohesion: 0.17
Nodes (0): 

### Community 61 - "Auth (12 nodes)"
Cohesion: 0.27
Nodes (7): RateLimiter, visitor, ipFromRequest(), ipFromRequestTrusted(), isTrusted(), NewRateLimiter(), NewRateLimiterWithTrustedProxies()

### Community 62 - "Exif (12 nodes)"
Cohesion: 0.17
Nodes (0): 

### Community 63 - "Book (12 nodes)"
Cohesion: 0.21
Nodes (4): setupBookFileHandler(), TestDeleteBookFile_Handler(), TestGetBookFile_Handler(), TestGetBookFile_NotFound()

### Community 64 - "Cover (12 nodes)"
Cohesion: 0.32
Nodes (11): archiveCandidates(), cleanArchivePath(), extensionForMIME(), extractEPUBCoverDataURL(), findArchiveFile(), findEPUBCoverRef(), readEPUBArchiveFile(), readEPUBRootFilePath() (+3 more)

### Community 65 - "Api (12 nodes)"
Cohesion: 0.23
Nodes (6): toAPIKeyDTO(), APIKey, apiKeyCreateRequest, apiKeyCreateResponse, apiKeyDTO, APIKeyHandler

### Community 66 - "Auth (11 nodes)"
Cohesion: 0.24
Nodes (5): mockOPDSChecker, newOPDSCheckerWithUser(), TestOPDSBasicAuth_Success(), TestOPDSBasicAuth_UsernameLowercased(), TestOPDSBasicAuth_WrongPassword()

### Community 67 - "Auth (11 nodes)"
Cohesion: 0.24
Nodes (5): mockKOSyncChecker, newKOSyncCheckerWithUser(), TestKOSyncHeaderAuth_Success(), TestKOSyncHeaderAuth_UsernameLowercased(), TestKOSyncHeaderAuth_WrongKey()

### Community 68 - "Tsv (11 nodes)"
Cohesion: 0.4
Nodes (10): flushIdent(), flushManifest(), flushMeta(), isLikelyImage(), parseIdentifierLine(), parseManifestLine(), parseMetaLine(), parseScalar() (+2 more)

### Community 69 - "Db (11 nodes)"
Cohesion: 0.18
Nodes (1): fakeEntity

### Community 70 - "Mobi (11 nodes)"
Cohesion: 0.25
Nodes (7): buildEXTH(), buildMOBI(), MakeTestAZW3(), MakeTestMOBI(), uint32ToBytes(), exthRecord, MOBIOptions

### Community 71 - "Screenshot (11 nodes)"
Cohesion: 0.2
Nodes (11): Dashboard Dark Theme, Dashboard Light Mobile, Libraries Dark Mobile, My Library Dark Empty State, Settings Account Dark, Settings Kobo Sync Dark, Settings Kobo Sync Light Mobile, Settings Non-Admin Kobo Dark (+3 more)

### Community 72 - "Jwt (10 nodes)"
Cohesion: 0.2
Nodes (0): 

### Community 73 - "Book (10 nodes)"
Cohesion: 0.2
Nodes (0): 

### Community 74 - "Book (10 nodes)"
Cohesion: 0.2
Nodes (0): 

### Community 75 - "Auth (9 nodes)"
Cohesion: 0.22
Nodes (1): mockAdminChecker

### Community 76 - "Named (9 nodes)"
Cohesion: 0.22
Nodes (0): 

### Community 77 - "Opf (9 nodes)"
Cohesion: 0.25
Nodes (7): marshalOPF(), WriteOPF(), OPFData, opfItem, opfManifest, opfMetadata, opfPackage

### Community 78 - "Jobs (9 nodes)"
Cohesion: 0.22
Nodes (2): genericEnqueuedJob, genericMockEnqueuer

### Community 79 - "Jobs (9 nodes)"
Cohesion: 0.22
Nodes (1): mockLibraryLister

### Community 80 - "Decode (9 nodes)"
Cohesion: 0.22
Nodes (0): 

### Community 81 - "Auth (9 nodes)"
Cohesion: 0.22
Nodes (6): authResponse, changePasswordRequest, loginRequest, signupRequest, updateProfileRequest, userDTO

### Community 82 - "Kobo (9 nodes)"
Cohesion: 0.22
Nodes (0): 

### Community 83 - "Epub (9 nodes)"
Cohesion: 0.33
Nodes (6): MakeTestEPUB(), MakeTestEPUBWithOptions(), writeZipFile(), writeZipFileBytes(), xmlEscape(), EPUBOptions

### Community 84 - "Concept:Rbac (9 nodes)"
Cohesion: 0.22
Nodes (9): Role-Based Access Control, OIDC Authentication, SMTP Email Configuration, Settings Non-Admin (Dark, Mobile), Settings Non-Admin (Light, Mobile), Settings OIDC (Light, Desktop), Settings SMTP (Dark, Mobile), Settings Users (Dark, Mobile) (+1 more)

### Community 85 - "Pagination (8 nodes)"
Cohesion: 0.25
Nodes (0): 

### Community 86 - "Auth (8 nodes)"
Cohesion: 0.25
Nodes (1): mockAPIKeyValidator

### Community 87 - "Sql (8 nodes)"
Cohesion: 0.25
Nodes (0): 

### Community 88 - "Logging (8 nodes)"
Cohesion: 0.25
Nodes (1): statusRecorder

### Community 89 - "Validation (7 nodes)"
Cohesion: 0.29
Nodes (0): 

### Community 90 - "Auth (7 nodes)"
Cohesion: 0.48
Nodes (5): TestAuthTransport_AddsKeyForAllowedHost(), TestAuthTransport_DoesNotMutateOriginalRequest(), TestAuthTransport_EmptyAllowedHostSendsKeyEverywhere(), TestAuthTransport_OmitsKeyForDifferentHost(), recordingTransport

### Community 91 - "Auth (7 nodes)"
Cohesion: 0.33
Nodes (4): KoboTokenChecker, KoboTokenResult, KoboTokenAuthMiddleware(), writeKoboJSONError()

### Community 92 - "Auth (7 nodes)"
Cohesion: 0.29
Nodes (1): mockKoboTokenChecker

### Community 93 - "Handlers (7 nodes)"
Cohesion: 0.38
Nodes (1): AuthHandler

### Community 94 - "Book (7 nodes)"
Cohesion: 0.29
Nodes (3): fakeItem, fakeItemDTO, fakeSetRequest

### Community 95 - "Book (6 nodes)"
Cohesion: 0.33
Nodes (0): 

### Community 96 - "Book (6 nodes)"
Cohesion: 0.4
Nodes (4): scanBookSeriesEntry(), BookSeriesEntry, BookSeriesInput, prefixedScanner

### Community 97 - "Sql (6 nodes)"
Cohesion: 0.67
Nodes (4): isSQLIdentifierRune(), removeInlineComments(), scanDollarTag(), splitStatements()

### Community 98 - "Handlers (6 nodes)"
Cohesion: 0.47
Nodes (5): errorResponse, decodeJSON(), writeError(), writeJSON(), writeSecretTokenResponse()

### Community 99 - "Auth (6 nodes)"
Cohesion: 0.6
Nodes (5): defaultPort(), matchRequestOrigin(), normalizeHost(), parseHostPort(), sameOrigin()

### Community 100 - "Kobo (6 nodes)"
Cohesion: 0.33
Nodes (0): 

### Community 101 - "Api-Keys (6 nodes)"
Cohesion: 0.47
Nodes (6): API Keys management, Kobo Sync feature, API Keys settings (light, desktop), Non-admin API Keys (light, mobile), Non-admin Kobo Sync (dark, mobile), Non-admin Kobo Sync (light, desktop)

### Community 102 - "P (5 nodes)"
Cohesion: 0.4
Nodes (2): errorValue, myStruct

### Community 103 - "Logger (5 nodes)"
Cohesion: 0.4
Nodes (1): Payload

### Community 104 - "Pathparser (5 nodes)"
Cohesion: 0.4
Nodes (0): 

### Community 105 - "Db (5 nodes)"
Cohesion: 0.4
Nodes (1): listQuery

### Community 106 - "Pubsub (5 nodes)"
Cohesion: 0.6
Nodes (3): redisTestURL(), TestPublishSubscribe(), TestSubscribeCancelStopsChannel()

### Community 107 - "Telemetry (5 nodes)"
Cohesion: 0.4
Nodes (0): 

### Community 108 - "Cover (5 nodes)"
Cohesion: 0.5
Nodes (2): coverMIMEType(), dataURLMIMEType()

### Community 109 - "Libraries (5 nodes)"
Cohesion: 0.4
Nodes (0): 

### Community 110 - "Cover (5 nodes)"
Cohesion: 0.4
Nodes (0): 

### Community 111 - "Helpers (5 nodes)"
Cohesion: 0.4
Nodes (0): 

### Community 112 - "Middleware (5 nodes)"
Cohesion: 0.5
Nodes (3): contextKey, RequestIDHandler(), WithRequestID()

### Community 113 - "Db (5 nodes)"
Cohesion: 0.4
Nodes (5): PostgreSQL Database, Database Schema, Shared Catalog Access Model, SQLite Database, dbmate Migrations

### Community 114 - "Icon (5 nodes)"
Cohesion: 0.4
Nodes (5): Android Chrome Icon 192, Android Chrome Icon 512, Apple Touch Icon, Favicon 16x16, Favicon 32x32

### Community 115 - "Book (4 nodes)"
Cohesion: 0.5
Nodes (0): 

### Community 116 - "Auth (4 nodes)"
Cohesion: 0.67
Nodes (3): OPDSCredentialChecker, OPDSBasicAuthMiddleware(), writeOPDSError()

### Community 117 - "Scan (4 nodes)"
Cohesion: 0.5
Nodes (0): 

### Community 118 - "Setup (4 nodes)"
Cohesion: 0.5
Nodes (0): 

### Community 119 - "Ptr (4 nodes)"
Cohesion: 0.5
Nodes (0): 

### Community 120 - "Helpers (4 nodes)"
Cohesion: 0.5
Nodes (0): 

### Community 121 - "Request (4 nodes)"
Cohesion: 0.5
Nodes (0): 

### Community 122 - "Helpers (4 nodes)"
Cohesion: 0.5
Nodes (0): 

### Community 123 - "Validate (4 nodes)"
Cohesion: 0.5
Nodes (0): 

### Community 124 - "Observability (4 nodes)"
Cohesion: 0.5
Nodes (4): Structured Logging (slog/JSON), OTel Key Constants, Request ID Correlation, Distributed Tracing (OpenTelemetry)

### Community 125 - "Agents (4 nodes)"
Cohesion: 0.67
Nodes (4): Agent Instructions (AGENTS.md), Claude Instructions (CLAUDE.md), Conventional Commits v1.0.0, Gemini Instructions (GEMINI.md)

### Community 126 - "Nav:Sidebar (4 nodes)"
Cohesion: 0.5
Nodes (4): Sidebar Navigation, Books (Light, Desktop), Dashboard (Light, Desktop), Libraries (Light, Desktop)

### Community 127 - "Feature:Api-Keys (4 nodes)"
Cohesion: 0.83
Nodes (4): API Key Management, Settings API Keys (Dark, Mobile), Settings API Keys Non-Admin (Dark, Desktop), Settings API Keys Non-Admin (Dark, Mobile)

### Community 128 - "Book-Listing (4 nodes)"
Cohesion: 0.67
Nodes (4): Book listing grid, Books listing (dark, desktop), Books listing (dark, mobile), Books listing (light, mobile)

### Community 129 - "Preferences (4 nodes)"
Cohesion: 0.5
Nodes (4): User preferences, Non-admin preferences (dark, desktop), Preferences settings (dark, mobile), Preferences settings (light, mobile)

### Community 130 - "Clipboard, Copytoclipboard"
Cohesion: 0.67
Nodes (0): 

### Community 131 - "Helpers, Validisbn10Checkdigit"
Cohesion: 0.67
Nodes (0): 

### Community 132 - "Transport, Goodreadsauthtransport"
Cohesion: 0.67
Nodes (1): GoodReadsAuthTransport

### Community 133 - "Kosynccredentialchecker, Middleware"
Cohesion: 0.67
Nodes (1): KOSyncCredentialChecker

### Community 134 - "Test, Testisbenigncleanupremoveerror"
Cohesion: 0.67
Nodes (0): 

### Community 135 - "Classification, Isbenigncleanupremoveerror"
Cohesion: 0.67
Nodes (0): 

### Community 136 - "Tracing, Starttracer"
Cohesion: 1.0
Nodes (2): StartTracer(), TraceMiddleware()

### Community 137 - "Send, Newclientwithcontext"
Cohesion: 1.0
Nodes (2): newClientWithContext(), Send()

### Community 138 - "Test, Testexiftooloutputsetisbn"
Cohesion: 0.67
Nodes (0): 

### Community 139 - "Test, Testisasin"
Cohesion: 0.67
Nodes (0): 

### Community 140 - "Isbn, Isasin"
Cohesion: 0.67
Nodes (0): 

### Community 141 - "Write, Namedentitycreate"
Cohesion: 0.67
Nodes (0): 

### Community 142 - "Setting, Settingexecer"
Cohesion: 0.67
Nodes (2): Setting, settingExecer

### Community 143 - "Test, Testsidecartarget"
Cohesion: 0.67
Nodes (0): 

### Community 144 - "Scanlibrarypayload, Library"
Cohesion: 0.67
Nodes (1): ScanLibraryPayload

### Community 145 - "Scanpathpayload, Path"
Cohesion: 0.67
Nodes (1): ScanPathPayload

### Community 146 - "Feature:Kobo-Sync, Screenshot:Settings-Kobo-Dark-Mobile"
Cohesion: 1.0
Nodes (3): Kobo E-Reader Sync, Settings Kobo (Dark, Mobile), Settings Kobo (Light, Desktop)

### Community 147 - "Screenshot:Settings-Nonadmin-Preferences-Dark-Mobile, Screenshot:Settings-Nonadmin-Preferences-Light-Mobile"
Cohesion: 0.67
Nodes (3): Settings Preferences Non-Admin (Dark, Mobile), Settings Preferences Non-Admin (Light, Mobile), Settings Preferences (Light, Desktop)

### Community 148 - "Config, Restoregitkeep"
Cohesion: 1.0
Nodes (0): 

### Community 149 - "Setup, Createlocalstoragemock"
Cohesion: 1.0
Nodes (0): 

### Community 150 - "Test, Makechildren"
Cohesion: 1.0
Nodes (0): 

### Community 151 - "Test, Makechildren"
Cohesion: 1.0
Nodes (0): 

### Community 152 - "Actions, Autofocusfirstbutton"
Cohesion: 1.0
Nodes (0): 

### Community 153 - "Test, Testnewclient"
Cohesion: 1.0
Nodes (0): 

### Community 154 - "Test, Testanalyzer"
Cohesion: 1.0
Nodes (0): 

### Community 155 - "Helpers, Mustgeneratedummybcrypthash"
Cohesion: 1.0
Nodes (0): 

### Community 156 - "Test, Testmustgeneratedummybcrypthash"
Cohesion: 1.0
Nodes (0): 

### Community 157 - "Linux, Renamenoreplace"
Cohesion: 1.0
Nodes (0): 

### Community 158 - "Other, Renamenoreplace"
Cohesion: 1.0
Nodes (0): 

### Community 159 - "Test, Testdialectorderby"
Cohesion: 1.0
Nodes (0): 

### Community 160 - "Create, Findorcreate"
Cohesion: 1.0
Nodes (0): 

### Community 161 - "Test, Newtestdb"
Cohesion: 1.0
Nodes (0): 

### Community 162 - "Tx, Deferrollback"
Cohesion: 1.0
Nodes (0): 

### Community 163 - "Decode, Decodedataurl"
Cohesion: 1.0
Nodes (0): 

### Community 164 - "Authors, Setbookauthorsrequest"
Cohesion: 1.0
Nodes (1): setBookAuthorsRequest

### Community 165 - "Headers, Securityheadersmiddleware"
Cohesion: 1.0
Nodes (0): 

### Community 166 - "Docs, Init"
Cohesion: 1.0
Nodes (0): 

### Community 167 - "Cla, Guide"
Cohesion: 1.0
Nodes (2): Contributor License Agreement, Contributing Guide

### Community 168 - "Endpoints, Only"
Cohesion: 1.0
Nodes (2): JWT-Only Endpoints, Rationale: JWT-Only Endpoint Restrictions

### Community 169 - "Screenshots/Libraries-Dark.Png, Screenshots/Libraries-Light-Mobile.Png"
Cohesion: 1.0
Nodes (2): Libraries Dark Desktop, Libraries Light Mobile

### Community 170 - "Screenshot:Login-Light-Mobile, Screenshot:Signup-Dark-Mobile"
Cohesion: 1.0
Nodes (2): Login (Light, Mobile), Signup (Dark, Mobile)

### Community 171 - "Concept:Design-System, Concept:Responsive-Layout"
Cohesion: 1.0
Nodes (2): Biblioteka Design System, Responsive Layout Strategy

### Community 172 - "Dashboard, Dashboard-Dark-Mobile"
Cohesion: 1.0
Nodes (2): Dashboard, Dashboard (dark, mobile)

### Community 173 - "Dark-Theme, Light-Theme"
Cohesion: 1.0
Nodes (2): Dark theme, Light theme

### Community 174 - "Config"
Cohesion: 1.0
Nodes (0): 

### Community 175 - "Config"
Cohesion: 1.0
Nodes (0): 

### Community 176 - "Test"
Cohesion: 1.0
Nodes (0): 

### Community 177 - "Test"
Cohesion: 1.0
Nodes (0): 

### Community 178 - "Test"
Cohesion: 1.0
Nodes (0): 

### Community 179 - "Test"
Cohesion: 1.0
Nodes (0): 

### Community 180 - "Test"
Cohesion: 1.0
Nodes (0): 

### Community 181 - "Test"
Cohesion: 1.0
Nodes (0): 

### Community 182 - "Gen"
Cohesion: 1.0
Nodes (0): 

### Community 183 - "Search"
Cohesion: 1.0
Nodes (0): 

### Community 184 - "Fixtures"
Cohesion: 1.0
Nodes (0): 

### Community 185 - "Frontend"
Cohesion: 1.0
Nodes (0): 

### Community 186 - "Doc"
Cohesion: 1.0
Nodes (0): 

### Community 187 - "Keys"
Cohesion: 1.0
Nodes (0): 

### Community 188 - "Queries"
Cohesion: 1.0
Nodes (0): 

### Community 189 - "Cover"
Cohesion: 1.0
Nodes (0): 

### Community 190 - "Telemetry"
Cohesion: 1.0
Nodes (1): Anonymous Telemetry (Opt-in)

### Community 191 - "Changelog"
Cohesion: 1.0
Nodes (1): Changelog

### Community 192 - "Screenshots/Settings-Nonadmin-Kobo-Light-Mobile.Png"
Cohesion: 1.0
Nodes (1): Settings Non-Admin Kobo Light Mobile

### Community 193 - "Responsive-Design"
Cohesion: 1.0
Nodes (1): Responsive design

## Knowledge Gaps
- **203 isolated node(s):** `Feed`, `Entry`, `Link`, `Author`, `Content` (+198 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Config, Restoregitkeep`** (2 nodes): `vite.config.ts`, `restoreGitkeep()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Setup, Createlocalstoragemock`** (2 nodes): `test-setup.ts`, `createLocalStorageMock()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test, Makechildren`** (2 nodes): `AlertBanner.test.ts`, `makeChildren()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test, Makechildren`** (2 nodes): `Button.test.ts`, `makeChildren()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Actions, Autofocusfirstbutton`** (2 nodes): `actions.ts`, `autofocusFirstButton()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test, Testnewclient`** (2 nodes): `client_test.go`, `TestNewClient()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test, Testanalyzer`** (2 nodes): `analyzer_test.go`, `TestAnalyzer()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Helpers, Mustgeneratedummybcrypthash`** (2 nodes): `bcrypt_helpers.go`, `mustGenerateDummyBcryptHash()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test, Testmustgeneratedummybcrypthash`** (2 nodes): `bcrypt_helpers_test.go`, `TestMustGenerateDummyBcryptHash()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Linux, Renamenoreplace`** (2 nodes): `rename_noreplace_linux.go`, `renameNoReplace()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Other, Renamenoreplace`** (2 nodes): `rename_noreplace_other.go`, `renameNoReplace()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test, Testdialectorderby`** (2 nodes): `db_test.go`, `TestDialectOrderBy()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Create, Findorcreate`** (2 nodes): `find_or_create.go`, `findOrCreate()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test, Newtestdb`** (2 nodes): `testhelper_test.go`, `newTestDB()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Tx, Deferrollback`** (2 nodes): `tx.go`, `deferRollback()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Decode, Decodedataurl`** (2 nodes): `decode.go`, `DecodeDataURL()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Authors, Setbookauthorsrequest`** (2 nodes): `books_authors.go`, `setBookAuthorsRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Headers, Securityheadersmiddleware`** (2 nodes): `security_headers.go`, `SecurityHeadersMiddleware()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Docs, Init`** (2 nodes): `docs.go`, `init()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Cla, Guide`** (2 nodes): `Contributor License Agreement`, `Contributing Guide`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Endpoints, Only`** (2 nodes): `JWT-Only Endpoints`, `Rationale: JWT-Only Endpoint Restrictions`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Screenshots/Libraries-Dark.Png, Screenshots/Libraries-Light-Mobile.Png`** (2 nodes): `Libraries Dark Desktop`, `Libraries Light Mobile`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Screenshot:Login-Light-Mobile, Screenshot:Signup-Dark-Mobile`** (2 nodes): `Login (Light, Mobile)`, `Signup (Dark, Mobile)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Concept:Design-System, Concept:Responsive-Layout`** (2 nodes): `Biblioteka Design System`, `Responsive Layout Strategy`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Dashboard, Dashboard-Dark-Mobile`** (2 nodes): `Dashboard`, `Dashboard (dark, mobile)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Dark-Theme, Light-Theme`** (2 nodes): `Dark theme`, `Light theme`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Config`** (1 nodes): `svelte.config.js`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Config`** (1 nodes): `eslint.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test`** (1 nodes): `MyLibrary.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test`** (1 nodes): `Sidebar.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test`** (1 nodes): `TextInput.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test`** (1 nodes): `UsersTab.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test`** (1 nodes): `APIKeysTab.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Test`** (1 nodes): `KoboTab.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Gen`** (1 nodes): `gen.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Search`** (1 nodes): `search.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Fixtures`** (1 nodes): `fixtures.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Frontend`** (1 nodes): `frontend.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Doc`** (1 nodes): `doc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Keys`** (1 nodes): `logger_keys.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Queries`** (1 nodes): `book_queries.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Cover`** (1 nodes): `kobo_cover.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Telemetry`** (1 nodes): `Anonymous Telemetry (Opt-in)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Changelog`** (1 nodes): `Changelog`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Screenshots/Settings-Nonadmin-Kobo-Light-Mobile.Png`** (1 nodes): `Settings Non-Admin Kobo Light Mobile`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Responsive-Design`** (1 nodes): `Responsive design`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `DB` connect `DB Models & CRUD` to `Admin & Auth Handlers`, `Library Management`, `Author Entity CRUD`?**
  _High betweenness centrality (0.042) - this node is a cross-community bridge._
- **Why does `Biblioteka Project` connect `Documentation Hub` to `Admin & Auth Handlers`?**
  _High betweenness centrality (0.009) - this node is a cross-community bridge._
- **Why does `OPDSHandler` connect `Handlers (17 nodes)` to `OPDS Feed Protocol`?**
  _High betweenness centrality (0.007) - this node is a cross-community bridge._
- **What connects `Feed`, `Entry`, `Link` to the rest of the system?**
  _203 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Admin & Auth Handlers` be split into smaller, more focused modules?**
  _Cohesion score 0.01 - nodes in this community are weakly interconnected._
- **Should `Book Metadata Pipeline` be split into smaller, more focused modules?**
  _Cohesion score 0.02 - nodes in this community are weakly interconnected._
- **Should `Kobo API Generated Code` be split into smaller, more focused modules?**
  _Cohesion score 0.01 - nodes in this community are weakly interconnected._