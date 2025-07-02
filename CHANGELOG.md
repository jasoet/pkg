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
