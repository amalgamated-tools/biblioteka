# Graph Report - .  (2026-04-09)

## Corpus Check
- 363 files · ~2,500,551 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2730 nodes · 3545 edges · 163 communities detected
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS
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
- `main()` --calls--> `realMain()`  [EXTRACTED]
  cmd/cli/main.go → cmd/server/main.go
- `TestListAPIKeys_Empty()` --calls--> `createTestUser()`  [EXTRACTED]
  internal/handlers/api_keys_test.go → internal/db/api_keys_test.go
- `TestDeleteAPIKey_NotFound()` --calls--> `createTestUser()`  [EXTRACTED]
  internal/handlers/api_keys_test.go → internal/db/api_keys_test.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (47): getFeatureEnabled(), getOidcEnabled(), getSignupEnabled(), openAuthPage(), openLoginForm(), openSignupForm(), signUp(), putBookSubResource() (+39 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (45): __marshalSearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchResultEdge(), __unmarshalSearchGetSearchSuggestionsSearchResultsConnectionEdgesSearchResultEdge(), GetBookByAsinGetBookByAsinBook, GetBookByAsinGetBookByAsinBookWork, GetBookByAsinGetBookByAsinBookWorkBestBook, GetBookByAsinGetBookByAsinBookWorkBestBookDetails, GetBookByAsinGetBookByAsinBookWorkBestBookDetailsLanguage, GetBookByAsinGetBookByAsinBookWorkBestBookPrimaryContributorEdgeBookContributorEdge (+37 more)

### Community 2 - "Community 2"
Cohesion: 0.02
Nodes (26): scanAPIKey(), scanAuditLog(), toAuditLogDTO(), scanAuthor(), bookFileColumnsWithPrefix(), scanBookFile(), bookColumnsWithPrefix(), scanBook() (+18 more)

### Community 3 - "Community 3"
Cohesion: 0.02
Nodes (37): requireExifToolExtractor(), TestEPUB3CoverImport_EndToEnd(), isBookFilePathAllowed(), isPathUnderRoot(), finishBook(), finishEPUB(), finishMOBI(), hasProperty() (+29 more)

### Community 4 - "Community 4"
Cohesion: 0.02
Nodes (9): OPDSHandler, writeEntityBooksFeed(), writeOPDSError(), writeOPDSFeed(), padInt(), TestAllBooks_Pagination(), setupOPDSHandler(), TestHandleOPDS_MethodNotAllowed() (+1 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (22): isSlogAny(), run(), AuthStore, AuthorStore, CrudStore, ExifToolOutput, GuideReference, Identifier (+14 more)

### Community 6 - "Community 6"
Cohesion: 0.04
Nodes (34): checkDuplicate(), linkExistingBookAndSkip(), reorganizedCandidatePaths(), resolveSourcePath(), validateField(), validatePayload(), bookFileLookupFunc, cleanEmptyDirs() (+26 more)

### Community 7 - "Community 7"
Cohesion: 0.04
Nodes (39): adminCacheEntry, AdminChecker, APIKeyValidator, cachingAdminChecker, contextKey, tokenSource, main(), pathExists() (+31 more)

### Community 8 - "Community 8"
Cohesion: 0.04
Nodes (15): NormalizeAuthorName(), normalizeName(), ApiError, getToken(), request(), Author, authorListQuery, Series (+7 more)

### Community 9 - "Community 9"
Cohesion: 0.06
Nodes (33): fetchMetadataResponse, metadataDTO, MetadataHandler, ProcessFilePayload, ActivePeriod, Bookmark, ContributorRole, DisplayPrice (+25 more)

### Community 10 - "Community 10"
Cohesion: 0.06
Nodes (11): dollarN(), toBookFileDTO(), Book, BookInput, bookDTO, bookFileDTO, BookHandler, bookListDTO (+3 more)

### Community 11 - "Community 11"
Cohesion: 0.07
Nodes (24): setupBookHandler(), TestBookAuthors_Handler(), TestBookFiles_Handler(), TestBookSeries_Handler(), TestCreateBook_EnqueueFailureDoesNotFailRequest(), TestCreateBook_EnqueuesGoodreadsJob(), TestCreateBook_Handler(), TestCreateBook_MissingTitle() (+16 more)

### Community 12 - "Community 12"
Cohesion: 0.1
Nodes (32): createTestUserForKOSync(), setupKOSyncHandler(), TestKOSyncCredential_Delete(), TestKOSyncCredential_Upsert_UpdatesExisting(), TestKOSyncCredential_UpsertAndGet(), TestKOSyncCredential_UsernameConflict(), TestKOSyncCredentials_Delete(), TestKOSyncCredentials_Delete_NotFound() (+24 more)

### Community 13 - "Community 13"
Cohesion: 0.08
Nodes (13): Library, libraryListQuery, libraryDTO, LibraryHandler, libraryRequest, pathValidationError, isColumnUniqueViolation(), isUniqueViolation() (+5 more)

### Community 14 - "Community 14"
Cohesion: 0.09
Nodes (20): mockSubscriber, createTestBook(), MockEventSource, setupMetadataHandler(), strPtr(), TestApplyMetadata(), TestApplyMetadata_NoPending(), TestBookMetadata_Language_Set() (+12 more)

### Community 15 - "Community 15"
Cohesion: 0.08
Nodes (14): setupAuthorHandler(), TestCreateAuthor_Duplicate(), TestCreateAuthor_Handler(), TestCreateAuthor_MissingName(), TestCreateAuthor_WhitespaceOnlyName(), TestDeleteAuthor_Handler(), TestDeleteAuthor_NotFound(), TestGetAuthor_Handler() (+6 more)

### Community 16 - "Community 16"
Cohesion: 0.08
Nodes (14): setupSeriesHandler(), TestCreateSeries_Duplicate(), TestCreateSeries_Handler(), TestCreateSeries_MissingName(), TestCreateSeries_WhitespaceOnlyName(), TestDeleteSeries_Handler(), TestDeleteSeries_NotFound(), TestGetSeries_Handler() (+6 more)

### Community 17 - "Community 17"
Cohesion: 0.06
Nodes (0): 

### Community 18 - "Community 18"
Cohesion: 0.08
Nodes (16): IsLoopbackHost(), isValidSMTPHostForStatus(), isPrivateIP(), ssrfSafeHTTPClient(), validateOIDCIssuerURL(), ValidateForSend(), ValidateHost(), validationErr() (+8 more)

### Community 19 - "Community 19"
Cohesion: 0.09
Nodes (19): flusherRecorder, plainResponseWriter, assertJSONError(), TestLoggingMiddleware_CallsNext(), TestLoggingMiddleware_ImplicitOKOnWrite(), TestLoggingMiddleware_PreservesStatus(), TestMiddleware_InvalidAuthorizationFormat(), TestMiddleware_InvalidToken() (+11 more)

### Community 20 - "Community 20"
Cohesion: 0.07
Nodes (5): linkNonce, OIDCHandler, newTestOIDCHandlerWithTokenResponse(), TestOIDCCallback_MissingIDToken(), generateState()

### Community 21 - "Community 21"
Cohesion: 0.14
Nodes (30): assertAuthCookie(), mustSignup(), newAuthHandler(), TestChangePassword_Invalid(), TestChangePassword_MethodNotAllowed(), TestChangePassword_Success(), TestLogin_MethodNotAllowed(), TestLogin_MissingFields() (+22 more)

### Community 22 - "Community 22"
Cohesion: 0.11
Nodes (19): normalizeExifDate(), requireExifTool(), TestExtractMetadata_EPUB(), TestExtractMetadata_EPUB2And3ProduceSameMetadata(), TestExtractMetadata_EPUB2CoverViaMeta(), TestExtractMetadata_EPUB2Metadata(), TestExtractMetadata_EPUB3CoverViaProperties(), TestExtractMetadata_EPUB3Metadata() (+11 more)

### Community 23 - "Community 23"
Cohesion: 0.12
Nodes (23): Claims, JWTManager, testOIDCProvider, callbackRequest(), newTestOIDCHandler(), newTestOIDCProvider(), requireOIDCCookies(), seedLinkNonce() (+15 more)

### Community 24 - "Community 24"
Cohesion: 0.07
Nodes (0): 

### Community 25 - "Community 25"
Cohesion: 0.07
Nodes (0): 

### Community 26 - "Community 26"
Cohesion: 0.13
Nodes (26): testEntity, testEntityDTO, testEntityRequest, makeTestNamedEntityOps(), TestCreateNamedEntity_EmptyName(), TestCreateNamedEntity_ErrInvalidName(), TestCreateNamedEntity_ErrNameExists(), TestCreateNamedEntity_GenericCreateError() (+18 more)

### Community 27 - "Community 27"
Cohesion: 0.1
Nodes (11): mockGraphQLClient, mockHTTPClient, noResponseClient(), TestParseISBNSearchResponse_CancelledContext(), TestParseISBNSearchResponse_EmptyArray(), TestParseISBNSearchResponse_InvalidBookID(), TestParseISBNSearchResponse_InvalidJSON(), TestParseISBNSearchResponse_InvalidWorkID() (+3 more)

### Community 28 - "Community 28"
Cohesion: 0.14
Nodes (21): createTestUser(), setupAPIKeyHandler(), TestCreateAPIKey(), TestCreateAPIKey_AuditLog(), TestCreateAPIKey_EmptyName(), TestCreateAPIKey_NameTooLong(), TestCreateAPIKey_Success(), TestDeleteAPIKey() (+13 more)

### Community 29 - "Community 29"
Cohesion: 0.16
Nodes (15): createTestBookWithFields(), createTestUser(), TestEnrichGoodreads_ASINLookup(), TestEnrichGoodreads_GoodreadsIDLookup(), TestEnrichGoodreads_ISBN10Lookup(), TestEnrichGoodreads_ISBNFailsFallsToTitle(), TestEnrichGoodreads_ISBNLookup(), TestEnrichGoodreads_ISBNPreferredOverTitle() (+7 more)

### Community 30 - "Community 30"
Cohesion: 0.21
Nodes (23): testKoboTokenChecker, createTestKoboToken(), koboDeviceHandler(), TestHandleKobo_AnalyticsGetTests(), TestHandleKobo_Auth_Stub(), TestHandleKobo_BookMetadata_NonGET(), TestHandleKobo_BookMetadata_NotFound(), TestHandleKobo_BookMetadata_Success() (+15 more)

### Community 31 - "Community 31"
Cohesion: 0.08
Nodes (0): 

### Community 32 - "Community 32"
Cohesion: 0.14
Nodes (13): mustParseCIDR(), TestIpFromRequestTrusted_AllTrusted(), TestIpFromRequestTrusted_IPv6(), TestIpFromRequestTrusted_IPv6_AllTrusted(), TestIpFromRequestTrusted_MultipleProxies(), TestIpFromRequestTrusted_NoXFF(), TestIpFromRequestTrusted_PortSuffixedXFF(), TestIpFromRequestTrusted_RemoteNotTrusted() (+5 more)

### Community 33 - "Community 33"
Cohesion: 0.1
Nodes (4): newTestDB(), TestNewServer_DefaultPort(), TestNewServer_WithDB(), TestNewServer_WithPort()

### Community 34 - "Community 34"
Cohesion: 0.11
Nodes (5): setupConfigHandler(), TestHandleConfigStatus_MethodNotAllowed(), TestHandleConfigStatus_RegularUser(), TestHandleConfigStatus_Success(), TestHandleConfigStatus_WhenConfigured()

### Community 35 - "Community 35"
Cohesion: 0.1
Nodes (4): AutoDismissTimer, CopyTimeoutState, TimeoutState, TestTimeoutState

### Community 36 - "Community 36"
Cohesion: 0.15
Nodes (15): fakeRow, sample, memDB(), scanSampleAndTotal(), TestCollectRows_AlwaysClosesRows(), TestCollectRows_ClosesRowsOnError(), TestCollectRows_EmptyResult(), TestCollectRows_HappyPath() (+7 more)

### Community 37 - "Community 37"
Cohesion: 0.1
Nodes (0): 

### Community 38 - "Community 38"
Cohesion: 0.16
Nodes (12): setupAuditLogHandler(), TestHandleAuditLogs_AdminSuccess(), TestHandleAuditLogs_CustomPagination(), TestHandleAuditLogs_DefaultPagination(), TestHandleAuditLogs_EmptyList(), TestHandleAuditLogs_InvalidLimit(), TestHandleAuditLogs_InvalidOffset(), TestHandleAuditLogs_LimitCappedAtMax() (+4 more)

### Community 39 - "Community 39"
Cohesion: 0.13
Nodes (6): addZipEntry(), makeEPUBWithCover(), TestExtractEPUBCoverDataURL_EPUB3Cover(), TestExtractEPUBCoverDataURL_MissingArchiveFile(), TestParseTSV_EPUB2CoverExtractionE2E(), TestParseTSV_EPUB3CoverExtractionE2E()

### Community 40 - "Community 40"
Cohesion: 0.13
Nodes (3): KoboHandler, koboRandomUUID(), writeKoboJSON()

### Community 41 - "Community 41"
Cohesion: 0.14
Nodes (7): loadBookResult(), autocompleteEntry, Client, HttpClient, workData, buildFallbackResult(), parseAutocompleteEntries()

### Community 42 - "Community 42"
Cohesion: 0.21
Nodes (14): createTestBookForKobo(), koboTestFutureTime(), koboTestPastTime(), TestGetKoboReadingState_NotFound(), TestGetKoboReadingState_UserIsolation(), TestGetReadingStatesForBooks_ReturnsMap(), TestGetReadingStatesForBooks_UserIsolation(), TestGetReadingStatesForBooks_WithSinceFilter() (+6 more)

### Community 43 - "Community 43"
Cohesion: 0.19
Nodes (7): checkSystemEndpointMethod(), swaggerSecurityHeaders(), writeSystemJSON(), enabledResponse, healthResponse, Server, versionResponse

### Community 44 - "Community 44"
Cohesion: 0.12
Nodes (1): invalidListQuery

### Community 45 - "Community 45"
Cohesion: 0.2
Nodes (12): newTestWorker(), TestEnqueue_NonMarshalablePayload(), TestNew_ValidURL(), TestRedisConnOpt(), TestRegister(), TestRegister_HandlerError(), TestRegister_NilPayload(), TestRegisterSchedule() (+4 more)

### Community 46 - "Community 46"
Cohesion: 0.18
Nodes (8): networkError, TestSearchByISBN_EmptyISBN(), TestSearchByISBN_HTTPFailure(), TestSearchByISBN_InvalidISBN10CheckDigit(), TestSearchByISBN_InvalidISBN13CheckDigit(), TestSearchByISBN_InvalidLength(), TestSearchByISBN_NonOKStatus(), TestSearchByISBN_ResponseTooLarge()

### Community 47 - "Community 47"
Cohesion: 0.12
Nodes (0): 

### Community 48 - "Community 48"
Cohesion: 0.12
Nodes (0): 

### Community 49 - "Community 49"
Cohesion: 0.13
Nodes (0): 

### Community 50 - "Community 50"
Cohesion: 0.25
Nodes (13): setupAdminHandler(), TestHandleListUsers_AdminSuccess(), TestHandleListUsers_MethodNotAllowed(), TestHandleListUsers_NonAdminForbidden(), TestHandleListUsers_ResponseContainsAdminFlag(), TestHandleSetAdmin_AdminDemotesUser(), TestHandleSetAdmin_AdminPromotesUser(), TestHandleSetAdmin_CannotChangeSelf() (+5 more)

### Community 51 - "Community 51"
Cohesion: 0.18
Nodes (6): createTestLibrary(), registerTestLibrary(), TestPostBookFiles_AuditLog(), TestPostBookFiles_PathOutsideLibrary(), TestPostBookFiles_PathTraversal(), TestPostBookFiles_Success()

### Community 52 - "Community 52"
Cohesion: 0.2
Nodes (4): Exiftool, FileMetadata, handleWriteMetadataResponse(), toString()

### Community 53 - "Community 53"
Cohesion: 0.2
Nodes (5): ReadingProgress, KOSyncHandler, kosyncProgressRequest, kosyncProgressResponse, toKOSyncProgressResponse()

### Community 54 - "Community 54"
Cohesion: 0.15
Nodes (0): 

### Community 55 - "Community 55"
Cohesion: 0.15
Nodes (1): errEnqueuer

### Community 56 - "Community 56"
Cohesion: 0.17
Nodes (0): 

### Community 57 - "Community 57"
Cohesion: 0.27
Nodes (7): RateLimiter, visitor, ipFromRequest(), ipFromRequestTrusted(), isTrusted(), NewRateLimiter(), NewRateLimiterWithTrustedProxies()

### Community 58 - "Community 58"
Cohesion: 0.32
Nodes (11): archiveCandidates(), cleanArchivePath(), extensionForMIME(), extractEPUBCoverDataURL(), findArchiveFile(), findEPUBCoverRef(), readEPUBArchiveFile(), readEPUBRootFilePath() (+3 more)

### Community 59 - "Community 59"
Cohesion: 0.17
Nodes (0): 

### Community 60 - "Community 60"
Cohesion: 0.21
Nodes (4): setupBookFileHandler(), TestDeleteBookFile_Handler(), TestGetBookFile_Handler(), TestGetBookFile_NotFound()

### Community 61 - "Community 61"
Cohesion: 0.23
Nodes (6): toAPIKeyDTO(), APIKey, apiKeyCreateRequest, apiKeyCreateResponse, apiKeyDTO, APIKeyHandler

### Community 62 - "Community 62"
Cohesion: 0.24
Nodes (5): mockOPDSChecker, newOPDSCheckerWithUser(), TestOPDSBasicAuth_Success(), TestOPDSBasicAuth_UsernameLowercased(), TestOPDSBasicAuth_WrongPassword()

### Community 63 - "Community 63"
Cohesion: 0.24
Nodes (5): mockKOSyncChecker, newKOSyncCheckerWithUser(), TestKOSyncHeaderAuth_Success(), TestKOSyncHeaderAuth_UsernameLowercased(), TestKOSyncHeaderAuth_WrongKey()

### Community 64 - "Community 64"
Cohesion: 0.4
Nodes (10): flushIdent(), flushManifest(), flushMeta(), isLikelyImage(), parseIdentifierLine(), parseManifestLine(), parseMetaLine(), parseScalar() (+2 more)

### Community 65 - "Community 65"
Cohesion: 0.18
Nodes (1): fakeEntity

### Community 66 - "Community 66"
Cohesion: 0.33
Nodes (10): createGoodreadsMetadataFromResult(), enrichGoodreads(), lookupGoodreads(), NewEnrichGoodreadsHandler(), publishEvent(), publishProgress(), titleSimilar(), EnrichGoodreadsPayload (+2 more)

### Community 67 - "Community 67"
Cohesion: 0.25
Nodes (7): buildEXTH(), buildMOBI(), MakeTestAZW3(), MakeTestMOBI(), uint32ToBytes(), exthRecord, MOBIOptions

### Community 68 - "Community 68"
Cohesion: 0.2
Nodes (0): 

### Community 69 - "Community 69"
Cohesion: 0.2
Nodes (0): 

### Community 70 - "Community 70"
Cohesion: 0.2
Nodes (0): 

### Community 71 - "Community 71"
Cohesion: 0.22
Nodes (1): mockAdminChecker

### Community 72 - "Community 72"
Cohesion: 0.22
Nodes (1): mockLibraryLister

### Community 73 - "Community 73"
Cohesion: 0.22
Nodes (0): 

### Community 74 - "Community 74"
Cohesion: 0.22
Nodes (2): genericEnqueuedJob, genericMockEnqueuer

### Community 75 - "Community 75"
Cohesion: 0.22
Nodes (0): 

### Community 76 - "Community 76"
Cohesion: 0.22
Nodes (0): 

### Community 77 - "Community 77"
Cohesion: 0.33
Nodes (6): MakeTestEPUB(), MakeTestEPUBWithOptions(), writeZipFile(), writeZipFileBytes(), xmlEscape(), EPUBOptions

### Community 78 - "Community 78"
Cohesion: 0.25
Nodes (0): 

### Community 79 - "Community 79"
Cohesion: 0.25
Nodes (1): mockAPIKeyValidator

### Community 80 - "Community 80"
Cohesion: 0.25
Nodes (0): 

### Community 81 - "Community 81"
Cohesion: 0.29
Nodes (0): 

### Community 82 - "Community 82"
Cohesion: 0.48
Nodes (5): TestAuthTransport_AddsKeyForAllowedHost(), TestAuthTransport_DoesNotMutateOriginalRequest(), TestAuthTransport_EmptyAllowedHostSendsKeyEverywhere(), TestAuthTransport_OmitsKeyForDifferentHost(), recordingTransport

### Community 83 - "Community 83"
Cohesion: 0.33
Nodes (4): KoboTokenChecker, KoboTokenResult, KoboTokenAuthMiddleware(), writeKoboJSONError()

### Community 84 - "Community 84"
Cohesion: 0.29
Nodes (1): mockKoboTokenChecker

### Community 85 - "Community 85"
Cohesion: 0.29
Nodes (3): fakeItem, fakeItemDTO, fakeSetRequest

### Community 86 - "Community 86"
Cohesion: 0.33
Nodes (0): 

### Community 87 - "Community 87"
Cohesion: 0.4
Nodes (4): scanBookSeriesEntry(), BookSeriesEntry, BookSeriesInput, prefixedScanner

### Community 88 - "Community 88"
Cohesion: 0.67
Nodes (4): isSQLIdentifierRune(), removeInlineComments(), scanDollarTag(), splitStatements()

### Community 89 - "Community 89"
Cohesion: 0.47
Nodes (5): errorResponse, decodeJSON(), writeError(), writeJSON(), writeSecretTokenResponse()

### Community 90 - "Community 90"
Cohesion: 0.6
Nodes (5): defaultPort(), matchRequestOrigin(), normalizeHost(), parseHostPort(), sameOrigin()

### Community 91 - "Community 91"
Cohesion: 0.33
Nodes (0): 

### Community 92 - "Community 92"
Cohesion: 0.4
Nodes (2): errorValue, myStruct

### Community 93 - "Community 93"
Cohesion: 0.4
Nodes (1): Payload

### Community 94 - "Community 94"
Cohesion: 0.4
Nodes (0): 

### Community 95 - "Community 95"
Cohesion: 0.4
Nodes (1): listQuery

### Community 96 - "Community 96"
Cohesion: 0.6
Nodes (3): redisTestURL(), TestPublishSubscribe(), TestSubscribeCancelStopsChannel()

### Community 97 - "Community 97"
Cohesion: 0.4
Nodes (0): 

### Community 98 - "Community 98"
Cohesion: 0.5
Nodes (2): coverMIMEType(), dataURLMIMEType()

### Community 99 - "Community 99"
Cohesion: 0.4
Nodes (0): 

### Community 100 - "Community 100"
Cohesion: 0.4
Nodes (0): 

### Community 101 - "Community 101"
Cohesion: 0.4
Nodes (0): 

### Community 102 - "Community 102"
Cohesion: 0.5
Nodes (3): contextKey, RequestIDHandler(), WithRequestID()

### Community 103 - "Community 103"
Cohesion: 0.5
Nodes (0): 

### Community 104 - "Community 104"
Cohesion: 0.67
Nodes (3): OPDSCredentialChecker, OPDSBasicAuthMiddleware(), writeOPDSError()

### Community 105 - "Community 105"
Cohesion: 0.5
Nodes (0): 

### Community 106 - "Community 106"
Cohesion: 0.5
Nodes (0): 

### Community 107 - "Community 107"
Cohesion: 0.5
Nodes (0): 

### Community 108 - "Community 108"
Cohesion: 0.5
Nodes (0): 

### Community 109 - "Community 109"
Cohesion: 0.5
Nodes (0): 

### Community 110 - "Community 110"
Cohesion: 0.5
Nodes (0): 

### Community 111 - "Community 111"
Cohesion: 0.5
Nodes (0): 

### Community 112 - "Community 112"
Cohesion: 0.67
Nodes (0): 

### Community 113 - "Community 113"
Cohesion: 0.67
Nodes (0): 

### Community 114 - "Community 114"
Cohesion: 0.67
Nodes (1): GoodReadsAuthTransport

### Community 115 - "Community 115"
Cohesion: 0.67
Nodes (1): KOSyncCredentialChecker

### Community 116 - "Community 116"
Cohesion: 0.67
Nodes (0): 

### Community 117 - "Community 117"
Cohesion: 0.67
Nodes (0): 

### Community 118 - "Community 118"
Cohesion: 1.0
Nodes (2): StartTracer(), TraceMiddleware()

### Community 119 - "Community 119"
Cohesion: 1.0
Nodes (2): newClientWithContext(), Send()

### Community 120 - "Community 120"
Cohesion: 0.67
Nodes (0): 

### Community 121 - "Community 121"
Cohesion: 0.67
Nodes (0): 

### Community 122 - "Community 122"
Cohesion: 0.67
Nodes (0): 

### Community 123 - "Community 123"
Cohesion: 0.67
Nodes (0): 

### Community 124 - "Community 124"
Cohesion: 0.67
Nodes (2): Setting, settingExecer

### Community 125 - "Community 125"
Cohesion: 0.67
Nodes (0): 

### Community 126 - "Community 126"
Cohesion: 0.67
Nodes (1): ScanLibraryPayload

### Community 127 - "Community 127"
Cohesion: 0.67
Nodes (1): ScanPathPayload

### Community 128 - "Community 128"
Cohesion: 1.0
Nodes (0): 

### Community 129 - "Community 129"
Cohesion: 1.0
Nodes (0): 

### Community 130 - "Community 130"
Cohesion: 1.0
Nodes (0): 

### Community 131 - "Community 131"
Cohesion: 1.0
Nodes (0): 

### Community 132 - "Community 132"
Cohesion: 1.0
Nodes (0): 

### Community 133 - "Community 133"
Cohesion: 1.0
Nodes (0): 

### Community 134 - "Community 134"
Cohesion: 1.0
Nodes (0): 

### Community 135 - "Community 135"
Cohesion: 1.0
Nodes (0): 

### Community 136 - "Community 136"
Cohesion: 1.0
Nodes (0): 

### Community 137 - "Community 137"
Cohesion: 1.0
Nodes (0): 

### Community 138 - "Community 138"
Cohesion: 1.0
Nodes (0): 

### Community 139 - "Community 139"
Cohesion: 1.0
Nodes (0): 

### Community 140 - "Community 140"
Cohesion: 1.0
Nodes (0): 

### Community 141 - "Community 141"
Cohesion: 1.0
Nodes (0): 

### Community 142 - "Community 142"
Cohesion: 1.0
Nodes (0): 

### Community 143 - "Community 143"
Cohesion: 1.0
Nodes (0): 

### Community 144 - "Community 144"
Cohesion: 1.0
Nodes (1): setBookAuthorsRequest

### Community 145 - "Community 145"
Cohesion: 1.0
Nodes (0): 

### Community 146 - "Community 146"
Cohesion: 1.0
Nodes (0): 

### Community 147 - "Community 147"
Cohesion: 1.0
Nodes (0): 

### Community 148 - "Community 148"
Cohesion: 1.0
Nodes (0): 

### Community 149 - "Community 149"
Cohesion: 1.0
Nodes (0): 

### Community 150 - "Community 150"
Cohesion: 1.0
Nodes (0): 

### Community 151 - "Community 151"
Cohesion: 1.0
Nodes (0): 

### Community 152 - "Community 152"
Cohesion: 1.0
Nodes (0): 

### Community 153 - "Community 153"
Cohesion: 1.0
Nodes (0): 

### Community 154 - "Community 154"
Cohesion: 1.0
Nodes (0): 

### Community 155 - "Community 155"
Cohesion: 1.0
Nodes (0): 

### Community 156 - "Community 156"
Cohesion: 1.0
Nodes (0): 

### Community 157 - "Community 157"
Cohesion: 1.0
Nodes (0): 

### Community 158 - "Community 158"
Cohesion: 1.0
Nodes (0): 

### Community 159 - "Community 159"
Cohesion: 1.0
Nodes (0): 

### Community 160 - "Community 160"
Cohesion: 1.0
Nodes (0): 

### Community 161 - "Community 161"
Cohesion: 1.0
Nodes (0): 

### Community 162 - "Community 162"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **143 isolated node(s):** `Feed`, `Entry`, `Link`, `Author`, `Content` (+138 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 128`** (2 nodes): `vite.config.ts`, `restoreGitkeep()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 129`** (2 nodes): `test-setup.ts`, `createLocalStorageMock()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 130`** (2 nodes): `AlertBanner.test.ts`, `makeChildren()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 131`** (2 nodes): `Button.test.ts`, `makeChildren()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 132`** (2 nodes): `actions.ts`, `autofocusFirstButton()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 133`** (2 nodes): `client_test.go`, `TestNewClient()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 134`** (2 nodes): `analyzer_test.go`, `TestAnalyzer()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 135`** (2 nodes): `bcrypt_helpers.go`, `mustGenerateDummyBcryptHash()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 136`** (2 nodes): `bcrypt_helpers_test.go`, `TestMustGenerateDummyBcryptHash()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 137`** (2 nodes): `rename_noreplace_linux.go`, `renameNoReplace()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 138`** (2 nodes): `rename_noreplace_other.go`, `renameNoReplace()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 139`** (2 nodes): `db_test.go`, `TestDialectOrderBy()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 140`** (2 nodes): `find_or_create.go`, `findOrCreate()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 141`** (2 nodes): `testhelper_test.go`, `newTestDB()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 142`** (2 nodes): `tx.go`, `deferRollback()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 143`** (2 nodes): `decode.go`, `DecodeDataURL()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 144`** (2 nodes): `books_authors.go`, `setBookAuthorsRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 145`** (2 nodes): `security_headers.go`, `SecurityHeadersMiddleware()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 146`** (2 nodes): `docs.go`, `init()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 147`** (1 nodes): `svelte.config.js`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 148`** (1 nodes): `eslint.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 149`** (1 nodes): `MyLibrary.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 150`** (1 nodes): `Sidebar.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 151`** (1 nodes): `TextInput.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 152`** (1 nodes): `UsersTab.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 153`** (1 nodes): `APIKeysTab.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 154`** (1 nodes): `KoboTab.test.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 155`** (1 nodes): `gen.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 156`** (1 nodes): `search.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 157`** (1 nodes): `fixtures.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 158`** (1 nodes): `frontend.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 159`** (1 nodes): `doc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 160`** (1 nodes): `logger_keys.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 161`** (1 nodes): `book_queries.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 162`** (1 nodes): `kobo_cover.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `DB` connect `Community 2` to `Community 0`, `Community 8`, `Community 10`, `Community 13`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 43` to `Community 7`?**
  _High betweenness centrality (0.007) - this node is a cross-community bridge._
- **Why does `KoboHandler` connect `Community 40` to `Community 0`?**
  _High betweenness centrality (0.007) - this node is a cross-community bridge._
- **What connects `Feed`, `Entry`, `Link` to the rest of the system?**
  _143 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.01 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.01 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.02 - nodes in this community are weakly interconnected._