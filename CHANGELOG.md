# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Fixed

## [0.0.3]

### Changed
- Renamed project from gevals to mcpchecker, migrated to mcpchecker github org

## [0.0.2]

### Added
- Extension support with Go extension SDK (#79)
- Gemini agent support (#69)
- Builtin steps for task execution (#56)
- View command for eval results (#36)
- Functional test framework (#71)
- Dependabot for automated dependency updates (#64)

### Changed
- Updated modelcontextprotocol/go-sdk to v1.2.0 (#75)
- Updated Go version to 1.25.x in GitHub Action (#80)
- Bumped actions/checkout from v5 to v6 (#66, #74)
- Bumped actions/upload-artifact from v5 to v6 (#68)

### Fixed
- Action correctly picks up pinned version when set (#81)
- Race conditions in mcpproxy (#70)

## [0.0.1]

### Added
- Initial release of gevals
- MCP server evaluation framework
- Support for multiple agent types (Claude Code, OpenAI)
- Kubernetes MCP server examples
- LLM judge for evaluating responses
- Release workflows for automated publishing
- GitHub Action for running gevals evaluations
- Support for nightly releases
