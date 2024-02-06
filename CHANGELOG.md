# Change Log

## [v0.0.0-alpha.3]

### Added

- Implemented file locking to avoid data races (guaranteed for Linux/MacOSX)
- Implemented ```pull``` command to fetch and save multiple thing models at once
- Implemented setting CORS options for API
- Implemented fetching a TM by a \[partial\] semantic version also in REST API
- Print information about used config file in `help`
- Implemented a 'tmc' remote type, which uses our own REST API as the underlying TM storage
- Added 'filter.name' parameter to REST API '/inventory' listing
- Added '--exact' flag to `list` and `pull`

### Changed

- ```create-toc```: renamed to ```update-toc``` and allow for partial updates
- ```list```: allows now listing by name pattern
- ```serve```: separate configuration of the remote(s) to be served from the target remote for push
- ```fetch```: ```--output``` now accepts only a target folder to save TM to, ```--with-path``` has been removed
- enable/disable logging is now done only by setting a loglevel

### Fixed

- count only enabled remotes when checking if empty remote specification is unambiguous
- make fetch by partial semantic version match the most recent version beginning with given string
- (BREAKING!) ```push```: file hash calculation has been made more reliable and idempotent. Consequently, some files if pushed to TMC, may receive a new version hash, despite no change in contents
- ```fetch```: fixed "Unable to parse TMID..." error when fetching an official TM by content hash
- prevent ```serve``` from using one of remotes from config as push target when '-r' or '-d' are given
- print the actual error if updating TOC after ```push``` fails

## [v0.0.0-alpha.2] - 2023-01-15

### Fixed

- config is now created if not existing
- Adding ".exe" to Windows binaries 


## [v0.0.0-alpha.1] - 2023-01-15

This the first alpha release, which implements the most basic verbs to create and interact with a thing model catalog. 

### Added

- Verbs: create-toc, fetch, list, push, remote, serve, validate, versions 
- Target local catalogs with the '--directory' flag
- fetch can now create a file instead of printing to stdout 
- serve now exposes a REST API

See README.md for a description of all current features.
