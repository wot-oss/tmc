# Change Log

## [Unreleased]

### Added

- Implemented file locking to avoid data races (guaranteed for Linux/MacOSX)

### Changed

- ```create-toc```: renamed to ```update-toc``` and allow for partial updates
- ```serve```: separate configuration of the remote(s) to be served from the target remote for push  

### Fixed

- count only enabled remotes when checking if empty remote specification is unambiguous

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
