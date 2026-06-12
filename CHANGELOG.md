# Changelog

## [0.20.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.19.0...v0.20.0) (2026-06-05)


### Features

* **generate_books:** add link to generate_books.py script in new text file ([893d05a](https://github.com/amalgamated-tools/biblioteka/commit/893d05a749a8db743d0c54c36f1366b951adcfde))


### Bug Fixes

* **docker-compose:** update JWT_SECRET default value for security and add books volume ([893d05a](https://github.com/amalgamated-tools/biblioteka/commit/893d05a749a8db743d0c54c36f1366b951adcfde))
* recompile ([3ff3719](https://github.com/amalgamated-tools/biblioteka/commit/3ff3719ec6faf70364c207e444ee22cc5098c8b3))

## [0.19.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.18.0...v0.19.0) (2026-05-27)


### Features

* **worker:** asynq improvements — priority queues, opt-in Unique, slog adapter, trace propagation, graceful shutdown ([#3079](https://github.com/amalgamated-tools/biblioteka/issues/3079)) ([126235c](https://github.com/amalgamated-tools/biblioteka/commit/126235c57c0b98e2c3895ef9cf68599234ba9d95))

## [0.18.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.17.0...v0.18.0) (2026-05-26)


### Features

* add technical documentation writer and jqschema utility; update daily doc healer workflow ([4622dca](https://github.com/amalgamated-tools/biblioteka/commit/4622dca0ec7386d4f8f672603e69e524bce2371d))


### Bug Fixes

* Architecture Guardian metric collection on zero-match grep counts ([#2968](https://github.com/amalgamated-tools/biblioteka/issues/2968)) ([31a8038](https://github.com/amalgamated-tools/biblioteka/commit/31a803879567d672f1a96492542612c7cf86a46f))
* daily file diet ([1f8326b](https://github.com/amalgamated-tools/biblioteka/commit/1f8326b7361bd8ba3b9121ca34ec2768f6d33244))
* missing agent ([852c7b8](https://github.com/amalgamated-tools/biblioteka/commit/852c7b8728601ed88cbcc9f0f1a09678cd2c6963))

## [0.17.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.16.0...v0.17.0) (2026-05-07)


### Features

* **audit:** add audit logging for book relationships and OIDC config changes ([#2735](https://github.com/amalgamated-tools/biblioteka/issues/2735)) ([14f2201](https://github.com/amalgamated-tools/biblioteka/commit/14f2201e8e4cb19e74f8fd908bbcc546bb53fc34))
* **frontend:** AI enrichment review panel on book detail page ([#2732](https://github.com/amalgamated-tools/biblioteka/issues/2732)) ([3be700e](https://github.com/amalgamated-tools/biblioteka/commit/3be700e66872eff17a1f70388f3f3176bf3e3be3))
* **handlers:** introduce userOwnedNamedEntityOps to eliminate duplicate user-owned CRUD pattern ([#2822](https://github.com/amalgamated-tools/biblioteka/issues/2822)) ([e1b28b8](https://github.com/amalgamated-tools/biblioteka/commit/e1b28b82c080be296b4a0dd7b0d6a500edb4c603))
* **handlers:** paginate GET /api/authors, /api/series, and /api/tags ([#2863](https://github.com/amalgamated-tools/biblioteka/issues/2863)) ([73fa826](https://github.com/amalgamated-tools/biblioteka/commit/73fa826aef22fffcecb8780af9456e5bfd8a164c))
* **registration:** add runtime registration_disabled setting to lock public self-registration ([#2731](https://github.com/amalgamated-tools/biblioteka/issues/2731)) ([5e2fc24](https://github.com/amalgamated-tools/biblioteka/commit/5e2fc246d634c3d449053b0eb7d1786accd1ef93))


### Bug Fixes

* **db:** GetAnnotation includes group-visible annotations for members ([#2894](https://github.com/amalgamated-tools/biblioteka/issues/2894)) ([9f9a153](https://github.com/amalgamated-tools/biblioteka/commit/9f9a153fa772d44c0797894e828a6996ed53f9b2))
* **db:** reuse execAffected for AI enrichment and Goodreads metadata deletes ([#2909](https://github.com/amalgamated-tools/biblioteka/issues/2909)) ([18ae0d4](https://github.com/amalgamated-tools/biblioteka/commit/18ae0d44b8eae58682dc844e03826b1806a6aef7))
* **handlers:** align group delete audit entity_type and resource label to "group" ([#2860](https://github.com/amalgamated-tools/biblioteka/issues/2860)) ([ce095b1](https://github.com/amalgamated-tools/biblioteka/commit/ce095b1ad7d586f75a014c9662eba8b75acd225c))
* **handlers:** replace http.Error() with writeError()/writeOPDSError() and add structured logging ([#2882](https://github.com/amalgamated-tools/biblioteka/issues/2882)) ([b11d035](https://github.com/amalgamated-tools/biblioteka/commit/b11d035764f5a88618a61421e4e835b30abd5c67))
* **llm:** cap Ollama response body with io.LimitReader ([#2893](https://github.com/amalgamated-tools/biblioteka/issues/2893)) ([577be85](https://github.com/amalgamated-tools/biblioteka/commit/577be851cb079ecda67babbafb0bb42328c45688))
* **recommendations:** respect `offset` query parameter in GET /api/recommendations ([#2806](https://github.com/amalgamated-tools/biblioteka/issues/2806)) ([30723b5](https://github.com/amalgamated-tools/biblioteka/commit/30723b51fa06e54d3e16ec7b7fd1d27e1012986f))
* **repo assist:** fix(opds): replace sync.WaitGroup with errgroup; harden cover URL SSRF check ([#2844](https://github.com/amalgamated-tools/biblioteka/issues/2844)) ([3a042d8](https://github.com/amalgamated-tools/biblioteka/commit/3a042d8300ac407099a20617293f900ba97d6f01))
* **repo assist:** test(handlers): add audit metadata test for reading list deletion ([#2826](https://github.com/amalgamated-tools/biblioteka/issues/2826)) ([855ca22](https://github.com/amalgamated-tools/biblioteka/commit/855ca223e4086d1e5959707765070396e0b7d157))
* **smtp:** block private/loopback/link-local SMTP hosts (SSRF) ([#2736](https://github.com/amalgamated-tools/biblioteka/issues/2736)) ([a3bd164](https://github.com/amalgamated-tools/biblioteka/commit/a3bd164bc208906e96b6e8823c7298710c29689e))
* **workflows:** revert gh-aw-actions/setup from v0.71.3 to v0.71.0 ([#2763](https://github.com/amalgamated-tools/biblioteka/issues/2763)) ([fe0b337](https://github.com/amalgamated-tools/biblioteka/commit/fe0b33746e757130b85f48b555d3444850d5d342))


### Performance Improvements

* add (book_id, created_at) index on book_annotations to eliminate ORDER BY filesort ([#2474](https://github.com/amalgamated-tools/biblioteka/issues/2474)) ([b4309df](https://github.com/amalgamated-tools/biblioteka/commit/b4309dfa3b8382f5983e8b8e8b5b45caf5d41bab))
* add (reading_list_id, added_at) index on reading_list_books to eliminate ORDER BY filesort ([#2499](https://github.com/amalgamated-tools/biblioteka/issues/2499)) ([651885d](https://github.com/amalgamated-tools/biblioteka/commit/651885d0f292379840ee30854a0f40623236e33b))
* add (series_id, position) index on book_series to eliminate ORDER BY filesort ([#2555](https://github.com/amalgamated-tools/biblioteka/issues/2555)) ([de523d9](https://github.com/amalgamated-tools/biblioteka/commit/de523d9b231f3a9912dd1421bee264efac74c9b4))
* Add BenchmarkGetPendingAIEnrichmentByBook ([#2780](https://github.com/amalgamated-tools/biblioteka/issues/2780)) ([ee9fa4f](https://github.com/amalgamated-tools/biblioteka/commit/ee9fa4fff065dad0e845c09bbeb0ea69e585244c))
* Add BenchmarkGetPendingGoodreadsMetadataByBook ([#2813](https://github.com/amalgamated-tools/biblioteka/issues/2813)) ([47301de](https://github.com/amalgamated-tools/biblioteka/commit/47301de858ce78af328e6966b2f0358e4adcf532))
* Add BenchmarkGetRecommendations baseline ([#2818](https://github.com/amalgamated-tools/biblioteka/issues/2818)) ([66da786](https://github.com/amalgamated-tools/biblioteka/commit/66da786119729b6e9b8cd6326d2465025b7e1ce6))
* Add BenchmarkListBooksModifiedSince benchmarks ([#2905](https://github.com/amalgamated-tools/biblioteka/issues/2905)) ([35977f4](https://github.com/amalgamated-tools/biblioteka/commit/35977f46be45de38c6a34c6e4b49f6ab0877746f))
* add benchmarks for ListAuditLogs, ListGroupMembers, and ListAnnotationsForBook ([#2451](https://github.com/amalgamated-tools/biblioteka/issues/2451)) ([9a73b07](https://github.com/amalgamated-tools/biblioteka/commit/9a73b073cf71ff8efd2e74cabc92e7689b0c0a62))
* add benchmarks for ListReadingLists and ListReadingListBooks ([#2616](https://github.com/amalgamated-tools/biblioteka/issues/2616)) ([8979a94](https://github.com/amalgamated-tools/biblioteka/commit/8979a945a52a679f7bf8fe0460eebebbbf9558b6))
* add composite sort indexes for reading_group_members and audit_logs ([#2423](https://github.com/amalgamated-tools/biblioteka/issues/2423)) ([1eb1139](https://github.com/amalgamated-tools/biblioteka/commit/1eb11396f53a60d67c145ca893333055f91cce30))
* add temp_store=MEMORY and cache_size PRAGMAs to SQLite setup ([#2673](https://github.com/amalgamated-tools/biblioteka/issues/2673)) ([da27a42](https://github.com/amalgamated-tools/biblioteka/commit/da27a4253dfc2af9dbfc9f52ca1c4d7f55109412))
* extend books created_at index to include id tiebreaker ([#2642](https://github.com/amalgamated-tools/biblioteka/issues/2642)) ([26ced81](https://github.com/amalgamated-tools/biblioteka/commit/26ced819979bf6a236f1793215db953af82d8b43))
* parallelize batch DB queries in Kobo sync and OPDS ([#2576](https://github.com/amalgamated-tools/biblioteka/issues/2576)) ([06a3673](https://github.com/amalgamated-tools/biblioteka/commit/06a3673bd32e540478810484debf3b45b54b6977))
* rewrite reading list queries to use correlated subquery for book count ([#2533](https://github.com/amalgamated-tools/biblioteka/issues/2533)) ([cc48c90](https://github.com/amalgamated-tools/biblioteka/commit/cc48c904b3a628bedb8ef6ed23d63619aad941e0))
* Update GitHub Actions versions - 2026-04-20 ([#2408](https://github.com/amalgamated-tools/biblioteka/issues/2408)) ([bdf13f9](https://github.com/amalgamated-tools/biblioteka/commit/bdf13f95a572732fcace8f749c42cbcd32015c59))
* Update GitHub Actions versions - 2026-04-22 ([#2466](https://github.com/amalgamated-tools/biblioteka/issues/2466)) ([cf7d6b6](https://github.com/amalgamated-tools/biblioteka/commit/cf7d6b6f3292bfc14c22203afee6ec192d792307))
* Update GitHub Actions versions - 2026-04-27 ([#2599](https://github.com/amalgamated-tools/biblioteka/issues/2599)) ([d7539b8](https://github.com/amalgamated-tools/biblioteka/commit/d7539b8d6c655cca7f7e6fefcf4cc021288b6e59))
* Update GitHub Actions versions - 2026-04-29 ([#2656](https://github.com/amalgamated-tools/biblioteka/issues/2656)) ([5432844](https://github.com/amalgamated-tools/biblioteka/commit/5432844df9928f241d58116815f9b6ba891f322a))
* Update GitHub Actions versions - 2026-05-01 ([#2725](https://github.com/amalgamated-tools/biblioteka/issues/2725)) ([a22f7a0](https://github.com/amalgamated-tools/biblioteka/commit/a22f7a0babed04cc35f906307939f3a17a2decfb))
* use LOWER(name) sort for authors and series list queries ([#2702](https://github.com/amalgamated-tools/biblioteka/issues/2702)) ([75c785c](https://github.com/amalgamated-tools/biblioteka/commit/75c785c4d6d51b439d6c4d7988b432ef2c8e1f28))

## [0.16.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.15.0...v0.16.0) (2026-04-20)


### Features

* **annotations:** HTTP handler and frontend API for book annotations ([#2213](https://github.com/amalgamated-tools/biblioteka/issues/2213)) ([9b19b94](https://github.com/amalgamated-tools/biblioteka/commit/9b19b947357a8e07ac82eec96e7403c3f9f5a1ac))
* **auth:** support INITIAL_ADMIN_EMAIL and INITIAL_ADMIN_PASSWORD bootstrap env vars ([#2321](https://github.com/amalgamated-tools/biblioteka/issues/2321)) ([458e511](https://github.com/amalgamated-tools/biblioteka/commit/458e511245396ef9b81dfea1c69514a0561f8c88))
* **books:** embed tags in bookDTO and TypeScript Book interface ([#2325](https://github.com/amalgamated-tools/biblioteka/issues/2325)) ([dd15bdc](https://github.com/amalgamated-tools/biblioteka/commit/dd15bdc9c6cf5060c57381425e39e89f15b1f0a4))
* **frontend:** add AI enrichment API client functions and TypeScript types ([#2366](https://github.com/amalgamated-tools/biblioteka/issues/2366)) ([13a1fdb](https://github.com/amalgamated-tools/biblioteka/commit/13a1fdb57c48a2fe2cc72d5cc2839eca04802ec7))
* **frontend:** add LLM config TypeScript types, API client, and admin settings tab ([#2331](https://github.com/amalgamated-tools/biblioteka/issues/2331)) ([68260bd](https://github.com/amalgamated-tools/biblioteka/commit/68260bd9d88a5834a3daf77842bad59cf019c352))
* **frontend:** add ReadingGroup TypeScript types and API client module ([#2198](https://github.com/amalgamated-tools/biblioteka/issues/2198)) ([1deff97](https://github.com/amalgamated-tools/biblioteka/commit/1deff97c63e316543ea8e1b05e7d281581c71a50))
* **frontend:** tag management UI — display, edit, and manage tags ([#2232](https://github.com/amalgamated-tools/biblioteka/issues/2232)) ([776361f](https://github.com/amalgamated-tools/biblioteka/commit/776361f984e6e6d674fd856cc6e6e0d58f30720d))
* **frontend:** tag management UI — display, edit, and manage tags ([#2317](https://github.com/amalgamated-tools/biblioteka/issues/2317)) ([79f808c](https://github.com/amalgamated-tools/biblioteka/commit/79f808c477391e0766541e83c1de5532591e1179))
* **groups:** add Reading Groups UI ([#2211](https://github.com/amalgamated-tools/biblioteka/issues/2211)) ([393dcba](https://github.com/amalgamated-tools/biblioteka/commit/393dcba167bdb9c3e704504ab436aebc33bd41e4))
* **metadata:** add AI enrichment TypeScript type and frontend API client ([#2324](https://github.com/amalgamated-tools/biblioteka/issues/2324)) ([708981c](https://github.com/amalgamated-tools/biblioteka/commit/708981c679ed2c8155ca77f1b2fb5445787df72d))
* **metadata:** wire "Apply All" to server-side atomic apply endpoint ([#2330](https://github.com/amalgamated-tools/biblioteka/issues/2330)) ([5433ef5](https://github.com/amalgamated-tools/biblioteka/commit/5433ef53522e10f8fa922301e02a7f28b811989f))
* **settings:** add Rebuild Search Index button in admin settings ([#2208](https://github.com/amalgamated-tools/biblioteka/issues/2208)) ([1f28af0](https://github.com/amalgamated-tools/biblioteka/commit/1f28af0b187d4b1ea66b220617d9a0bb3f0b9b9c))


### Bug Fixes

* **accessibility:** correct listbox option semantics in BookTagsEditor ([#2295](https://github.com/amalgamated-tools/biblioteka/issues/2295)) ([c88bf02](https://github.com/amalgamated-tools/biblioteka/commit/c88bf0243c0337bb94152e324a676da2dee89595))
* **accessibility:** keep metadata fetch live region mounted for initial progress announcement ([#2292](https://github.com/amalgamated-tools/biblioteka/issues/2292)) ([8165206](https://github.com/amalgamated-tools/biblioteka/commit/81652066a3244e5fbc89d8828c2de07df5bc7aed))
* **Auth.svelte:** remove redundant console.error calls for OIDC/signup status checks ([#2309](https://github.com/amalgamated-tools/biblioteka/issues/2309)) ([dae1381](https://github.com/amalgamated-tools/biblioteka/commit/dae1381990f0ceb66028f98e0ab66186a316a389))
* **Auth.svelte:** replace promise chains with async/await and Promise.all ([#2370](https://github.com/amalgamated-tools/biblioteka/issues/2370)) ([70cc270](https://github.com/amalgamated-tools/biblioteka/commit/70cc2700036363e1b234e572bd5f92c0312f4d9d))
* **auth:** scope signup aria-invalid to the actual invalid field ([#2288](https://github.com/amalgamated-tools/biblioteka/issues/2288)) ([eea1ac3](https://github.com/amalgamated-tools/biblioteka/commit/eea1ac3e90dac50f004616f243b05c193cbea255))
* **calibre:** resolve ambiguous double-wrap error chain in web_import.go ([#2303](https://github.com/amalgamated-tools/biblioteka/issues/2303)) ([9c1e624](https://github.com/amalgamated-tools/biblioteka/commit/9c1e62451fcc108087b4a8fe7350e6584b19ecc4))
* **ci:** update Daily Workflow Updater to handle missing gh aw extension ([#2376](https://github.com/amalgamated-tools/biblioteka/issues/2376)) ([d692c35](https://github.com/amalgamated-tools/biblioteka/commit/d692c354bbb701acf377feea86912432a5d9b03f))
* **db:** add ErrInvalidLibraryName sentinel and name normalization to libraries ([#2266](https://github.com/amalgamated-tools/biblioteka/issues/2266)) ([983e989](https://github.com/amalgamated-tools/biblioteka/commit/983e989602699acbf366d356f9ac49d3204bad92))
* **db:** add ErrInvalidLibraryName sentinel and name normalization to libraries ([#2365](https://github.com/amalgamated-tools/biblioteka/issues/2365)) ([f7cb9b1](https://github.com/amalgamated-tools/biblioteka/commit/f7cb9b1c6fc8bcebf288b12c0da371dd92c3aa1c))
* **db:** lowercase error strings in Timestamp.Scan ([#2302](https://github.com/amalgamated-tools/biblioteka/issues/2302)) ([cb60d31](https://github.com/amalgamated-tools/biblioteka/commit/cb60d31d35a5f7d17df9c5bd28a23feae03d62d7))
* **db:** make AddGroupMember owner check and insert atomic ([#2320](https://github.com/amalgamated-tools/biblioteka/issues/2320)) ([8541099](https://github.com/amalgamated-tools/biblioteka/commit/8541099ce724a73ab8068740ec251aee0fc60f35))
* **db:** remove "db: " prefix from sentinel errors in ai_enrichments.go ([#2301](https://github.com/amalgamated-tools/biblioteka/issues/2301)) ([a42150c](https://github.com/amalgamated-tools/biblioteka/commit/a42150c8deb428ab67e58377059c4f1fb95c3fea))
* **db:** standardize ErrInvalidAuthorName message ([#2300](https://github.com/amalgamated-tools/biblioteka/issues/2300)) ([fe10c08](https://github.com/amalgamated-tools/biblioteka/commit/fe10c084da1121ad8709d4dd3b659400ba539d64))
* **db:** standardize sentinel error message wording in internal/db ([#2313](https://github.com/amalgamated-tools/biblioteka/issues/2313)) ([c7091fd](https://github.com/amalgamated-tools/biblioteka/commit/c7091fddbd5870e3c6ca40ab0ff990c245296764))
* **frontend:** remove redundant console.error calls in Recommendations and Auth components ([#2199](https://github.com/amalgamated-tools/biblioteka/issues/2199)) ([a547dc4](https://github.com/amalgamated-tools/biblioteka/commit/a547dc4ccd0280f3813a757a7552f9135d8064cc))
* **handlers:** add empty-ID guard to APIKeyHandler.Create logAudit call ([#2194](https://github.com/amalgamated-tools/biblioteka/issues/2194)) ([9b01227](https://github.com/amalgamated-tools/biblioteka/commit/9b012274b6ae81d6b35109465a114594e994162b))
* **handlers:** record group name in deleteGroup audit log ([#2319](https://github.com/amalgamated-tools/biblioteka/issues/2319)) ([ba9c4ae](https://github.com/amalgamated-tools/biblioteka/commit/ba9c4ae20a9bc1b1e78c8675bf3655e191697120))
* **migrations:** standardize SQLite timestamp format to ISO 8601 in reading_groups era migrations ([#2306](https://github.com/amalgamated-tools/biblioteka/issues/2306)) ([992447f](https://github.com/amalgamated-tools/biblioteka/commit/992447fa501fbb906d0e6423e0fc78199031153f))
* **Recommendations.svelte:** replace promise chain with async/await and AbortController ([#2369](https://github.com/amalgamated-tools/biblioteka/issues/2369)) ([ff539d9](https://github.com/amalgamated-tools/biblioteka/commit/ff539d92e569c56b14b8f8ebf26ec1e49add9e96))
* **repo assist:** fix(migrations): standardize SQLite timestamps to ISO 8601 in reading-groups era ([#2348](https://github.com/amalgamated-tools/biblioteka/issues/2348)) ([adc55af](https://github.com/amalgamated-tools/biblioteka/commit/adc55af312a6d06b153e96169d1b5fe57d476e0e))
* **repo assist:** perf(db): run GetYearInBooks queries concurrently via errgroup ([#2225](https://github.com/amalgamated-tools/biblioteka/issues/2225)) ([92d00ff](https://github.com/amalgamated-tools/biblioteka/commit/92d00ff539d21340db254a3d545dd861babc4cf6))
* **screenshots:** expand coverage to all routes and UI states ([#2322](https://github.com/amalgamated-tools/biblioteka/issues/2322)) ([ca3714e](https://github.com/amalgamated-tools/biblioteka/commit/ca3714ec17db6dd19c29cfef84dd57ce3c57e894))
* **security:** add SSRF validation to admin-configurable LLM endpoint URL ([#2367](https://github.com/amalgamated-tools/biblioteka/issues/2367)) ([6d8d7e8](https://github.com/amalgamated-tools/biblioteka/commit/6d8d7e8b1245caa7b340a1ca4c3316c484a0121d))
* **security:** Add SSRF validation to LLM Ollama endpoint URL ([#2311](https://github.com/amalgamated-tools/biblioteka/issues/2311)) ([fc9be90](https://github.com/amalgamated-tools/biblioteka/commit/fc9be90720cf973db5d5e97711c0b8c781fe05e3))
* **security:** reject non-empty JWT_SECRET shorter than MinSecretLength at startup ([#2315](https://github.com/amalgamated-tools/biblioteka/issues/2315)) ([7b0a4e9](https://github.com/amalgamated-tools/biblioteka/commit/7b0a4e9f007fe1c9fd2ce719c1a4214630b19ef5))
* **sidebar:** raise library settings icon resting-state contrast ([#2294](https://github.com/amalgamated-tools/biblioteka/issues/2294)) ([c8055ba](https://github.com/amalgamated-tools/biblioteka/commit/c8055ba2862631a64aab485d5accd31069561792))
* **smtp:** inject time into BuildAttachmentMessage for deterministic output ([#2267](https://github.com/amalgamated-tools/biblioteka/issues/2267)) ([91d05c2](https://github.com/amalgamated-tools/biblioteka/commit/91d05c211a23ced8bd7904e8628df2251b2923d7))
* **smtp:** inject time parameter into BuildAttachmentMessage for deterministic testing ([#2368](https://github.com/amalgamated-tools/biblioteka/issues/2368)) ([4a81b42](https://github.com/amalgamated-tools/biblioteka/commit/4a81b42e2bd116f07ff2bdffbe2dae49a8df539b))
* **swagger:** correct OpenAPI annotations from review feedback ([#2327](https://github.com/amalgamated-tools/biblioteka/issues/2327)) ([cfc0924](https://github.com/amalgamated-tools/biblioteka/commit/cfc0924ed8f40740fca278b684742fb5899fbf2b))
* **tags:** associate rename errors with rename input and add status announcement ([#2290](https://github.com/amalgamated-tools/biblioteka/issues/2290)) ([858fe2a](https://github.com/amalgamated-tools/biblioteka/commit/858fe2aec263e6972fcc4ebb9c060436fc6e6f3f))
* **types:** correct ReadingProgressItem nullable fields to optional-only ([#2196](https://github.com/amalgamated-tools/biblioteka/issues/2196)) ([01506d6](https://github.com/amalgamated-tools/biblioteka/commit/01506d69a6cc01fb942d50724bf275deccd751bc))
* **UsersTab:** replace fragile $state.raw init with clean $props + $effect ([#2307](https://github.com/amalgamated-tools/biblioteka/issues/2307)) ([d79f330](https://github.com/amalgamated-tools/biblioteka/commit/d79f33062b79eca23918d433da179d7170fb5d51))
* **workflows:** enforce Conventional Commits format on agentic workflow issue/PR title prefixes ([#2381](https://github.com/amalgamated-tools/biblioteka/issues/2381)) ([0d3a51d](https://github.com/amalgamated-tools/biblioteka/commit/0d3a51d9d8da709f2343eac6f1771cfcfbcefd1c))


### Performance Improvements

* add composite sort indexes for api_keys, kobo_tokens, passkey_credentials ([#2361](https://github.com/amalgamated-tools/biblioteka/issues/2361)) ([85cf002](https://github.com/amalgamated-tools/biblioteka/commit/85cf0026fe586052d0764af81b52868705385abf))
* add sort index on tags.name for ORDER BY optimization ([#2275](https://github.com/amalgamated-tools/biblioteka/issues/2275)) ([814fe5d](https://github.com/amalgamated-tools/biblioteka/commit/814fe5d311c27aaba2785b2c14358f6b084b9c3d))

## [0.15.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.14.0...v0.15.0) (2026-04-18)


### Features

* update network permissions in Dependabot workflow to allow additional domains ([a06cdd3](https://github.com/amalgamated-tools/biblioteka/commit/a06cdd3dd6b88dc74262de1697eec3e453b4d578))
* use goauth ([#2101](https://github.com/amalgamated-tools/biblioteka/issues/2101)) ([c4d9b34](https://github.com/amalgamated-tools/biblioteka/commit/c4d9b34d98242052bde79cb998c26a5e68153e79))


### Bug Fixes

* **accessibility:** add required-field semantics to reading list create/edit forms ([#2183](https://github.com/amalgamated-tools/biblioteka/issues/2183)) ([bb2e126](https://github.com/amalgamated-tools/biblioteka/commit/bb2e1262765bb2450dc9157b780fcca98c803657))
* **accessibility:** add visible invalid-state styling to TextInput ([#2124](https://github.com/amalgamated-tools/biblioteka/issues/2124)) ([16e0a5d](https://github.com/amalgamated-tools/biblioteka/commit/16e0a5daebf1eb32cca5f32e36193d82ff7e8001))
* **accessibility:** announce BookList page position via live region ([#2129](https://github.com/amalgamated-tools/biblioteka/issues/2129)) ([c338553](https://github.com/amalgamated-tools/biblioteka/commit/c338553b3e871582b9aa5b5ed769aae256f15230))
* **accessibility:** expose BookList pagination as a named navigation landmark ([#2185](https://github.com/amalgamated-tools/biblioteka/issues/2185)) ([c03cfd6](https://github.com/amalgamated-tools/biblioteka/commit/c03cfd642f9303ad475ceabbf4dc4b6abc42823c))
* **accessibility:** focus Email Book modal container on open ([#2128](https://github.com/amalgamated-tools/biblioteka/issues/2128)) ([e5cba3d](https://github.com/amalgamated-tools/biblioteka/commit/e5cba3d335b18bc83008d58d3bbb20edfd97d456))
* **accessibility:** mark Auth required fields and add required-field guidance ([#2179](https://github.com/amalgamated-tools/biblioteka/issues/2179)) ([4f7c156](https://github.com/amalgamated-tools/biblioteka/commit/4f7c1562977a0cbf81b1c58833e5d18f3ee9627a))
* **accessibility:** raise low-contrast ink-400 text usages to AA-compliant tokens ([#2186](https://github.com/amalgamated-tools/biblioteka/issues/2186)) ([b474584](https://github.com/amalgamated-tools/biblioteka/commit/b474584bb85f80e15c046488982ec332a11e56b7))
* add otelkeys.Months, fix stats.go log key, fmt.Errorf in config_watch_folder, remove dead validate tag ([#2109](https://github.com/amalgamated-tools/biblioteka/issues/2109)) ([16c9ae4](https://github.com/amalgamated-tools/biblioteka/commit/16c9ae455bccbc392c4e6c6c10175a28402e5a98))
* **auth:** add required-field indicators to login and sign-up forms ([#2126](https://github.com/amalgamated-tools/biblioteka/issues/2126)) ([534b4c3](https://github.com/amalgamated-tools/biblioteka/commit/534b4c3bcb852a9e4bd561c6ba15d4092d4b97ad))
* **auth:** scope login aria-invalid to the field in error ([#2181](https://github.com/amalgamated-tools/biblioteka/issues/2181)) ([788dfdf](https://github.com/amalgamated-tools/biblioteka/commit/788dfdff7a65e0fe70fe9e762e624ad8af706d64))
* **repo assist:** test(db): add tests for ai_enrichments.go ([#2150](https://github.com/amalgamated-tools/biblioteka/issues/2150)) ([057f590](https://github.com/amalgamated-tools/biblioteka/commit/057f590f812c1cf3aac8649bd031e906642dcb0c))


### Performance Improvements

* Add missing FK cascade indexes on reading_group_lists and passkey_challenges ([#2107](https://github.com/amalgamated-tools/biblioteka/issues/2107)) ([0dff8cc](https://github.com/amalgamated-tools/biblioteka/commit/0dff8ccb0ced8d634877b4201f25795e659dced0))
* Add missing FK indexes on kobo_reading_states, ai_enrichments, goodreads_metadata, book_annotations ([#2168](https://github.com/amalgamated-tools/biblioteka/issues/2168)) ([a30b33a](https://github.com/amalgamated-tools/biblioteka/commit/a30b33aa0050c53a8fb285a58288759d1ca055ed))

## [0.14.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.13.0...v0.14.0) (2026-04-16)


### Features

* **frontend:** show cover image preview on book edit page ([#2061](https://github.com/amalgamated-tools/biblioteka/issues/2061)) ([a0e3f7e](https://github.com/amalgamated-tools/biblioteka/commit/a0e3f7e31b0b0e658bf120e8086fcf4ae32eb1bf))


### Bug Fixes

* **accessibility:** add visible label for passkey name input in Account settings ([#2075](https://github.com/amalgamated-tools/biblioteka/issues/2075)) ([5ffbc8c](https://github.com/amalgamated-tools/biblioteka/commit/5ffbc8c631022de2f808246183a925650b5a9675))
* **accessibility:** prevent duplicate screen reader announcements in BookCard links ([#2072](https://github.com/amalgamated-tools/biblioteka/issues/2072)) ([9367248](https://github.com/amalgamated-tools/biblioteka/commit/9367248f3fca4a90a14cf23f5f3d183b12195a1c))
* **accessibility:** raise light-mode helper text contrast from ink-400 to ink-500 ([#2070](https://github.com/amalgamated-tools/biblioteka/issues/2070)) ([91dbab9](https://github.com/amalgamated-tools/biblioteka/commit/91dbab950c9b3a28f1b2ed605d7dfb53267b1283))
* **ci:** stop false `[aw] Contribution Check failed` issue creation ([#2039](https://github.com/amalgamated-tools/biblioteka/issues/2039)) ([e2a13f2](https://github.com/amalgamated-tools/biblioteka/commit/e2a13f25c9086799d583351da3f32f789c9a64d4))
* **frontend:** replace outline-none with focus-visible:outline-none across form inputs ([#2060](https://github.com/amalgamated-tools/biblioteka/issues/2060)) ([1bbf63a](https://github.com/amalgamated-tools/biblioteka/commit/1bbf63ab16e9771f40f466225ba0523669bb0c55))
* **frontend:** surface dashboard fetch failures via AlertBanner and onMount loaders ([#2034](https://github.com/amalgamated-tools/biblioteka/issues/2034)) ([46737f1](https://github.com/amalgamated-tools/biblioteka/commit/46737f1b574f5718b5281c93da89022db4847633))
* **passkeys:** convert base64url strings to ArrayBuffer for WebAuthn API ([#2062](https://github.com/amalgamated-tools/biblioteka/issues/2062)) ([be2c1d3](https://github.com/amalgamated-tools/biblioteka/commit/be2c1d3b7b9071cb9e2daed425c4b3afb1bed28d))
* **repo assist:** add tags API client, Tag types, and getBookTags/setBookTags ([#2076](https://github.com/amalgamated-tools/biblioteka/issues/2076)) ([b193014](https://github.com/amalgamated-tools/biblioteka/commit/b193014465d10e9d7992b53aab5e141bdbc33c70))
* **ui:** add missing padding and alignment to Button component ([#2057](https://github.com/amalgamated-tools/biblioteka/issues/2057)) ([ac87ef0](https://github.com/amalgamated-tools/biblioteka/commit/ac87ef0455d0b19bb0b7483e8aea60d3fcd4eb07))


### Performance Improvements

* Add missing FK indexes: book_tags.tag_id and goodreads_metadata user+book+status ([#2051](https://github.com/amalgamated-tools/biblioteka/issues/2051)) ([f723789](https://github.com/amalgamated-tools/biblioteka/commit/f72378990cb5d08bc01fb09d4a0c03cc4cf18d77))

## [0.13.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.12.0...v0.13.0) (2026-04-15)


### Features

* **auth:** add WebAuthn passkey support ([#1966](https://github.com/amalgamated-tools/biblioteka/issues/1966)) ([6150e5d](https://github.com/amalgamated-tools/biblioteka/commit/6150e5d90ed7e10f6f3d2026eb8232e07c449d6f))
* **calibre:** web UI Calibre import wizard ([#1960](https://github.com/amalgamated-tools/biblioteka/issues/1960)) ([b905c7a](https://github.com/amalgamated-tools/biblioteka/commit/b905c7a3cf1d942987a5680dc03968c19e95df0a))
* **daily-doc-updater:** add file-based deduplication guard ([#1932](https://github.com/amalgamated-tools/biblioteka/issues/1932)) ([892ccde](https://github.com/amalgamated-tools/biblioteka/commit/892ccde570b57c83072c26911ebe53995a73c07d))
* **db:** add PostgreSQL integration tests for SearchBooks ([#1908](https://github.com/amalgamated-tools/biblioteka/issues/1908)) ([7018c5a](https://github.com/amalgamated-tools/biblioteka/commit/7018c5a2540670a9d1d8d904393a194212e41697))
* **db:** converge multi-word search semantics across SQLite and PostgreSQL ([#1906](https://github.com/amalgamated-tools/biblioteka/issues/1906)) ([900f7c4](https://github.com/amalgamated-tools/biblioteka/commit/900f7c44bbf97037cb56941d2e27ddcc1044489a))
* **frontend:** add skippable first-library onboarding wizard ([#2012](https://github.com/amalgamated-tools/biblioteka/issues/2012)) ([8324bda](https://github.com/amalgamated-tools/biblioteka/commit/8324bda3a153159c5bb1d5ad64523782a8545ae1))
* **frontend:** wire up book search UI with debouncing and URL persistence ([#1965](https://github.com/amalgamated-tools/biblioteka/issues/1965)) ([ebf0ce9](https://github.com/amalgamated-tools/biblioteka/commit/ebf0ce93edfa18b7415af8434b9389648e164663))
* **groups:** shared reading groups foundation — migrations, DB layer, group handler ([#1963](https://github.com/amalgamated-tools/biblioteka/issues/1963)) ([3d94cd6](https://github.com/amalgamated-tools/biblioteka/commit/3d94cd68c7ff747e741dc46e2d280585b2233862))
* **makefile:** add frontend formatting command to Makefile ([a1087cd](https://github.com/amalgamated-tools/biblioteka/commit/a1087cdad59f6a778b25b1446883ee2510e964a0))
* provider-agnostic AI metadata enrichment with tags system and Ollama support ([#1974](https://github.com/amalgamated-tools/biblioteka/issues/1974)) ([74bee8a](https://github.com/amalgamated-tools/biblioteka/commit/74bee8a73705c0514036a4c21fa7fa0edad9df5c))
* **recommendations:** add local book recommender endpoint and UI ([#1973](https://github.com/amalgamated-tools/biblioteka/issues/1973)) ([df80064](https://github.com/amalgamated-tools/biblioteka/commit/df800645d8380eafbfd9bd4b9569c80deddf6bee))
* **search:** FTS5 index health check, startup auto-rebuild, and admin reindex endpoint ([#1909](https://github.com/amalgamated-tools/biblioteka/issues/1909)) ([a6cb1ae](https://github.com/amalgamated-tools/biblioteka/commit/a6cb1aee0c024bf3731f404a1c0192d02372e499))
* **stats:** add Year in Books analytics endpoint and dashboard card ([#1946](https://github.com/amalgamated-tools/biblioteka/issues/1946)) ([7065e51](https://github.com/amalgamated-tools/biblioteka/commit/7065e51fe7ee651e39b679de4a14feb846b31d32))
* **storage:** define Storage interface and LocalStorage filesystem implementation ([#1920](https://github.com/amalgamated-tools/biblioteka/issues/1920)) ([d2b7b7c](https://github.com/amalgamated-tools/biblioteka/commit/d2b7b7c36785a13057a73bf3d695c8367cd64391))
* **workflows:** throttle daily-doc-updater with open-PR gate and lower per-run cap ([#1931](https://github.com/amalgamated-tools/biblioteka/issues/1931)) ([0092e8b](https://github.com/amalgamated-tools/biblioteka/commit/0092e8b7f8a9076eb6a1b88438f74b6f006e7835))


### Bug Fixes

* **accessibility:** associate validation errors to invalid form controls ([#2003](https://github.com/amalgamated-tools/biblioteka/issues/2003)) ([d479f59](https://github.com/amalgamated-tools/biblioteka/commit/d479f59bc266904c72be0320bc79502fd0ed65ff))
* **accessibility:** expose mobile menu state and remove closed sidebar from AT tree ([#2001](https://github.com/amalgamated-tools/biblioteka/issues/2001)) ([16c6f50](https://github.com/amalgamated-tools/biblioteka/commit/16c6f50833c9ab44c8251a627c1c359f673af524))
* **accessibility:** make BookDetail file download links uniquely descriptive to assistive tech ([#2004](https://github.com/amalgamated-tools/biblioteka/issues/2004)) ([6934553](https://github.com/amalgamated-tools/biblioteka/commit/69345536e58c453ee89d0e2b17b8ac7e94a18f4e))
* **accessibility:** replace DownloadsHistogram listitem tabindex pattern with semantic data table ([#1905](https://github.com/amalgamated-tools/biblioteka/issues/1905)) ([1f73e66](https://github.com/amalgamated-tools/biblioteka/commit/1f73e660a0530d4959829baa23e15a464ca4fb13))
* **accessibility:** restore focus when ReadingListDetail delete confirmation mounts ([#1903](https://github.com/amalgamated-tools/biblioteka/issues/1903)) ([1aa9dfd](https://github.com/amalgamated-tools/biblioteka/commit/1aa9dfd5d24d9ffbeb933aa7e295b56a44350260))
* **accessibility:** restore native link semantics in BookList table view ([#1907](https://github.com/amalgamated-tools/biblioteka/issues/1907)) ([f847ce7](https://github.com/amalgamated-tools/biblioteka/commit/f847ce72f91f3c5ec5a9441ca5d796605a0e7e0b))
* **accessibility:** use semantic table structure in MetadataComparison ([#1901](https://github.com/amalgamated-tools/biblioteka/issues/1901)) ([4c4696c](https://github.com/amalgamated-tools/biblioteka/commit/4c4696c5ecb3ddc98dc97b8f6c0dd203f4158f2b))
* **ai:** address PR [#1974](https://github.com/amalgamated-tools/biblioteka/issues/1974) review comments ([#2011](https://github.com/amalgamated-tools/biblioteka/issues/2011)) ([a90afcd](https://github.com/amalgamated-tools/biblioteka/commit/a90afcdd45ec4df509fdb627490dc588e8575df2))
* **calibre:** downgrade tags advisory log from Info to Debug ([#1962](https://github.com/amalgamated-tools/biblioteka/issues/1962)) ([f0ea21e](https://github.com/amalgamated-tools/biblioteka/commit/f0ea21e68b6f008ebe4410001456c8ecf11eb6ce))
* **frontend:** raise dashboard and histogram label contrast to meet WCAG 1.4.3 ([#2006](https://github.com/amalgamated-tools/biblioteka/issues/2006)) ([9674d65](https://github.com/amalgamated-tools/biblioteka/commit/9674d65bd44797007e88ca66a5c23c164c184289))
* **goodreads:** use http.MethodGet constant instead of string literal ([#1928](https://github.com/amalgamated-tools/biblioteka/issues/1928)) ([1cc05f4](https://github.com/amalgamated-tools/biblioteka/commit/1cc05f401c9d66bc2369eb14e0778cfdacc6b362))
* remove double-logging in goodreads, fix otelkeys.Count→Limit, wire search query ([#1951](https://github.com/amalgamated-tools/biblioteka/issues/1951)) ([0853827](https://github.com/amalgamated-tools/biblioteka/commit/08538270460caa9aa1bfbb56e0265059398bde17))
* **repo assist:** refactor(frontend): move AuthResponse and MetadataFetchResponse to types.ts ([#2007](https://github.com/amalgamated-tools/biblioteka/issues/2007)) ([c9368e3](https://github.com/amalgamated-tools/biblioteka/commit/c9368e3342dcedd767824373efb74ddb3531f7e1))
* **stats:** use otelkeys.Limit instead of otelkeys.Count for monthly download window ([#1964](https://github.com/amalgamated-tools/biblioteka/issues/1964)) ([7430886](https://github.com/amalgamated-tools/biblioteka/commit/743088626ec12fbb98d8a1b59b02fe59fc3074d1))


### Performance Improvements

* **handlers:** eliminate extra GetReadingStats DB query in reading progress stats ([#1912](https://github.com/amalgamated-tools/biblioteka/issues/1912)) ([13f8f93](https://github.com/amalgamated-tools/biblioteka/commit/13f8f939afab9053f2cee1202b05d44b57ade157))
* parallelize LoadBookRelations queries with errgroup ([#1982](https://github.com/amalgamated-tools/biblioteka/issues/1982)) ([8f25abe](https://github.com/amalgamated-tools/biblioteka/commit/8f25abea17910ee45817cd3be208cf94024f6537))

## [0.12.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.11.0...v0.12.0) (2026-04-14)


### Features

* **frontend:** wire up book search UI in BookList and Books components ([#1895](https://github.com/amalgamated-tools/biblioteka/issues/1895)) ([e0dfccf](https://github.com/amalgamated-tools/biblioteka/commit/e0dfccf27271b1325a7ddd09f74acce1d6d54f20))

## [0.11.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.10.0...v0.11.0) (2026-04-14)


### Features

* **calibre:** add Calibre library importer CLI command ([#1795](https://github.com/amalgamated-tools/biblioteka/issues/1795)) ([2302cfd](https://github.com/amalgamated-tools/biblioteka/commit/2302cfdb8da72543cb867afd4abed9d995c2e5a5))
* **dashboard:** add reading progress dashboard with KOSync streak and in-progress tracking ([#1799](https://github.com/amalgamated-tools/biblioteka/issues/1799)) ([5e17867](https://github.com/amalgamated-tools/biblioteka/commit/5e1786793d55eaaa46bc6f9f9479eed40654c2aa))
* **reading-lists:** implement user-curated reading lists & shelves ([#1801](https://github.com/amalgamated-tools/biblioteka/issues/1801)) ([7a5c2bc](https://github.com/amalgamated-tools/biblioteka/commit/7a5c2bce32b147a21b0f39cc8ad12efa22faf606))
* **security:** encrypt SMTP password and OIDC client secret at rest ([#1862](https://github.com/amalgamated-tools/biblioteka/issues/1862)) ([46cbb24](https://github.com/amalgamated-tools/biblioteka/commit/46cbb2438412149ce2563645a6143485f7289fa3))
* **stats:** add downloads-per-month histogram to dashboard ([#1794](https://github.com/amalgamated-tools/biblioteka/issues/1794)) ([c82e2a7](https://github.com/amalgamated-tools/biblioteka/commit/c82e2a7daf2b7fa85169af24d85b9ecadae64ad6))


### Bug Fixes

* **accessibility:** fix WCAG 1.4.3 contrast violations in DownloadsHistogram ([#1818](https://github.com/amalgamated-tools/biblioteka/issues/1818)) ([a80bf07](https://github.com/amalgamated-tools/biblioteka/commit/a80bf0794a57345cbe5f062c434f8eee35759d5f))
* **accessibility:** focus close button on EmailBookModal open; scope keyboard handler to dialog ([#1822](https://github.com/amalgamated-tools/biblioteka/issues/1822)) ([e8991b1](https://github.com/amalgamated-tools/biblioteka/commit/e8991b1df81c90ace673789cd19e32e544e6f37c))
* **accessibility:** high-contrast keyboard focus indicator for DownloadsHistogram bars ([#1823](https://github.com/amalgamated-tools/biblioteka/issues/1823)) ([b9cb897](https://github.com/amalgamated-tools/biblioteka/commit/b9cb8970be085ee6536cae85b17e4bf104a41be5))
* **accessibility:** make BookList table rows keyboard accessible (WCAG 2.1.1) ([#1763](https://github.com/amalgamated-tools/biblioteka/issues/1763)) ([732502f](https://github.com/amalgamated-tools/biblioteka/commit/732502fa0febe37f90a5ee492101b4f9e4e30928))
* **accessibility:** remove redundant aria-label on DownloadsHistogram heading, use aria-labelledby on list ([#1825](https://github.com/amalgamated-tools/biblioteka/issues/1825)) ([43304e9](https://github.com/amalgamated-tools/biblioteka/commit/43304e9c939c62e67e3076013e3b51f4dc1bc82a))
* **accessibility:** replace undefined `warning` palette in MetadataComparison ([#1820](https://github.com/amalgamated-tools/biblioteka/issues/1820)) ([83d287b](https://github.com/amalgamated-tools/biblioteka/commit/83d287bcb1c9b0ea6ffb8f3e5e9ab0a20b86001e))
* **code-simplifier:** refactor(opds,frontend): extract OPDS ErrorID constant and simplify Settings effect ([#1789](https://github.com/amalgamated-tools/biblioteka/issues/1789)) ([b3da489](https://github.com/amalgamated-tools/biblioteka/commit/b3da4897cc771662cf736c515f40678e742ce829))
* **csp:** remove 'unsafe-inline' from style-src in global CSP ([#1861](https://github.com/amalgamated-tools/biblioteka/issues/1861)) ([955d93c](https://github.com/amalgamated-tools/biblioteka/commit/955d93cabca2e0cefedcbc8c279638361507c89f))
* **handlers:** record download events for Kobo and OPDS downloads ([#1826](https://github.com/amalgamated-tools/biblioteka/issues/1826)) ([784d3d7](https://github.com/amalgamated-tools/biblioteka/commit/784d3d74a3dc30b6354c8203fee89012cd153454))
* **security:** add HSTS header and iss/aud JWT claims for defense-in-depth ([#1767](https://github.com/amalgamated-tools/biblioteka/issues/1767)) ([0e18a7f](https://github.com/amalgamated-tools/biblioteka/commit/0e18a7f7115ed6bb575f99789a948b4e3697739a))
* **security:** harden CSP by replacing unsafe-inline with sha256 hash in script-src ([#1768](https://github.com/amalgamated-tools/biblioteka/issues/1768)) ([aaae436](https://github.com/amalgamated-tools/biblioteka/commit/aaae436b2f84daf12845b36015776471f2578472))
* **workflows:** remove `--search "-parent-issue:*"` from Issue Arborist to unblock DIFC proxy ([#1780](https://github.com/amalgamated-tools/biblioteka/issues/1780)) ([f557ba7](https://github.com/amalgamated-tools/biblioteka/commit/f557ba7f9196105f84d606bb31d0442667f939ca))
* **workflows:** update concurrency group names and metadata hashes in reviewer workflows ([357591f](https://github.com/amalgamated-tools/biblioteka/commit/357591f604052835c04228db0ef8414d59fabf3c))


### Performance Improvements

* **db:** add composite index on reading_progress(user_id, updated_at) ([#1876](https://github.com/amalgamated-tools/biblioteka/issues/1876)) ([58f51e1](https://github.com/amalgamated-tools/biblioteka/commit/58f51e194d049e75ce239a6c15f7e9b06f22b678))
* **db:** add index on book_downloads.book_file_id for cascade delete ([#1806](https://github.com/amalgamated-tools/biblioteka/issues/1806)) ([66cfa6d](https://github.com/amalgamated-tools/biblioteka/commit/66cfa6d19dc70e2309f21c20d09026f14966a38c))
* **db:** optimize ListAuditLogs to single COUNT(*) OVER() query ([#1854](https://github.com/amalgamated-tools/biblioteka/issues/1854)) ([9e06d5d](https://github.com/amalgamated-tools/biblioteka/commit/9e06d5d280d897f48375ec67825d602bdd487886))
* **server:** add Cache-Control headers for Vite content-hashed assets ([#1790](https://github.com/amalgamated-tools/biblioteka/issues/1790)) ([140a1ae](https://github.com/amalgamated-tools/biblioteka/commit/140a1ae5f15fe5514064722781b8febd72aca0c3))

## [0.10.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.9.0...v0.10.0) (2026-04-12)


### Features

* **db:** add benchmark suite for DB query functions ([#1705](https://github.com/amalgamated-tools/biblioteka/issues/1705)) ([c0e0b41](https://github.com/amalgamated-tools/biblioteka/commit/c0e0b41eb804016481f23ee8b67e8ca035760e59))
* **db:** SQLite FTS5 for book search + PostgreSQL pg_trgm indexes ([#1716](https://github.com/amalgamated-tools/biblioteka/issues/1716)) ([b435e40](https://github.com/amalgamated-tools/biblioteka/commit/b435e40c31d041d4f079aaa24b0f69a40cc244ab))


### Bug Fixes

* **a11y:** address four keyboard/screen-reader accessibility issues ([#1641](https://github.com/amalgamated-tools/biblioteka/issues/1641)–[#1644](https://github.com/amalgamated-tools/biblioteka/issues/1644)) ([#1700](https://github.com/amalgamated-tools/biblioteka/issues/1700)) ([7323215](https://github.com/amalgamated-tools/biblioteka/commit/7323215aac4fb16799795bab2733e0709c411598))
* **accessibility:** add accessible label to MetadataComparison "values match" checkmark ([#1728](https://github.com/amalgamated-tools/biblioteka/issues/1728)) ([9ba1c81](https://github.com/amalgamated-tools/biblioteka/commit/9ba1c815abaffeb79e28bdf2c86626a501cb4f2e))
* **accessibility:** add required-fields legend to BookEdit and LibraryForm ([#1662](https://github.com/amalgamated-tools/biblioteka/issues/1662)) ([038a012](https://github.com/amalgamated-tools/biblioteka/commit/038a0120ce2a0a158da324f561eb51d43ab5b6a0))
* **accessibility:** add role="status" to Settings tab loading paragraphs ([#1726](https://github.com/amalgamated-tools/biblioteka/issues/1726)) ([b7311d0](https://github.com/amalgamated-tools/biblioteka/commit/b7311d01a957575965ec6fa44558d25798faa5b2))
* **accessibility:** manage keyboard focus on Email Book modal open/close ([#1659](https://github.com/amalgamated-tools/biblioteka/issues/1659)) ([4f6dcdb](https://github.com/amalgamated-tools/biblioteka/commit/4f6dcdbdbc48c9933ea94373a83aff399a57d6cf))
* **accessibility:** replace role="link" on `<tr>` with native anchor in BookList table view ([#1660](https://github.com/amalgamated-tools/biblioteka/issues/1660)) ([c1e360c](https://github.com/amalgamated-tools/biblioteka/commit/c1e360c69c9bc480ae69ecafa4dd7911f32904bd))
* **accessibility:** trap focus within EmailBookModal dialog ([#1723](https://github.com/amalgamated-tools/biblioteka/issues/1723)) ([c914486](https://github.com/amalgamated-tools/biblioteka/commit/c914486d882e948f3edb39cc4f62d77c1cd8a0ad))
* **accessibility:** unique aria-label on "Remove folder" buttons in LibraryForm ([#1661](https://github.com/amalgamated-tools/biblioteka/issues/1661)) ([0e77a72](https://github.com/amalgamated-tools/biblioteka/commit/0e77a720f4f7a4ae40a6ec39220612a0522e2144))
* **auth:** add rate limiting to PUT /api/auth/password ([#1715](https://github.com/amalgamated-tools/biblioteka/issues/1715)) ([706a0e3](https://github.com/amalgamated-tools/biblioteka/commit/706a0e37d74f20a8ff9aebfd690246e9e5aba61c))
* **auth:** normalize OIDC login error to prevent account enumeration ([#1713](https://github.com/amalgamated-tools/biblioteka/issues/1713)) ([0d64e5c](https://github.com/amalgamated-tools/biblioteka/commit/0d64e5cb8be4c75a868549150fb32d64d6a83111))
* **ci:** fix permission denied error when uploading super-linter log artifact ([#1674](https://github.com/amalgamated-tools/biblioteka/issues/1674)) ([5c2cbc6](https://github.com/amalgamated-tools/biblioteka/commit/5c2cbc64a01d81626dfac25895f924683110cd05))
* **code-simplifier:** consolidate MIME type fallback in filetype package ([#1648](https://github.com/amalgamated-tools/biblioteka/issues/1648)) ([273bfff](https://github.com/amalgamated-tools/biblioteka/commit/273bfff842187f048679df73aa797f0b975ad282))
* **kobo:** replace inline json.NewEncoder with writeKoboJSON in kobo_sync.go ([#1711](https://github.com/amalgamated-tools/biblioteka/issues/1711)) ([6864dfe](https://github.com/amalgamated-tools/biblioteka/commit/6864dfe26b9cb90c60f6d1f4766b2fd1bdaa2844))
* **telemetry:** fix Go convention violations in telemetry.go ([#1712](https://github.com/amalgamated-tools/biblioteka/issues/1712)) ([0c5a736](https://github.com/amalgamated-tools/biblioteka/commit/0c5a736465393101f50e0cd30519e32c0f1c607a))
* **workflows:** add contents: read permission to weekly-issue-summary ([#1710](https://github.com/amalgamated-tools/biblioteka/issues/1710)) ([1622a74](https://github.com/amalgamated-tools/biblioteka/commit/1622a748b1659b38cdf3660bd44b6b8c1a3cf3fa))
* **workflows:** disable lockdown mode for issue arborist and weekly summary tools ([c4b70b7](https://github.com/amalgamated-tools/biblioteka/commit/c4b70b777094861b9552b029343d25d16ca59eed))
* **workflows:** disable lockdown mode in discussion task miner ([c700fcc](https://github.com/amalgamated-tools/biblioteka/commit/c700fcc0726bb1755a2da4f356dd919ceed08513))
* **workflows:** update gh-aw metadata and prompt identifiers in markdown linter workflow ([f8d6d70](https://github.com/amalgamated-tools/biblioteka/commit/f8d6d7069a60d8603ab77d22207d1e873c93c27c))


### Performance Improvements

* **db:** add indexes on books.title and books.created_at ([#1677](https://github.com/amalgamated-tools/biblioteka/issues/1677)) ([780a5a3](https://github.com/amalgamated-tools/biblioteka/commit/780a5a31266c37d2ef65ecabff3d10c0b4dc6e1c))
* **db:** cache strings.NewReplacer at package level in SearchBooks ([#1706](https://github.com/amalgamated-tools/biblioteka/issues/1706)) ([fbdaa63](https://github.com/amalgamated-tools/biblioteka/commit/fbdaa63f935a039766e73824a8f1a97c3bdf231a))

## [0.9.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.8.0...v0.9.0) (2026-04-11)


### Features

* add download file button and count downloads across web UI, OPDS, and Kobo ([#1577](https://github.com/amalgamated-tools/biblioteka/issues/1577)) ([6cb6e58](https://github.com/amalgamated-tools/biblioteka/commit/6cb6e582eb86dc905ee834d14231cec2e86e6510))
* add errorfcheck analyzer to catch fmt.Errorf with no format verbs ([#1592](https://github.com/amalgamated-tools/biblioteka/issues/1592)) ([f671f68](https://github.com/amalgamated-tools/biblioteka/commit/f671f68898ad5a14d9accca89566383d8e8b94f4))
* add graph data ([#1588](https://github.com/amalgamated-tools/biblioteka/issues/1588)) ([c6c012d](https://github.com/amalgamated-tools/biblioteka/commit/c6c012def1ce3765d546c7abd1a8800cbc570f31))
* add watch folder setting for automatic book import ([#1578](https://github.com/amalgamated-tools/biblioteka/issues/1578)) ([c30c9f2](https://github.com/amalgamated-tools/biblioteka/commit/c30c9f2a1be9b70d52fd665d531e7e5b680e35f6))
* add weekly test coverage tracking agent ([#1519](https://github.com/amalgamated-tools/biblioteka/issues/1519)) ([5628e72](https://github.com/amalgamated-tools/biblioteka/commit/5628e72a2ce40995c27e703b13e572825b4d7aa6))
* **api:** add getTotalBooksCount helper ([#1596](https://github.com/amalgamated-tools/biblioteka/issues/1596)) ([4df0e4c](https://github.com/amalgamated-tools/biblioteka/commit/4df0e4c3fd61e9dd06d387d07e93ee75ebfd07dc))
* **auth:** add DISABLE_SIGNUP env var to gate public signup ([#1434](https://github.com/amalgamated-tools/biblioteka/issues/1434)) ([581bde5](https://github.com/amalgamated-tools/biblioteka/commit/581bde52f886554cff4b245e0079f6f0eab553ec))
* **auth:** warn when JWT_SECRET is shorter than 32 characters ([#1517](https://github.com/amalgamated-tools/biblioteka/issues/1517)) ([f431e29](https://github.com/amalgamated-tools/biblioteka/commit/f431e290d41f67a4894472c45cba1936cd4b7312))
* **books:** add POST /api/books/upload endpoint ([#1584](https://github.com/amalgamated-tools/biblioteka/issues/1584)) ([afd3082](https://github.com/amalgamated-tools/biblioteka/commit/afd30824ab7864f16e9949cfbe0406702b0465c7))
* **books:** email a book file as an attachment ([#1602](https://github.com/amalgamated-tools/biblioteka/issues/1602)) ([25bf393](https://github.com/amalgamated-tools/biblioteka/commit/25bf3931c6ee8ff58e0fda9e4bf91eabff03212e))
* **lint:** add slogcheck analyzer to ban slog.Any for typed values ([#1439](https://github.com/amalgamated-tools/biblioteka/issues/1439)) ([cdce158](https://github.com/amalgamated-tools/biblioteka/commit/cdce1588e780d053466093ea42b67a7c22e049ff))
* **metadata:** add remote metadata fetch, review, and apply workflow ([#1532](https://github.com/amalgamated-tools/biblioteka/issues/1532)) ([58931c9](https://github.com/amalgamated-tools/biblioteka/commit/58931c97fc2e734c6c02ec8e24fa0455e5a669bf))
* **server:** apply security headers globally via middleware ([#1432](https://github.com/amalgamated-tools/biblioteka/issues/1432)) ([596d34c](https://github.com/amalgamated-tools/biblioteka/commit/596d34c133e5fb2011202d36d64e5973cdcabfb8))


### Bug Fixes

* **accessibility:** address WCAG issues from Daily Accessibility Review ([#1568](https://github.com/amalgamated-tools/biblioteka/issues/1568)) ([19470fc](https://github.com/amalgamated-tools/biblioteka/commit/19470fc6ac667157d9c0435c90bb183e0b6324fa))
* **accessibility:** announce theme changes to screen readers via live region ([#1513](https://github.com/amalgamated-tools/biblioteka/issues/1513)) ([ebfed6b](https://github.com/amalgamated-tools/biblioteka/commit/ebfed6b7994301a2ccc98ec54290bf5df2842894))
* **accessibility:** fix 5 WCAG AA violations from daily accessibility review ([#1479](https://github.com/amalgamated-tools/biblioteka/issues/1479)) ([3dd274b](https://github.com/amalgamated-tools/biblioteka/commit/3dd274bc0bd37d8226beff4320af41a02aeb87d8))
* **accessibility:** fix broken heading hierarchy on Libraries page ([#1478](https://github.com/amalgamated-tools/biblioteka/issues/1478)) ([2432c99](https://github.com/amalgamated-tools/biblioteka/commit/2432c99a3b0995c312788cded1b46aa29b023274))
* **accessibility:** improve dark mode placeholder contrast in TextInput ([#1512](https://github.com/amalgamated-tools/biblioteka/issues/1512)) ([d42fd74](https://github.com/amalgamated-tools/biblioteka/commit/d42fd74a8b8b6504d46c118793c52935355b2067))
* **accessibility:** improve form border contrast and add theme change announcements ([#1520](https://github.com/amalgamated-tools/biblioteka/issues/1520)) ([50fe2b2](https://github.com/amalgamated-tools/biblioteka/commit/50fe2b20e530d96e836538ebedb610557d428c9b))
* **accessibility:** improve form control contrast and add theme change announcements ([#1511](https://github.com/amalgamated-tools/biblioteka/issues/1511)) ([25f5015](https://github.com/amalgamated-tools/biblioteka/commit/25f50156659ae9c1aaff80fdebb3775b0ac2facd))
* **accessibility:** include library name in page title (WCAG 2.4.2) ([#1571](https://github.com/amalgamated-tools/biblioteka/issues/1571)) ([b6988e8](https://github.com/amalgamated-tools/biblioteka/commit/b6988e8faaaddd8416d3e3283e5e183a380c1807))
* address four recurring code quality issues across handlers, jobs, and db ([#1444](https://github.com/amalgamated-tools/biblioteka/issues/1444)) ([5d4089f](https://github.com/amalgamated-tools/biblioteka/commit/5d4089f566a9fc7a1dc1add8aa575ba1593bf5f2))
* **auth:** max password length + API key entropy hardening ([#1494](https://github.com/amalgamated-tools/biblioteka/issues/1494)) ([bdd49cd](https://github.com/amalgamated-tools/biblioteka/commit/bdd49cdf22a2c10e9da327802e9f4f4cb499704c))
* **auth:** prevent rate limiter X-Forwarded-For spoofing ([#1518](https://github.com/amalgamated-tools/biblioteka/issues/1518)) ([8c6bc41](https://github.com/amalgamated-tools/biblioteka/commit/8c6bc41fcd0d98a14360e863b644926f3319723b))
* **auth:** raise bcrypt work factor from 10 to 12 ([#1433](https://github.com/amalgamated-tools/biblioteka/issues/1433)) ([f3681f2](https://github.com/amalgamated-tools/biblioteka/commit/f3681f26a39dd72b743ed999d76b6a2adc863f72))
* **auth:** raise minPasswordLength from 6 to 8 per NIST SP 800-63B ([#1435](https://github.com/amalgamated-tools/biblioteka/issues/1435)) ([3e5d0fb](https://github.com/amalgamated-tools/biblioteka/commit/3e5d0fb840b482ce6291e4fb6f8af545e6fa87e5))
* **auth:** redact email in login and OIDC callback logs ([#1488](https://github.com/amalgamated-tools/biblioteka/issues/1488)) ([2aa3356](https://github.com/amalgamated-tools/biblioteka/commit/2aa3356bd5d58a0ca5665699e099cc5caed73203))
* **ci:** add shared-instructions.md with noop fallback guidance for agentic workflows ([#1626](https://github.com/amalgamated-tools/biblioteka/issues/1626)) ([2bbafbf](https://github.com/amalgamated-tools/biblioteka/commit/2bbafbf6595da3193ed549331b4b75e708dda2b5))
* **ci:** ensure Daily Workflow Updater calls noop safe-output when no updates found ([#1627](https://github.com/amalgamated-tools/biblioteka/issues/1627)) ([65e6cf5](https://github.com/amalgamated-tools/biblioteka/commit/65e6cf5b5d598a04535e8ed60070548043fe3e79))
* **ci:** remove invalid `--squash` flag from dependabot auto-merge workflow ([#1457](https://github.com/amalgamated-tools/biblioteka/issues/1457)) ([772b43d](https://github.com/amalgamated-tools/biblioteka/commit/772b43dfffa5355148f7daff76c43788820a1125))
* **code-simplifier:** instruct agent to call noop when no action is needed ([#1636](https://github.com/amalgamated-tools/biblioteka/issues/1636)) ([35f4172](https://github.com/amalgamated-tools/biblioteka/commit/35f41729bd8e1b3fda25f33408c8b7ef634b6603))
* **code-simplifier:** refactor(db): add collectRowsAndTotal helper and apply to paginated book queries ([#1467](https://github.com/amalgamated-tools/biblioteka/issues/1467)) ([f7fb055](https://github.com/amalgamated-tools/biblioteka/commit/f7fb055fe1f968c306a9af574ed2dfe7497236c8))
* **code-simplifier:** refactor(preferences): remove redundant aria-live on role=status span ([#1560](https://github.com/amalgamated-tools/biblioteka/issues/1560)) ([29302de](https://github.com/amalgamated-tools/biblioteka/commit/29302defc846bbaf8afbcb4535cf82b97c6429a8))
* **code-simplifier:** refactor(preferences): use ThemePreference type and consolidate timer cleanup in tests ([#1524](https://github.com/amalgamated-tools/biblioteka/issues/1524)) ([4820c78](https://github.com/amalgamated-tools/biblioteka/commit/4820c785468dcdd102c3b9b362b1d4a34dc8aff6))
* **db:** use bookColumnsWithPrefix in ListBooksByLibrary ([#1438](https://github.com/amalgamated-tools/biblioteka/issues/1438)) ([3002656](https://github.com/amalgamated-tools/biblioteka/commit/30026566ad7d7fb95b5eb18b1a69f4a89611938c))
* enforce static slog messages (sloglint static-msg) ([#1440](https://github.com/amalgamated-tools/biblioteka/issues/1440)) ([f106484](https://github.com/amalgamated-tools/biblioteka/commit/f106484c92fdd2e6c23d4379da59eb264816edbd))
* **frontend:** migrate JWT from localStorage to in-memory storage ([#1430](https://github.com/amalgamated-tools/biblioteka/issues/1430)) ([d1306e5](https://github.com/amalgamated-tools/biblioteka/commit/d1306e5e3461d28dc95dc2f0a335847cb00b8dbd))
* **handlers:** add context timeout to library path validation to prevent handler goroutine blocking ([#1486](https://github.com/amalgamated-tools/biblioteka/issues/1486)) ([828c6ac](https://github.com/amalgamated-tools/biblioteka/commit/828c6ac46f0b0e86c65f7c67458b285a788c973e))
* **handlers:** prevent open redirect via unvalidated cover_image_url scheme ([#1431](https://github.com/amalgamated-tools/biblioteka/issues/1431)) ([0eedebc](https://github.com/amalgamated-tools/biblioteka/commit/0eedebcb30371a2af2516f29a4b5e588316dd11b))
* **handlers:** remove premature log and unify admin auth check ([#1441](https://github.com/amalgamated-tools/biblioteka/issues/1441)) ([7bf4e82](https://github.com/amalgamated-tools/biblioteka/commit/7bf4e8294fc15002f98effc7c782a7181a2eda28))
* **handlers:** use write-specific error handling in updateProfile and HandleSetAdmin ([#1591](https://github.com/amalgamated-tools/biblioteka/issues/1591)) ([d1a4575](https://github.com/amalgamated-tools/biblioteka/commit/d1a4575c377454c2501b61475f874dd0eb752418))
* **kobo:** replace aria-disabled workaround with real disabled attribute ([#1484](https://github.com/amalgamated-tools/biblioteka/issues/1484)) ([0ea3bbc](https://github.com/amalgamated-tools/biblioteka/commit/0ea3bbcb5e1eccc8ba5908ed3c0cef81702fd6ee))
* replace err.Error() leakage with typed sentinel errors ([#1445](https://github.com/amalgamated-tools/biblioteka/issues/1445)) ([99faac2](https://github.com/amalgamated-tools/biblioteka/commit/99faac25fd4ea8c9b2d8e45c26e176c14a1b9674))
* **security:** block SSRF via admin-supplied OIDC issuer URL ([#1422](https://github.com/amalgamated-tools/biblioteka/issues/1422)) ([cd53b42](https://github.com/amalgamated-tools/biblioteka/commit/cd53b429e77fabae7ee4858d3eff6bb9f60dda69))
* **security:** validate book file path against library roots to prevent path traversal ([#1421](https://github.com/amalgamated-tools/biblioteka/issues/1421)) ([4698a39](https://github.com/amalgamated-tools/biblioteka/commit/4698a39383094045147802586de28b0394622e8d))
* **workflow:** add noop safe-output instructions to update-docs workflow ([#1580](https://github.com/amalgamated-tools/biblioteka/issues/1580)) ([7ea4de1](https://github.com/amalgamated-tools/biblioteka/commit/7ea4de10c2d22fbf24fc9aea6991ee5a3f9c867e))
* **workflow:** add noop safe-output to schema consistency checker ([#1570](https://github.com/amalgamated-tools/biblioteka/issues/1570)) ([a0f2161](https://github.com/amalgamated-tools/biblioteka/commit/a0f2161d574ff08ce523039ceb60bb509be6fa92))
* **workflows:** add noop safe-output fallback to daily-team-evolution-insights ([#1635](https://github.com/amalgamated-tools/biblioteka/issues/1635)) ([08bdc96](https://github.com/amalgamated-tools/biblioteka/commit/08bdc96d7e33c5fa337eddd8dc303dc27c72482e))

## [0.8.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.7.0...v0.8.0) (2026-04-06)


### Features

* **auth:** expose name in userDTO and add PUT /api/auth/me profile update ([#1373](https://github.com/amalgamated-tools/biblioteka/issues/1373)) ([cb99ca4](https://github.com/amalgamated-tools/biblioteka/commit/cb99ca4a7881b72a73e0dbac72c585bebaadbe24))
* **workflows:** add cross-agent awareness to doc automation workflows ([#1394](https://github.com/amalgamated-tools/biblioteka/issues/1394)) ([f86d3c8](https://github.com/amalgamated-tools/biblioteka/commit/f86d3c807cb11c55221796292c0197af231f38f8))
* **workflows:** add daily security review workflow ([#1362](https://github.com/amalgamated-tools/biblioteka/issues/1362)) ([ad5dd3b](https://github.com/amalgamated-tools/biblioteka/commit/ad5dd3b39e433b8a8a090ebe1cfe4e40c0322c5c))


### Bug Fixes

* **accessibility:** prefers-reduced-motion, aria-live on loading state, and contrast fixes ([#1400](https://github.com/amalgamated-tools/biblioteka/issues/1400)) ([7df554f](https://github.com/amalgamated-tools/biblioteka/commit/7df554fb67ae629e73552f41ab4bbfb410562854))
* **accessibility:** raise icon-only button contrast to meet WCAG 1.4.11 (3:1) ([#1401](https://github.com/amalgamated-tools/biblioteka/issues/1401)) ([09e6469](https://github.com/amalgamated-tools/biblioteka/commit/09e64697e78a65a1b75a156f889f7412e2ff8ff3))
* address daily nitpick review findings ([#1396](https://github.com/amalgamated-tools/biblioteka/issues/1396)) ([ebd1725](https://github.com/amalgamated-tools/biblioteka/commit/ebd17256d46bf30f99ed5ffcfa6a293139d469b7))
* **code-simplifier:** refactor(db): apply scanRow/collectRows helpers to book_files and libraries ([#1404](https://github.com/amalgamated-tools/biblioteka/issues/1404)) ([ca8f248](https://github.com/amalgamated-tools/biblioteka/commit/ca8f248f9ce0408757c03293c62e6057012e5e57))
* **daily-doc-updater:** add deduplication, 8-PR hard cap, and 48h lookback window ([#1385](https://github.com/amalgamated-tools/biblioteka/issues/1385)) ([c2cbfce](https://github.com/amalgamated-tools/biblioteka/commit/c2cbfcee5e54496506ff78c1a35c90a2acc650f7))
* **dashboard:** replace hardcoded zero stats with real data ([#1372](https://github.com/amalgamated-tools/biblioteka/issues/1372)) ([4599740](https://github.com/amalgamated-tools/biblioteka/commit/45997403e7d48ca0428e8181690cacd80062f18b))
* **db:** use errors.New for ErrInvalidGoodreadsMetadataStatus sentinel ([#1365](https://github.com/amalgamated-tools/biblioteka/issues/1365)) ([10bdfd4](https://github.com/amalgamated-tools/biblioteka/commit/10bdfd46e3b5c0820ac71004a08333efbf82bc8e))
* **logging:** replace generic otelkeys.ID with specific entity keys in db and handler layers ([#1364](https://github.com/amalgamated-tools/biblioteka/issues/1364)) ([55138c6](https://github.com/amalgamated-tools/biblioteka/commit/55138c66c3d3b62df36d7bcbdbf9484609fb6dd4))
* normalize error handling in goodreads_metadata and validatePassword ([#1367](https://github.com/amalgamated-tools/biblioteka/issues/1367)) ([1e372ca](https://github.com/amalgamated-tools/biblioteka/commit/1e372ca2e96bda95c2d620db6e76591192b226ad))
* **smtp:** accept From addresses with display names ([#1371](https://github.com/amalgamated-tools/biblioteka/issues/1371)) ([8107391](https://github.com/amalgamated-tools/biblioteka/commit/810739133d80a8a5d74b77a63e8fb02df6580dc5))
* **stores:** expose error state in CrudStore.load() ([#1370](https://github.com/amalgamated-tools/biblioteka/issues/1370)) ([c2e3ffe](https://github.com/amalgamated-tools/biblioteka/commit/c2e3ffe096676ddca0d7370df3550fb43deea61d))

## [0.7.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.6.1...v0.7.0) (2026-04-05)


### Features

* **api:** add GET /api/authors/{id}/books and GET /api/series/{id}/books ([#1232](https://github.com/amalgamated-tools/biblioteka/issues/1232)) ([6c2c151](https://github.com/amalgamated-tools/biblioteka/commit/6c2c15137b3f09dab4b0e44da293da44a0967fd6))
* **books:** expose SearchBooks via GET /api/books?query= ([#1222](https://github.com/amalgamated-tools/biblioteka/issues/1222)) ([17fbb17](https://github.com/amalgamated-tools/biblioteka/commit/17fbb17201e010a15ef4941c33aea9bd5e05aba9))
* **frontend:** extract SuccessTimerState to eliminate duplicated timeout management ([#1341](https://github.com/amalgamated-tools/biblioteka/issues/1341)) ([9fc4600](https://github.com/amalgamated-tools/biblioteka/commit/9fc4600d322cec08e753d02112ec31d6ddab5029))
* **jobs:** add enrich:goodreads background job for book metadata ([#1354](https://github.com/amalgamated-tools/biblioteka/issues/1354)) ([a1d8e20](https://github.com/amalgamated-tools/biblioteka/commit/a1d8e208b3222bd9647825c87de86e7fa191cfcb))
* **metrics:** add tracker-id to 7 workflows missing from ecosystem metrics ([#1349](https://github.com/amalgamated-tools/biblioteka/issues/1349)) ([ccb3a24](https://github.com/amalgamated-tools/biblioteka/commit/ccb3a249404d9010b17ac2d80b0c2de8169a109d))
* **workflows:** add daily codebase nitpick reviewer workflow ([#1344](https://github.com/amalgamated-tools/biblioteka/issues/1344)) ([524a32a](https://github.com/amalgamated-tools/biblioteka/commit/524a32a11180a463d7c4a19122e75d7f4dabc4d2))
* **workflows:** compile and improve agentic workflow ecosystem ([#1355](https://github.com/amalgamated-tools/biblioteka/issues/1355)) ([c096674](https://github.com/amalgamated-tools/biblioteka/commit/c09667445b51e12561599ffecf11a7cc10680f75))
* **workflows:** explicit [@copilot](https://github.com/copilot) assignee on daily-accessibility-review issues ([#1342](https://github.com/amalgamated-tools/biblioteka/issues/1342)) ([6c8e806](https://github.com/amalgamated-tools/biblioteka/commit/6c8e806764fb503d1f811a26aeabd8028f9c492e))


### Bug Fixes

* **accessibility:** add aria-hidden="true" to decorative Lucide icons throughout the app ([#1338](https://github.com/amalgamated-tools/biblioteka/issues/1338)) ([559ee21](https://github.com/amalgamated-tools/biblioteka/commit/559ee21c81afa5f80cda689f04ad2aece8ff5823))
* **accessibility:** hide decorative spinner from screen readers; add role=status to loading message ([#1276](https://github.com/amalgamated-tools/biblioteka/issues/1276)) ([5be6a90](https://github.com/amalgamated-tools/biblioteka/commit/5be6a908d7b66c6c4e66a4b1290b7c4b86830aa9))
* **accessibility:** resolve WCAG 1.4.3 color contrast failures for secondary/informational text ([#1339](https://github.com/amalgamated-tools/biblioteka/issues/1339)) ([d3c1e50](https://github.com/amalgamated-tools/biblioteka/commit/d3c1e50299b2417e0c8592a445c2b899e7b40575))
* **accessibility:** trap keyboard focus in mobile sidebar using `inert` and Escape key ([#1275](https://github.com/amalgamated-tools/biblioteka/issues/1275)) ([7325e00](https://github.com/amalgamated-tools/biblioteka/commit/7325e00b147678895948c1a966b19f9d971acf1c))
* **accessibility:** use DeleteConfirmation component in LibraryForm ([#1250](https://github.com/amalgamated-tools/biblioteka/issues/1250)) ([a9ec408](https://github.com/amalgamated-tools/biblioteka/commit/a9ec408046746488243e0c452c37448b8a3fd540))
* **accessibility:** use dl/dt/dd semantic structure for Dashboard stat cards ([#1312](https://github.com/amalgamated-tools/biblioteka/issues/1312)) ([c782047](https://github.com/amalgamated-tools/biblioteka/commit/c782047c324433b7a093d21d9f2840df2024fd37))
* **code-simplifier:** refactor(dashboard): extract stat cards into data-driven loop ([#1327](https://github.com/amalgamated-tools/biblioteka/issues/1327)) ([8d75593](https://github.com/amalgamated-tools/biblioteka/commit/8d755936af53fa471faeab35bed6c3487baf563e))
* **code-simplifier:** refactor(handlers): extract listParentBooks helper and use mapSlice in listBooks ([#1258](https://github.com/amalgamated-tools/biblioteka/issues/1258)) ([4ccda47](https://github.com/amalgamated-tools/biblioteka/commit/4ccda47f26b6d9a6cb7abcc886a9c349b34db4bd))
* **code-simplifier:** refactor(tests): replace t.Fatal/t.Fatalf with testify/require ([#1336](https://github.com/amalgamated-tools/biblioteka/issues/1336)) ([e1e4b7b](https://github.com/amalgamated-tools/biblioteka/commit/e1e4b7b02251eb640340eb65ae44da28e7158b38))
* **db:** drop legacy kobo_tokens.token column ([#1345](https://github.com/amalgamated-tools/biblioteka/issues/1345)) ([fd6a05e](https://github.com/amalgamated-tools/biblioteka/commit/fd6a05ef0e1cabe27efbae91ca5a2acfd8a3735d))
* **db:** standardize goodreads_metadata index names across SQLite and PostgreSQL ([#1224](https://github.com/amalgamated-tools/biblioteka/issues/1224)) ([b6e7c53](https://github.com/amalgamated-tools/biblioteka/commit/b6e7c5394094477c32eac909acddc50d098c61aa))
* **q:** allow Q workflow to modify protected workflow files ([#1299](https://github.com/amalgamated-tools/biblioteka/issues/1299)) ([1d823f1](https://github.com/amalgamated-tools/biblioteka/commit/1d823f1f81d0a27173636eab5e2a7aa5e41c1aea))
* **types:** use `string | null` for optional input type fields ([#1297](https://github.com/amalgamated-tools/biblioteka/issues/1297)) ([be9498d](https://github.com/amalgamated-tools/biblioteka/commit/be9498dba955ebdee4e5ac29bd46d3caf75308ec))
* **workflows:** switch daily-team-evolution-insights from claude to copilot engine ([#1280](https://github.com/amalgamated-tools/biblioteka/issues/1280)) ([b9bbe70](https://github.com/amalgamated-tools/biblioteka/commit/b9bbe70f68899cc7f8ff5ae82cf636362a809a62))
* **workflows:** trigger update-docs after Test CI succeeds instead of every push ([#1226](https://github.com/amalgamated-tools/biblioteka/issues/1226)) ([e8cdbe5](https://github.com/amalgamated-tools/biblioteka/commit/e8cdbe5dd3c908657d1935d6e3e630904cf071d4))

## [0.6.1](https://github.com/amalgamated-tools/biblioteka/compare/v0.6.0...v0.6.1) (2026-04-03)


### Bug Fixes

* **workflows:** issue-triage agent fails with no safe outputs when applying labels ([#1199](https://github.com/amalgamated-tools/biblioteka/issues/1199)) ([dd55de8](https://github.com/amalgamated-tools/biblioteka/commit/dd55de8e95ad796eb917d82c3bcaf3a0dfa957d6))

## [0.6.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.5.0...v0.6.0) (2026-04-03)


### Features

* **epub:** add EPUB3 test coverage and VS Code debug config ([#1147](https://github.com/amalgamated-tools/biblioteka/issues/1147)) ([389f3a2](https://github.com/amalgamated-tools/biblioteka/commit/389f3a27e4ef4552b3210ff2f419eccf9c5bc695))


### Bug Fixes

* **accessibility:** sidebar library-settings link visible at focus time (WCAG 2.4.7) ([#1144](https://github.com/amalgamated-tools/biblioteka/issues/1144)) ([245f6bc](https://github.com/amalgamated-tools/biblioteka/commit/245f6bc97f8414425e41c31a3fa711337802d5fe))
* **code-simplifier:** refactor(server): consolidate OPDS and KOSync credential adapters ([#1167](https://github.com/amalgamated-tools/biblioteka/issues/1167)) ([2c6d695](https://github.com/amalgamated-tools/biblioteka/commit/2c6d69597c079775564ddd1911c6c059b8e6591a))

## [0.5.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.4.0...v0.5.0) (2026-04-03)


### Features

* auto-refresh book list after adding a library ([#1111](https://github.com/amalgamated-tools/biblioteka/issues/1111)) ([a115d15](https://github.com/amalgamated-tools/biblioteka/commit/a115d15c4603d2c8cefb17700d1e514ac8538a47))

## [0.4.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.3.0...v0.4.0) (2026-04-02)


### Features

* Exif refactor ([#934](https://github.com/amalgamated-tools/biblioteka/issues/934)) ([9ebba1f](https://github.com/amalgamated-tools/biblioteka/commit/9ebba1f1cbf70a994f94633fa69d79f90d6eb38f))


### Bug Fixes

* **accessibility:** expose role="switch" state via aria-checked in LibraryForm toggle ([#1042](https://github.com/amalgamated-tools/biblioteka/issues/1042)) ([68f2d48](https://github.com/amalgamated-tools/biblioteka/commit/68f2d489e801de92e95ce4d9e9867d7f04e16fe1))
* **accessibility:** move keyboard focus to main content after SPA navigation ([#1082](https://github.com/amalgamated-tools/biblioteka/issues/1082)) ([56a80a3](https://github.com/amalgamated-tools/biblioteka/commit/56a80a3115841e4d14197bbb693e2b84349cc355))
* **accessibility:** replace window.confirm() with inline accessible confirmation dialogs ([#1041](https://github.com/amalgamated-tools/biblioteka/issues/1041)) ([ce33f26](https://github.com/amalgamated-tools/biblioteka/commit/ce33f26339344db8694b5af65a244fc2e066769b))
* **code-simplifier:** refactor(auth,handlers): simplify extractCreds closures and extract ctx in named_entity helpers ([#1033](https://github.com/amalgamated-tools/biblioteka/issues/1033)) ([625366f](https://github.com/amalgamated-tools/biblioteka/commit/625366fa841743314ef98351b1f2860bf23003b6))
* **code-simplifier:** refactor(settings): extract clearCopyTimeout helper in APIKeysTab and KoboTab ([#1059](https://github.com/amalgamated-tools/biblioteka/issues/1059)) ([5ce1ad0](https://github.com/amalgamated-tools/biblioteka/commit/5ce1ad07e5eadf4b07cefc03166f4cb3e4cc7a4b))
* **db:** replace manual dialect branch in UpsertOPDSCredential with d.now() ([#1038](https://github.com/amalgamated-tools/biblioteka/issues/1038)) ([214219b](https://github.com/amalgamated-tools/biblioteka/commit/214219bf7231465b84128095d551763616cb60e1))
* schema consistency — hide token_hash, add audit-log/credential API, move types to types.ts ([#1090](https://github.com/amalgamated-tools/biblioteka/issues/1090)) ([3e6ad7c](https://github.com/amalgamated-tools/biblioteka/commit/3e6ad7c4493a13eda84b9bc19a215c608c07d243))

## [0.3.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.2.0...v0.3.0) (2026-03-30)


### Features

* add Swagger annotations for Kobo tokens, OPDS credentials, KOSync credentials, and KOSync protocol endpoints ([#931](https://github.com/amalgamated-tools/biblioteka/issues/931)) ([1755a38](https://github.com/amalgamated-tools/biblioteka/commit/1755a389e3a9b8169c6f823f639b4a42ed959217))
* **frontend:** extract Button, TextInput, and AlertBanner as reusable UI components ([#947](https://github.com/amalgamated-tools/biblioteka/issues/947)) ([0244dab](https://github.com/amalgamated-tools/biblioteka/commit/0244dabfb418437363fa15b3ffa6ec259045dfcc))
* **frontend:** extract shared form validation utility ([#953](https://github.com/amalgamated-tools/biblioteka/issues/953)) ([7000d8e](https://github.com/amalgamated-tools/biblioteka/commit/7000d8ecb8b215776a95e8cc3ac5ac549907ae07))
* **router:** 404 handling and hash-embedded query parameter support ([#951](https://github.com/amalgamated-tools/biblioteka/issues/951)) ([8f64403](https://github.com/amalgamated-tools/biblioteka/commit/8f64403789613a923627f0f7350d257d5b92033d))
* **stores:** standardize idempotency guard across author and series stores ([#949](https://github.com/amalgamated-tools/biblioteka/issues/949)) ([5c38cd5](https://github.com/amalgamated-tools/biblioteka/commit/5c38cd576f14642d680dd7fc5526ca10d7002230))
* **swagger:** mark bookRequest.title as required ([#932](https://github.com/amalgamated-tools/biblioteka/issues/932)) ([f511f2a](https://github.com/amalgamated-tools/biblioteka/commit/f511f2aa7630c24a3e7ce8c8623cedcfc00144cc))


### Bug Fixes

* **accessibility:** add aria-label to Settings content section ([#867](https://github.com/amalgamated-tools/biblioteka/issues/867)) ([2d46e49](https://github.com/amalgamated-tools/biblioteka/commit/2d46e4936cb2044d5c734a41af11d12ca39607f8))
* **accessibility:** add onclick guard and aria-hidden to disabled Copy button in KoboTab ([#903](https://github.com/amalgamated-tools/biblioteka/issues/903)) ([14dafb2](https://github.com/amalgamated-tools/biblioteka/commit/14dafb25b38ffb29af7d3dbfde00ea1419725223))
* **accessibility:** link form helper text to inputs via aria-describedby ([#1002](https://github.com/amalgamated-tools/biblioteka/issues/1002)) ([5575a72](https://github.com/amalgamated-tools/biblioteka/commit/5575a72eb4bb9d7f71ed73a964a00a0e99956965))
* **accessibility:** replace sidebar `<h1>` with `<p>` to eliminate duplicate top-level headings ([#846](https://github.com/amalgamated-tools/biblioteka/issues/846)) ([a224f0b](https://github.com/amalgamated-tools/biblioteka/commit/a224f0b5bf578f563db1cb3f50db60f312b00e8f))
* **ci:** expand sparse-checkout in push_repo_memory to include metrics dir ([#1012](https://github.com/amalgamated-tools/biblioteka/issues/1012)) ([e16c8be](https://github.com/amalgamated-tools/biblioteka/commit/e16c8beacd9ed3ff7c96fe1d9ed63a70e996035a))
* **code-simplifier:** refactor(db): extend listAll to cover ListUsers ([#818](https://github.com/amalgamated-tools/biblioteka/issues/818)) ([f6a6183](https://github.com/amalgamated-tools/biblioteka/commit/f6a618301cf304bf579aed0426a4a0eb2586f82f))
* **code-simplifier:** refactor(frontend): extract derived values to eliminate nested ternaries ([#980](https://github.com/amalgamated-tools/biblioteka/issues/980)) ([b125942](https://github.com/amalgamated-tools/biblioteka/commit/b125942ed48bbf4e6c819ade95832305126b54d6))
* **code-simplifier:** refactor(handlers): apply listEntities, handleDBErr, and handleUpdateErr to libraries and admin ([#887](https://github.com/amalgamated-tools/biblioteka/issues/887)) ([2079734](https://github.com/amalgamated-tools/biblioteka/commit/2079734e4d50e1bf762ca30194089d8318e36383))
* **code-simplifier:** refactor(router): extract VALID_VIEWS constant and deduplicate parseHash call ([#1005](https://github.com/amalgamated-tools/biblioteka/issues/1005)) ([40f6583](https://github.com/amalgamated-tools/biblioteka/commit/40f65834d31a7e65227880f657cc8ed82fbd3f82))
* correct `auditLogDTO.Metadata` swagger type from `array[integer]` to `object` ([#930](https://github.com/amalgamated-tools/biblioteka/issues/930)) ([c469b39](https://github.com/amalgamated-tools/biblioteka/commit/c469b39f6dc1c99507f19fae404aa985fec0b735))
* **frontend:** replace $effect guard pattern with onMount for initial data fetching ([#957](https://github.com/amalgamated-tools/biblioteka/issues/957)) ([06c6e1c](https://github.com/amalgamated-tools/biblioteka/commit/06c6e1c43cf14a662abaccd6c740314d2e5ddec0))
* **frontend:** upgrade picomatch to 4.0.4 to patch ReDoS vulnerability (CVE-2026-33671) ([#965](https://github.com/amalgamated-tools/biblioteka/issues/965)) ([b1f74b7](https://github.com/amalgamated-tools/biblioteka/commit/b1f74b76054c530418ef2b92db387bd534d8293a))
* **frontend:** use onMount instead of $effect for version fetch in Sidebar ([#955](https://github.com/amalgamated-tools/biblioteka/issues/955)) ([449dddd](https://github.com/amalgamated-tools/biblioteka/commit/449dddd8a0abe0301f568199bb295e0c6f0bfc40))
* **workflows:** add missing permissions to daily-code-metrics workflow ([#828](https://github.com/amalgamated-tools/biblioteka/issues/828)) ([bcd52f0](https://github.com/amalgamated-tools/biblioteka/commit/bcd52f06107ecf589cded47b4b3c42e0874bd78b))
* **workflows:** address 5 critical schema consistency issues in agentic workflow frontmatter ([#847](https://github.com/amalgamated-tools/biblioteka/issues/847)) ([7729469](https://github.com/amalgamated-tools/biblioteka/commit/77294695775882a7973305daeaf1bfc501245912))

## [0.2.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.1.2...v0.2.0) (2026-03-24)


### Features

* **db:** remove num_pages from books and add goodreads_metadata table ([#792](https://github.com/amalgamated-tools/biblioteka/issues/792)) ([e6b0b32](https://github.com/amalgamated-tools/biblioteka/commit/e6b0b329884bd120a432fa9e00f5e4645e203b2e))


### Bug Fixes

* **accessibility:** add ARIA live regions to status and error messages ([#786](https://github.com/amalgamated-tools/biblioteka/issues/786)) ([653862c](https://github.com/amalgamated-tools/biblioteka/commit/653862ca6f92e2ba561ab11bc39380d8cf0e4b36))
* **accessibility:** remove tabindex="0" from Auth tabpanels with focusable content ([#807](https://github.com/amalgamated-tools/biblioteka/issues/807)) ([6f942c6](https://github.com/amalgamated-tools/biblioteka/commit/6f942c66ac50f93cca183df650987582dde46937))
* **accessibility:** unique aria-labels for Kobo sync token copy/delete buttons and API key delete buttons ([#787](https://github.com/amalgamated-tools/biblioteka/issues/787)) ([939645b](https://github.com/amalgamated-tools/biblioteka/commit/939645b553cb526c75352aa9b76d6e5f9923dccb))
* double posting ([f72f714](https://github.com/amalgamated-tools/biblioteka/commit/f72f7148e57a16e3ee90acd486c730d8f9ff9b09))

## [0.1.2](https://github.com/amalgamated-tools/biblioteka/compare/v0.1.0...v0.1.2) (2026-03-23)


### Features

* Add in support for querying Goodreads ([#644](https://github.com/amalgamated-tools/biblioteka/issues/644)) ([3cd7f58](https://github.com/amalgamated-tools/biblioteka/commit/3cd7f5815e53aaff9f01afbf78064024334f1a5f))
* add in workflows ([#610](https://github.com/amalgamated-tools/biblioteka/issues/610)) ([6fcf024](https://github.com/amalgamated-tools/biblioteka/commit/6fcf024fac83c85b09e386b82e8bf33e53b22c44))
* Refactor file organization logic to support multiple organization types ([#553](https://github.com/amalgamated-tools/biblioteka/issues/553)) ([89f480b](https://github.com/amalgamated-tools/biblioteka/commit/89f480b014be2cbb4eff5d30b8209066af754649))


### Bug Fixes

* **accessibility:** add semantic sidebar group headings and scoped user table headers ([#577](https://github.com/amalgamated-tools/biblioteka/issues/577)) ([e7a4953](https://github.com/amalgamated-tools/biblioteka/commit/e7a4953693feaf64fd7d3943a6c30708657cf436))
* **accessibility:** describe toggle-admin button action in Users table ([#603](https://github.com/amalgamated-tools/biblioteka/issues/603)) ([16fcf18](https://github.com/amalgamated-tools/biblioteka/commit/16fcf18d8afdfb9657a614b6b38730c13ea67eca))
* **accessibility:** use header landmark for mobile top bar ([#675](https://github.com/amalgamated-tools/biblioteka/issues/675)) ([cac95b6](https://github.com/amalgamated-tools/biblioteka/commit/cac95b660bd1a6630b5a6420a38c2ef35509ebc9))
* **code-simplifier:** refactor(handlers): use errors.Is for sentinel error comparisons ([#686](https://github.com/amalgamated-tools/biblioteka/issues/686)) ([5b51848](https://github.com/amalgamated-tools/biblioteka/commit/5b518489b538dad78c3aa3e70417b90dc55faa75))
* **code-simplifier:** refactor(sidebar): use semantic h2 elements instead of span with ARIA roles ([#606](https://github.com/amalgamated-tools/biblioteka/issues/606)) ([661d592](https://github.com/amalgamated-tools/biblioteka/commit/661d592e567993a837acc5a49357187a36b25b5f))
* **code-simplifier:** refactor(sidecar): atomic cover write, urn:uuid prefix, and cleanup ([#551](https://github.com/amalgamated-tools/biblioteka/issues/551)) ([0c836c5](https://github.com/amalgamated-tools/biblioteka/commit/0c836c5a5a5cdc9d5e995766f7b8cb122c4fb4c5))
* don't lock ([#648](https://github.com/amalgamated-tools/biblioteka/issues/648)) ([f2a7b5b](https://github.com/amalgamated-tools/biblioteka/commit/f2a7b5bbaee71beeaa0fd471135f0238c5e129a7))
* **goodreads:** prevent API key leak, concurrent GraphQL fan-out, context-cancellation propagation ([#716](https://github.com/amalgamated-tools/biblioteka/issues/716)) ([e27f0b7](https://github.com/amalgamated-tools/biblioteka/commit/e27f0b75ae345770dcfe727e0874e5cb9274a5c8))
* **handlers:** return 400 for whitespace-only author and series names ([#649](https://github.com/amalgamated-tools/biblioteka/issues/649)) ([ca53101](https://github.com/amalgamated-tools/biblioteka/commit/ca53101cb1eb40dc55742cadcf7c3131a30ad831))


### Chores / CI

* **ci:** allow CI coach to open pull requests ([4a7a7c0](https://github.com/amalgamated-tools/biblioteka/commit/4a7a7c079663b23ecd8fef075e990bb76d875658))
* **ci:** refine and expand GitHub Actions workflows ([abb20d0](https://github.com/amalgamated-tools/biblioteka/commit/abb20d0543c42734173df6cb4b8e5ea8f3e42d6e), [282ca6b](https://github.com/amalgamated-tools/biblioteka/commit/282ca6bc36bd42c14ce8436d9fad383eb5d74c73), [b1a060d](https://github.com/amalgamated-tools/biblioteka/commit/b1a060d07a11eb80fce74a53849d5821448e8da9))
* **ci:** iterate on AI-assisted workflow configuration ([91df811](https://github.com/amalgamated-tools/biblioteka/commit/91df811b119bc90141de46db43756daa2b4fa6c5), [b9a8b10](https://github.com/amalgamated-tools/biblioteka/commit/b9a8b109782b4536335c6d83b23c2630d688cd20), [dcf1856](https://github.com/amalgamated-tools/biblioteka/commit/dcf1856409d4692e5d80a27126dcbc953f625b81))
* **workflows:** add concurrency cancel-in-progress and paths-ignore to Update Docs ([#755](https://github.com/amalgamated-tools/biblioteka/issues/755)) ([1204dff](https://github.com/amalgamated-tools/biblioteka/commit/1204dfff82fcaa0f0822293526f677ffd4292846))
* **workflows:** use announcement-capable discussion category for all agentic workflows ([#717](https://github.com/amalgamated-tools/biblioteka/issues/717)) ([50b5e41](https://github.com/amalgamated-tools/biblioteka/commit/50b5e4143573737a201dec11f2ff4df1ba9e8179))
* **workflows:** use installed gh-aw CLI in daily copilot token report ([#620](https://github.com/amalgamated-tools/biblioteka/issues/620)) ([a0fd72f](https://github.com/amalgamated-tools/biblioteka/commit/a0fd72f6bcfa8154c40c0f5d7e99e7788a822a71))

## [0.1.0](https://github.com/amalgamated-tools/biblioteka/compare/v0.0.7...v0.1.0) (2026-03-20)


### Features

* add support for extracting and serving embedded EPUB cover imag… ([#498](https://github.com/amalgamated-tools/biblioteka/issues/498)) ([c84cbc9](https://github.com/amalgamated-tools/biblioteka/commit/c84cbc9179236971c5b26fbc70da7a85b81097d7))
* **cli:** add scan-directory command to invoke jobs.ScanDirectory ([#321](https://github.com/amalgamated-tools/biblioteka/issues/321)) ([311533e](https://github.com/amalgamated-tools/biblioteka/commit/311533e38ea2a21d13a42d99f94bd7b5cd0cfa6e))
* implement ScanDirectory function for path scanning and job enqueuing ([#305](https://github.com/amalgamated-tools/biblioteka/issues/305)) ([5ce9b8c](https://github.com/amalgamated-tools/biblioteka/commit/5ce9b8c34dfbdf9f249ba4983956451c7359be15))
* Kobo e-reader sync integration ([#313](https://github.com/amalgamated-tools/biblioteka/issues/313)) ([0750122](https://github.com/amalgamated-tools/biblioteka/commit/07501225715e7a4da42e1cca5de27320272ab845))
* KOReader kosync-compatible reading progress sync ([#314](https://github.com/amalgamated-tools/biblioteka/issues/314)) ([bb8890f](https://github.com/amalgamated-tools/biblioteka/commit/bb8890f697a55c0a1b9b5104d4c8e93ecd50ebf8))
* **sidecar:** implement cover image and OPF metadata handling ([#534](https://github.com/amalgamated-tools/biblioteka/issues/534)) ([1e8821f](https://github.com/amalgamated-tools/biblioteka/commit/1e8821f25c47af15de1623f86cbb39cc58320487))
* update metadata extraction documentation and improve ExifTool integration ([#491](https://github.com/amalgamated-tools/biblioteka/issues/491)) ([b6dc3b0](https://github.com/amalgamated-tools/biblioteka/commit/b6dc3b0c98a24f9393657b3581855f0ee4c52651))
* we don't need the extra indirection ([#264](https://github.com/amalgamated-tools/biblioteka/issues/264)) ([4f28c96](https://github.com/amalgamated-tools/biblioteka/commit/4f28c964181d405851e1fe1d6ef13af4a9155045))


### Bug Fixes

* **accessibility:** add accessible labels to library form inputs ([#339](https://github.com/amalgamated-tools/biblioteka/issues/339)) ([b292e44](https://github.com/amalgamated-tools/biblioteka/commit/b292e442b6900f846123bb9294a318622807c1af))
* **accessibility:** Add ARIA tab semantics to Login/Sign Up toggle buttons ([#360](https://github.com/amalgamated-tools/biblioteka/issues/360)) ([6daadfb](https://github.com/amalgamated-tools/biblioteka/commit/6daadfbfa87802a3471d486e04c1e7bf034d0758))
* **accessibility:** add aria-current to active navigation buttons ([#338](https://github.com/amalgamated-tools/biblioteka/issues/338)) ([7b504e1](https://github.com/amalgamated-tools/biblioteka/commit/7b504e1d6ebe89f1929035da0e7340e01c065973))
* **accessibility:** add aria-label to icon-only "Create library" button (WCAG 4.1.2) ([#485](https://github.com/amalgamated-tools/biblioteka/issues/485)) ([fc99bdb](https://github.com/amalgamated-tools/biblioteka/commit/fc99bdb63be737d35d739bafd481eb8cb11975ff))
* **accessibility:** add aria-label to unlabelled landmark regions (WCAG 4.1.2) ([#484](https://github.com/amalgamated-tools/biblioteka/issues/484)) ([9172c80](https://github.com/amalgamated-tools/biblioteka/commit/9172c801bd38760fe6761a01af3e681cfd3d67ab))
* **accessibility:** add autocomplete attributes to password inputs in Account Settings ([#342](https://github.com/amalgamated-tools/biblioteka/issues/342)) ([c793573](https://github.com/amalgamated-tools/biblioteka/commit/c7935732f5562fd009770cf5a3b7c2b4c43edcad))
* **accessibility:** Add main landmark region to login page ([#358](https://github.com/amalgamated-tools/biblioteka/issues/358)) ([903a27e](https://github.com/amalgamated-tools/biblioteka/commit/903a27e4b094ba120b2a5531b45c006c102439f9))
* **accessibility:** add role="switch" to LibraryForm monitor toggle ([#361](https://github.com/amalgamated-tools/biblioteka/issues/361)) ([58a077c](https://github.com/amalgamated-tools/biblioteka/commit/58a077c0ec42f69f117609f0658c8374a8823e24))
* **accessibility:** add scope attribute to table headers and accessible label for actions column ([#402](https://github.com/amalgamated-tools/biblioteka/issues/402)) ([6e1fe87](https://github.com/amalgamated-tools/biblioteka/commit/6e1fe87d2fea23ae09c5cebf8268f4ed592dc351))
* **accessibility:** associate form validation errors with specific inputs in LibraryForm ([#403](https://github.com/amalgamated-tools/biblioteka/issues/403)) ([9ca05e7](https://github.com/amalgamated-tools/biblioteka/commit/9ca05e756fa6b5c65a079e5591d1661ef840057c))
* **accessibility:** LibraryForm: use fieldset/legend for folder paths group ([#401](https://github.com/amalgamated-tools/biblioteka/issues/401)) ([c885e79](https://github.com/amalgamated-tools/biblioteka/commit/c885e79f9d82a62b5d790b83fd2bab6e3f5226e9))
* **accessibility:** replace sidebar nav buttons with anchor links ([#517](https://github.com/amalgamated-tools/biblioteka/issues/517)) ([f118c63](https://github.com/amalgamated-tools/biblioteka/commit/f118c63af8e84ca7bbe8005fc37b9e619c7081c6))
* **accessibility:** update document.title on SPA navigation (WCAG 2.4.2) ([#341](https://github.com/amalgamated-tools/biblioteka/issues/341)) ([c0c919b](https://github.com/amalgamated-tools/biblioteka/commit/c0c919b02384b35d3574df33e5e896a5b67c2f14))
* accessiblle ([af164f8](https://github.com/amalgamated-tools/biblioteka/commit/af164f84bd0a7491c54b5c03b25a1dd6bfb371fe))
* Add Greptile labeler workflow and clean up code ([#432](https://github.com/amalgamated-tools/biblioteka/issues/432)) ([6592d2d](https://github.com/amalgamated-tools/biblioteka/commit/6592d2d56bad9fd04013c40af07e7bed86a6b4d3))
* **ci:** restore Daily File Diet agent execution ([#265](https://github.com/amalgamated-tools/biblioteka/issues/265)) ([9eff0af](https://github.com/amalgamated-tools/biblioteka/commit/9eff0af02f48e9d6dbf3c09e1bb9d3a7961c709d))
* **code-simplifier:** refactor: simplify kosync handler and system endpoint method checks ([#412](https://github.com/amalgamated-tools/biblioteka/issues/412)) ([d178c86](https://github.com/amalgamated-tools/biblioteka/commit/d178c86438cb54e5f8563614cc3bf9d60fe868af))
* **code-simplifier:** refactor(e2e): simplify auth spec imports and fix comment placement ([#306](https://github.com/amalgamated-tools/biblioteka/issues/306)) ([2789b4d](https://github.com/amalgamated-tools/biblioteka/commit/2789b4da5ae17428bb5af3064b0077f36ed11047))
* **code-simplifier:** refactor(metadata): remove redundant TrimSpace and use t.Context() consistently ([#501](https://github.com/amalgamated-tools/biblioteka/issues/501)) ([03dd3a4](https://github.com/amalgamated-tools/biblioteka/commit/03dd3a4fb99bebfec37955207ee2ced09320aa70))
* **config:** make multi-setting saves atomic ([#319](https://github.com/amalgamated-tools/biblioteka/issues/319)) ([706335f](https://github.com/amalgamated-tools/biblioteka/commit/706335fe6ffb687ec50b2389eb57d06956a09421))
* **frontend:** add skip-to-main-content link to app shell ([#316](https://github.com/amalgamated-tools/biblioteka/issues/316)) ([e706b6a](https://github.com/amalgamated-tools/biblioteka/commit/e706b6a73a7c6699b98595ba2aadc86b65f0e003))
* **frontend:** suppress Svelte state warnings and restore .gitkeep after build ([#285](https://github.com/amalgamated-tools/biblioteka/issues/285)) ([231cb84](https://github.com/amalgamated-tools/biblioteka/commit/231cb84ad9153824504c7ced71f938cf41c35589))
* improve greptile-labeler workflow prompt and suppress transient failure issues ([#435](https://github.com/amalgamated-tools/biblioteka/issues/435)) ([9f9c58d](https://github.com/amalgamated-tools/biblioteka/commit/9f9c58d2c8a3e84c4d3560024238767e6c3d6108))
* install gh-aw extension inline before downloading logs in portfolio-analyst ([#460](https://github.com/amalgamated-tools/biblioteka/issues/460)) ([71a04d9](https://github.com/amalgamated-tools/biblioteka/commit/71a04d9a175f71ba9c1f1db2872d6f691c2207a6))

## [0.0.7](https://github.com/amalgamated-tools/biblioteka/compare/v0.0.6...v0.0.7) (2026-03-17)


### Features

* Add /api/version endpoint and display version in sidebar ([#167](https://github.com/amalgamated-tools/biblioteka/issues/167)) ([9b9b0cc](https://github.com/amalgamated-tools/biblioteka/commit/9b9b0cc7c4061fbbeb0f5b515503509db969c4c6))
* Add `-mode` flag to run HTTP server and worker independently ([#158](https://github.com/amalgamated-tools/biblioteka/issues/158)) ([a87fae8](https://github.com/amalgamated-tools/biblioteka/commit/a87fae89cc315ce666f0aace346bd5495985f4ff))
* add debug logs across the backend for improved observability ([2cdd6a0](https://github.com/amalgamated-tools/biblioteka/commit/2cdd6a0661dc91ffc747526ffda592f5e309211d))
* add in favicon ([#140](https://github.com/amalgamated-tools/biblioteka/issues/140)) ([cb1b874](https://github.com/amalgamated-tools/biblioteka/commit/cb1b874e0c31699cfee37eb122b56a8673348989))
* add library book listing functionality and update UI for viewing libraries ([#77](https://github.com/amalgamated-tools/biblioteka/issues/77)) ([12f439f](https://github.com/amalgamated-tools/biblioteka/commit/12f439fb8262528b674e54896223fcb7df39ae9b))
* add OPDS 1.2 catalog server ([#114](https://github.com/amalgamated-tools/biblioteka/issues/114)) ([1372b33](https://github.com/amalgamated-tools/biblioteka/commit/1372b335dd8c74c1a60c4cddd8f70b250418478c))
* add SMTP configuration and testing functionality ([#110](https://github.com/amalgamated-tools/biblioteka/issues/110)) ([e568195](https://github.com/amalgamated-tools/biblioteka/commit/e56819588ff2c2593ec4e2f4c8dee3e47c34b659))
* audit log for all application changes ([#103](https://github.com/amalgamated-tools/biblioteka/issues/103)) ([dc89d5b](https://github.com/amalgamated-tools/biblioteka/commit/dc89d5b24c28c595527d27d4ff459771e8cce673))
* **auth:** add admin middleware and HttpOnly auth cookies for browser UIs ([#82](https://github.com/amalgamated-tools/biblioteka/issues/82)) ([756b821](https://github.com/amalgamated-tools/biblioteka/commit/756b82129456cae6db6b48e2258c37e7b054b6ce))
* **books:** Add paginated book list view with grid/table toggle ([#224](https://github.com/amalgamated-tools/biblioteka/issues/224)) ([19410bd](https://github.com/amalgamated-tools/biblioteka/commit/19410bd035b648a60c857c84e292cb27a7fa009a))
* Extract metadata from files ([#193](https://github.com/amalgamated-tools/biblioteka/issues/193)) ([0fb9a33](https://github.com/amalgamated-tools/biblioteka/commit/0fb9a3380f91ab4e094d904a813fc4e766a5b2cf))
* implement API key management with create, list, and delete functionalities ([#111](https://github.com/amalgamated-tools/biblioteka/issues/111)) ([1b957eb](https://github.com/amalgamated-tools/biblioteka/commit/1b957eb4ee3efa09db0af4762bbf95ec498fb60d))
* let's process the metadata ([#218](https://github.com/amalgamated-tools/biblioteka/issues/218)) ([b8014d6](https://github.com/amalgamated-tools/biblioteka/commit/b8014d6b7cd15df15937a91690c52927ae88d709))
* normalize book directory structures with path parsing ([#223](https://github.com/amalgamated-tools/biblioteka/issues/223)) ([6ad0d80](https://github.com/amalgamated-tools/biblioteka/commit/6ad0d80d2dabd8d4fb3fc5b3c0d482e6ce4c4e13))
* swagger take 1 ([#61](https://github.com/amalgamated-tools/biblioteka/issues/61)) ([1dc2b02](https://github.com/amalgamated-tools/biblioteka/commit/1dc2b0203618457a0904f54f3149ced6edc867b3))


### Bug Fixes

* **auth:** ensure cookie-backed OIDC sessions survive page reload ([#242](https://github.com/amalgamated-tools/biblioteka/issues/242)) ([4966bee](https://github.com/amalgamated-tools/biblioteka/commit/4966beefb2a454ca4296a78f46331caea247db49))
* correct daily-file-diet prompt to use allowed bash command format ([#187](https://github.com/amalgamated-tools/biblioteka/issues/187)) ([130f82d](https://github.com/amalgamated-tools/biblioteka/commit/130f82d85085840f3a74bd8908d95adbd9c77d68))
* disable provenance attestations to resolve GHCR 403 on push ([#102](https://github.com/amalgamated-tools/biblioteka/issues/102)) ([5c53fdd](https://github.com/amalgamated-tools/biblioteka/commit/5c53fddb6baace00f156703c4635ea27b1646b67))
* go fmt struct field alignment in mockEnqueuer ([f29b659](https://github.com/amalgamated-tools/biblioteka/commit/f29b659d728d4509a720d4fd545eadf3819743b6))
* **libraries:** restrict CRUD to admin users ([#243](https://github.com/amalgamated-tools/biblioteka/issues/243)) ([05193f0](https://github.com/amalgamated-tools/biblioteka/commit/05193f07981fc93ef956fbc56060c60ecc845f53))
* make sure releases work ([#124](https://github.com/amalgamated-tools/biblioteka/issues/124)) ([acf3222](https://github.com/amalgamated-tools/biblioteka/commit/acf32220027acc8f5390df5912147ff9bdf3ee78))
* **oidc:** require email_verified claim before auto-linking ([#244](https://github.com/amalgamated-tools/biblioteka/issues/244)) ([9eecf7f](https://github.com/amalgamated-tools/biblioteka/commit/9eecf7fd3892051b43a82b2bda9371b15e584740))
* remove piped grep command in daily-file-diet prompt to match allowed tool patterns ([#200](https://github.com/amalgamated-tools/biblioteka/issues/200)) ([7733e26](https://github.com/amalgamated-tools/biblioteka/commit/7733e264b2325702ebd35273d955dad227b1302c))
* resolve docker-build workflow GHCR 403 after org transfer ([#150](https://github.com/amalgamated-tools/biblioteka/issues/150)) ([146efd3](https://github.com/amalgamated-tools/biblioteka/commit/146efd39c4a436dee161ac909b64260f7059a7eb))
* use 'latest' copilot CLI version in agentic workflow lock files ([#136](https://github.com/amalgamated-tools/biblioteka/issues/136)) ([6a0a131](https://github.com/amalgamated-tools/biblioteka/commit/6a0a13148949e4969cb124745e3bb3e978727a7a))
* use PAT for CLA Assistant to persist signatures ([#108](https://github.com/amalgamated-tools/biblioteka/issues/108)) ([05653e6](https://github.com/amalgamated-tools/biblioteka/commit/05653e6a3fb2cb4d559a056973732b8d9c39aa27))
* use slog.DebugContext in handlers for request context propagation ([3dbde61](https://github.com/amalgamated-tools/biblioteka/commit/3dbde614eba6d074c11b0526ec1df8c01467c26d))


### Miscellaneous Chores

* release 0.0.2 ([#80](https://github.com/amalgamated-tools/biblioteka/issues/80)) ([b841de8](https://github.com/amalgamated-tools/biblioteka/commit/b841de8eb67d08481d52e392aae6624f4f5c7026))
* release 0.0.3 ([85c0640](https://github.com/amalgamated-tools/biblioteka/commit/85c06409e9fce7c6954188b359f553b0ecfa004a))
* release 0.0.4 ([21507df](https://github.com/amalgamated-tools/biblioteka/commit/21507dfb1fed062dcd20e99e27593ec1b825de5a))
* release 0.0.5 ([cfbf45b](https://github.com/amalgamated-tools/biblioteka/commit/cfbf45b9b224406ddc2f51a23fa009085e083a7e))
* release 0.0.6 ([2fc0dbe](https://github.com/amalgamated-tools/biblioteka/commit/2fc0dbe961f4b65b5affe93c9e98a8e11e88a4d0))
* release 0.0.7 ([bfb147b](https://github.com/amalgamated-tools/biblioteka/commit/bfb147b8c2d44b2b4ee11f566d7031340c021a41))

## [0.0.6](https://github.com/amalgamated-tools/biblioteka/compare/v0.0.5...v0.0.6) (2026-03-17)


### Features

* **books:** Add paginated book list view with grid/table toggle ([#224](https://github.com/amalgamated-tools/biblioteka/issues/224)) ([3264dab](https://github.com/amalgamated-tools/biblioteka/commit/3264dab8ae635626a1b8a375c11f6a031ff08187))
* let's process the metadata ([#218](https://github.com/amalgamated-tools/biblioteka/issues/218)) ([924da03](https://github.com/amalgamated-tools/biblioteka/commit/924da034d0cae9f6dc9ba5fdc606f362e0b1e7e6))
* normalize book directory structures with path parsing ([#223](https://github.com/amalgamated-tools/biblioteka/issues/223)) ([fa13322](https://github.com/amalgamated-tools/biblioteka/commit/fa13322c6efdb58a2bee7f8dc382a19a7f09a3d6))


### Bug Fixes

* **auth:** ensure cookie-backed OIDC sessions survive page reload ([#242](https://github.com/amalgamated-tools/biblioteka/issues/242)) ([7ee6ed4](https://github.com/amalgamated-tools/biblioteka/commit/7ee6ed4627c7829030903e96211c1f8d2a5f70a8))
* **libraries:** restrict CRUD to admin users ([#243](https://github.com/amalgamated-tools/biblioteka/issues/243)) ([45c5552](https://github.com/amalgamated-tools/biblioteka/commit/45c5552dc0267f768ca127ddd55d2fa8240dc484))
* **oidc:** require email_verified claim before auto-linking ([#244](https://github.com/amalgamated-tools/biblioteka/issues/244)) ([d62d9aa](https://github.com/amalgamated-tools/biblioteka/commit/d62d9aaa0798477aab4cacafc583f4c49dc04347))


### Miscellaneous Chores

* release 0.0.6 ([0449dc3](https://github.com/amalgamated-tools/biblioteka/commit/0449dc3ae165b3d01b3e310a4a14d8c830aa2f7c))

## [0.0.5](https://github.com/amalgamated-tools/biblioteka/compare/v0.0.4...v0.0.5) (2026-03-16)


### Features

* Extract metadata from files ([#193](https://github.com/amalgamated-tools/biblioteka/issues/193)) ([6e707f5](https://github.com/amalgamated-tools/biblioteka/commit/6e707f5106527109b8be89bb4081b6cfa9f033d1))


### Bug Fixes

* correct daily-file-diet prompt to use allowed bash command format ([#187](https://github.com/amalgamated-tools/biblioteka/issues/187)) ([672edfb](https://github.com/amalgamated-tools/biblioteka/commit/672edfb3879ca4748427b2b2251b96e41af7f116))
* remove piped grep command in daily-file-diet prompt to match allowed tool patterns ([#200](https://github.com/amalgamated-tools/biblioteka/issues/200)) ([e19b5d0](https://github.com/amalgamated-tools/biblioteka/commit/e19b5d05fce733663cb4964a700c916fb7acfaf3))


### Miscellaneous Chores

* release 0.0.5 ([468ec08](https://github.com/amalgamated-tools/biblioteka/commit/468ec08be4f8dacb373871e21c80b2f966a3bf41))

## [0.0.4](https://github.com/amalgamated-tools/biblioteka/compare/v0.0.3...v0.0.4) (2026-03-15)


### Features

* Add /api/version endpoint and display version in sidebar ([#167](https://github.com/amalgamated-tools/biblioteka/issues/167)) ([a0f34f5](https://github.com/amalgamated-tools/biblioteka/commit/a0f34f5c7b9ed08b3f8bb2c9750dd5018cad2a22))
* Add `-mode` flag to run HTTP server and worker independently ([#158](https://github.com/amalgamated-tools/biblioteka/issues/158)) ([2604c5d](https://github.com/amalgamated-tools/biblioteka/commit/2604c5da00731f09779b91693393e4b49f86beca))
* add in favicon ([#140](https://github.com/amalgamated-tools/biblioteka/issues/140)) ([2cb8457](https://github.com/amalgamated-tools/biblioteka/commit/2cb845795d6afcb1df51216b077b03b441180e85))
* add OPDS 1.2 catalog server ([#114](https://github.com/amalgamated-tools/biblioteka/issues/114)) ([ccc20a9](https://github.com/amalgamated-tools/biblioteka/commit/ccc20a9276bc27193e0264671f969f8f3dacb7c7))
* add SMTP configuration and testing functionality ([#110](https://github.com/amalgamated-tools/biblioteka/issues/110)) ([b7adf65](https://github.com/amalgamated-tools/biblioteka/commit/b7adf65d8447c4c1dbcbc777346bbf182eff44bb))
* audit log for all application changes ([#103](https://github.com/amalgamated-tools/biblioteka/issues/103)) ([1bd9c15](https://github.com/amalgamated-tools/biblioteka/commit/1bd9c15eb07c28448e2d2b1fbad1fe359995b3ff))
* **auth:** add admin middleware and HttpOnly auth cookies for browser UIs ([#82](https://github.com/amalgamated-tools/biblioteka/issues/82)) ([01428ea](https://github.com/amalgamated-tools/biblioteka/commit/01428ea53c59027f2055af751083a8129759bed4))
* implement API key management with create, list, and delete functionalities ([#111](https://github.com/amalgamated-tools/biblioteka/issues/111)) ([38bc54d](https://github.com/amalgamated-tools/biblioteka/commit/38bc54d6ce3f41639bb0e04853190e0f94335ad9))


### Bug Fixes

* disable provenance attestations to resolve GHCR 403 on push ([#102](https://github.com/amalgamated-tools/biblioteka/issues/102)) ([5d12388](https://github.com/amalgamated-tools/biblioteka/commit/5d12388a2f3981a31460f399e0f548c922afb6ae))
* make sure releases work ([#124](https://github.com/amalgamated-tools/biblioteka/issues/124)) ([fcf4e12](https://github.com/amalgamated-tools/biblioteka/commit/fcf4e122476b48141d81df1704a1e872c41bc237))
* resolve docker-build workflow GHCR 403 after org transfer ([#150](https://github.com/amalgamated-tools/biblioteka/issues/150)) ([08b1f11](https://github.com/amalgamated-tools/biblioteka/commit/08b1f11ca9f4eb128b7471ead4d8ba2df5ca8c19))
* use 'latest' copilot CLI version in agentic workflow lock files ([#136](https://github.com/amalgamated-tools/biblioteka/issues/136)) ([e780ca5](https://github.com/amalgamated-tools/biblioteka/commit/e780ca56e3a92be36a635bbbc1b0f29569043ebb))
* use PAT for CLA Assistant to persist signatures ([#108](https://github.com/amalgamated-tools/biblioteka/issues/108)) ([8e86f1d](https://github.com/amalgamated-tools/biblioteka/commit/8e86f1da84ab17221b9f7890ae96a456bb5282eb))


### Miscellaneous Chores

* release 0.0.4 ([42a07e8](https://github.com/amalgamated-tools/biblioteka/commit/42a07e82d5fce9c4c5fa34055eb110da67c939eb))

## [0.0.3](https://github.com/amalgamated-tools/biblioteka/compare/v0.0.2...v0.0.3) (2026-03-14)


### Features

* add library book listing functionality and update UI for viewing libraries ([#77](https://github.com/amalgamated-tools/biblioteka/issues/77)) ([9cdb820](https://github.com/amalgamated-tools/biblioteka/commit/9cdb820d34413a1a87c7e7e288b3e0b0f0815c84))


### Miscellaneous Chores

* release 0.0.3 ([be52686](https://github.com/amalgamated-tools/biblioteka/commit/be52686964de075f56644257b4a71bd53be97f25))

## 0.0.2 (2026-03-14)


### Features

* add debug logs across the backend for improved observability ([36b2b7d](https://github.com/amalgamated-tools/biblioteka/commit/36b2b7d863776dff1e1c82c575976433898cee41))
* swagger take 1 ([#61](https://github.com/amalgamated-tools/biblioteka/issues/61)) ([77cf28b](https://github.com/amalgamated-tools/biblioteka/commit/77cf28b19aa3d99a23012d2dddb18a3d6ab20d6c))


### Bug Fixes

* go fmt struct field alignment in mockEnqueuer ([b10e6a2](https://github.com/amalgamated-tools/biblioteka/commit/b10e6a2ecea73e4f8ede7ff689169045a434331f))
* use slog.DebugContext in handlers for request context propagation ([25136a5](https://github.com/amalgamated-tools/biblioteka/commit/25136a556654238ca5382edfc4747d69d5b6c73a))


### Miscellaneous Chores

* release 0.0.2 ([#80](https://github.com/amalgamated-tools/biblioteka/issues/80)) ([3980217](https://github.com/amalgamated-tools/biblioteka/commit/3980217673949fa0fc777615684dac77820a4bae))
