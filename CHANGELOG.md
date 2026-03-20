# Changelog

## [1.3.1](https://github.com/morganBlanloeil/c9e/compare/v1.3.0...v1.3.1) (2026-03-20)


### Bug Fixes

* pre-pad column values before ANSI styling to fix TUI alignment ([58093a9](https://github.com/morganBlanloeil/c9e/commit/58093a954a6e4c25b7346f8f4389bdcde8b929b6))


### Documentation

* update README with missing features and mise install ([#20](https://github.com/morganBlanloeil/c9e/issues/20)) ([4248320](https://github.com/morganBlanloeil/c9e/commit/424832035406cc68d3a2b2f631d84baa719f9634))

## [1.3.0](https://github.com/morganBlanloeil/c9e/compare/v1.2.1...v1.3.0) (2026-03-20)


### Features

* add cost estimation per session based on token usage ([3b819ef](https://github.com/morganBlanloeil/c9e/commit/3b819efd758b8f7cb1962b02abf0916ac9d8b15c))
* add desktop notifications for task completion ([b000c9e](https://github.com/morganBlanloeil/c9e/commit/b000c9e9c093e2f2c512a29afeed250420df035c))
* add quick wins - column sorting, turn counter, aggregate stats, copy CWD ([1777135](https://github.com/morganBlanloeil/c9e/commit/177713526ec5fd947fc662c3bd67e8975a1f9a9d))
* add Stop hook to auto-update documentation after sessions ([ac49cfc](https://github.com/morganBlanloeil/c9e/commit/ac49cfc6bb2b35ae4c9165696ef88296b8920bc9))


### Miscellaneous

* update gitignore with claude artifacts and build binary ([dd1c7ef](https://github.com/morganBlanloeil/c9e/commit/dd1c7ef88bdef9c7ff56bc583b05012acd4c24a6))

## [1.2.1](https://github.com/morganBlanloeil/c9e/compare/v1.2.0...v1.2.1) (2026-03-20)


### CI

* include ci/docs/chore commits in release-please changelog ([83ea857](https://github.com/morganBlanloeil/c9e/commit/83ea857335b343d0a2e3213da6f68e902f1d01cc))
* wait for CI workflow to succeed before running release-please ([c9471b8](https://github.com/morganBlanloeil/c9e/commit/c9471b86c5202640524f4fae5ec54e50589022ef))


### Documentation

* orient install docs toward GitHub releases and auto-update version ([e03db30](https://github.com/morganBlanloeil/c9e/commit/e03db304eb726c973ab6ba4f0f04aaee6dcfeebf))
* remove git clone install method, keep go install as alternative ([42c92ac](https://github.com/morganBlanloeil/c9e/commit/42c92ac09d8207713917e77fde33d5f60af5b369))


### Miscellaneous

* trigger release-please ([78e9223](https://github.com/morganBlanloeil/c9e/commit/78e9223ad55db125b0e60dc974690b3e686b654c))

## [1.2.0](https://github.com/morganBlanloeil/c9e/compare/v1.1.0...v1.2.0) (2026-03-20)


### Features

* add binary download docs, compilation hook, and custom skills ([#5](https://github.com/morganBlanloeil/c9e/issues/5)) ([ba76e63](https://github.com/morganBlanloeil/c9e/commit/ba76e63b717a89791a829070da0e5c4fec51bd8d))
* highlight finished Claude Code sessions with visual indicator ([11f8efa](https://github.com/morganBlanloeil/c9e/commit/11f8efafff9f73731c4fcdcb2d734630e4b7b774))

## [1.1.0](https://github.com/morganBlanloeil/c9e/compare/v1.0.1...v1.1.0) (2026-03-20)


### Features

* integrate binary build into release-please workflow ([#3](https://github.com/morganBlanloeil/c9e/issues/3)) ([930321c](https://github.com/morganBlanloeil/c9e/commit/930321cb2f1d1e3ae45339e07a0174bccaa15f82))

## [1.0.1](https://github.com/morganBlanloeil/c9e/compare/v1.0.0...v1.0.1) (2026-03-20)


### Bug Fixes

* show only directory basename in list view ([3fef506](https://github.com/morganBlanloeil/c9e/commit/3fef50692d701a4b9d95271b8d19d786c74e0a10))
* show only directory basename in list view, full path in detail ([415d92e](https://github.com/morganBlanloeil/c9e/commit/415d92eedf2989450415001b25fb7d66c37e958d))

## Changelog

All notable changes to this project will be automatically documented in this file
by [semantic-release](https://github.com/semantic-release/semantic-release).

This project adheres to [Semantic Versioning](https://semver.org/).
