## [1.3.0](https://github.com/jasoet/pkg/compare/v1.2.2...v1.3.0) (2025-07-27)

### Features

* **security:** add comprehensive security enhancements to compress and ssh packages ([2f01a6a](https://github.com/jasoet/pkg/commit/2f01a6ab03e0b3b33be8ba7b427ecc26242c28cb))

## [1.2.2](https://github.com/jasoet/pkg/compare/v1.2.1...v1.2.2) (2025-07-27)

### Bug Fixes

* improve documentation structure and build system ([706698f](https://github.com/jasoet/pkg/commit/706698fdc052c669adbfcc4f56a3e4c38e38c683))

### Documentation

* **claude:** enhance CLAUDE.md with comprehensive project information ([532ea9c](https://github.com/jasoet/pkg/commit/532ea9cd78a119a2bc41897053a58af0f6b71804))

### Continuous Integration

* add Claude code review workflow for automated PR reviews ([0f856d1](https://github.com/jasoet/pkg/commit/0f856d113680db4ede71f0ee5770a232efa4aabe))

## [1.2.1](https://github.com/jasoet/pkg/compare/v1.2.0...v1.2.1) (2025-07-02)

### Chores

* **tests:** remove redundant test cases and improve order failure simulation ([48c7eae](https://github.com/jasoet/pkg/commit/48c7eae3bb9226c44e6126c084c918349b9293e0))

## [1.2.0](https://github.com/jasoet/pkg/compare/v1.1.4...v1.2.0) (2025-07-02)

### Features

* **temporal:** add Docker Compose services for Temporal and enhance integration tests ([d770acb](https://github.com/jasoet/pkg/commit/d770acb7a63d0e98e6542b57f08cfbc90d96cb82))

### Documentation

* **tests:** add comprehensive testing guide and refine integration test execution ([1e358e6](https://github.com/jasoet/pkg/commit/1e358e661f4952fecb24a800f13e6bb7c34a506e))

## [1.1.4](https://github.com/jasoet/pkg/compare/v1.1.3...v1.1.4) (2025-07-02)

### Chores

* **tests:** update Temporal integration tests with consistent naming and improved task queue checks ([add78f0](https://github.com/jasoet/pkg/commit/add78f033fa9b4d50e6ed2815255926f088300a3))

## [1.1.3](https://github.com/jasoet/pkg/compare/v1.1.2...v1.1.3) (2025-07-02)

### Chores

* streamline examples and middleware, add build tags, and refactor logs ([abc234e](https://github.com/jasoet/pkg/commit/abc234e346e528d7d70bf06ed4e7e27e5c87c587))
* **tests:** add comprehensive integration and end-to-end tests for Temporal package ([e370655](https://github.com/jasoet/pkg/commit/e37065516d5ddfef630c7e30b63de3a7d8009ba7))

## [1.1.2](https://github.com/jasoet/pkg/compare/v1.1.1...v1.1.2) (2025-07-02)

### Chores

* remove Gitpod configuration and unused server examples ([7578588](https://github.com/jasoet/pkg/commit/757858844f4b3ad4a8bd3da492493202badfa98f))

## [1.1.1](https://github.com/jasoet/pkg/compare/v1.1.0...v1.1.1) (2025-07-02)

### Documentation

* add `CLAUDE.md` with development guidelines and architecture overview ([1d73b75](https://github.com/jasoet/pkg/commit/1d73b75b67d30166afdc7f07a4eb441761054e56))

### Chores

* introduce initial project templates, dev container, and Gitpod setup ([35278bf](https://github.com/jasoet/pkg/commit/35278bfd2ea24441e2fbb9a12cfb43ae3f54fbd8))

## [1.1.0](https://github.com/jasoet/pkg/compare/v1.0.0...v1.1.0) (2025-06-05)

### Features

* **config:** add support for string slice configurations with environment variable overrides ([184c527](https://github.com/jasoet/pkg/commit/184c527acf078e4b58fdaf2c5d69752e6b981bd2))

## 1.0.0 (2025-06-04)

### Features

* integrate semantic-release for automated versioning and changelog generation ([d62c61e](https://github.com/jasoet/pkg/commit/d62c61e5362b3432cc081735b9e6f89e76882548))

# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

This file will be automatically updated by semantic-release based on commit messages following the [Conventional Commits](https://www.conventionalcommits.org/) specification.

## [1.0.0] - YYYY-MM-DD

### Features

- Initial release of the Go utility packages
- Added logging package with zerolog integration
- Added database connection utilities for MySQL, PostgreSQL, and SQL Server
- Added HTTP client utilities with retry mechanisms
- Added HTTP server utilities using Echo framework
- Added SSH tunnel utilities
- Added concurrent execution utilities
- Added Temporal workflow utilities
