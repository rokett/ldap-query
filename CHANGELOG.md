# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2021/02/11
### Fixed
- Resolved an issue whereby the docker image was unable to be built because it was pointing to an old, pre Go module, release and was not copying the executable properly.

### Changed
- Added query details to logging if the query fails.

## [1.0.0] - 2020/05/08
Initial release
