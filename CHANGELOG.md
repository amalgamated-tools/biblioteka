# Changelog

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
