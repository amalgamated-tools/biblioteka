# Changelog

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
