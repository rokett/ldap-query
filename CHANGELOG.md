# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [1.2.1] - 2021/10/28
### Fixed
- Added missing import package to `main_linux.go`.

## [1.2.0] - 2021/10/28
### Added
- #9 Added CORS options to allow for lookups from websites.

## [1.1.0] - 2021/02/12
### Changed
- #5 Changed search to do a paged search in order to ensure we can return more than 1000 records in a single search.

## [1.0.1] - 2021/02/11
### Fixed
- Resolved an issue whereby the docker image was unable to be built because it was pointing to an old, pre Go module, release and was not copying the executable properly.

### Changed
- Added query details to logging if the query fails.

## [1.0.0] - 2020/05/08
Initial release
