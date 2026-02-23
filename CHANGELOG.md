# Change Log

## [unreleased]

### Added

- REST API: jwt authentication based on JWT scopes array
- REST API: jwt authentication default scopes for each token can be defined in a separate file using `--defaultScopesPath` flag
- REST API: pagination for inventory
- REST API: POST `/repos/export` creates a zip for the specified repository
- REST API: GET `/repos/export` to download archived repo
- Added `filter.latest` parameter to REST API `/inventory` listing
- generating `manufacturers.txt` and `mpns.txt` for improving static hosting (e.g. github, github pages)
- `jwtScopesPrefix` flag to set default prefix for scopes authentication

### Changed

- default TmcVersion is set to `dev`
- 
### Fixed

### Removed

## [v0.1.4]

### Added

- REST API: `/repos` returns directory name, when serving a catalog with `--directory` flag
- `import`: added flag `--with-attachments` to import the attachments along with TMs
- REST API: added reference to RFC 7807 and code key explanation for the error response
- Docs: explain sanitization rules for `manufacturer`, `author`, and `mpn`

### Fixed

- avoid escaping special characters when marshaling json

## [v0.1.3]

### Added

- `copy`: added flag `--ignore-existing` to ignore TMs and attachments that have conflicts with existing ones instead of returning an error code
- added possibility to import file attachments to TMs and TM names
- `export`: added flag to export attachments together with TMs
- added setting/storing/detecting of attachment media types
- `check`: added `.tmc/.tmcignore` file to explicitly exclude files from being validated by `check`
- added HTTP Basic auth to tmc and file repos
- `repo add`, `repo config set`: added flag to pass repo config json as string
- added possibility to define configuration parameters of repositories by referencing environment variables
- added flag to change commands' output format to JSON
- `list`, `copy`, `export`: added filtering by protocols supported by TMs
- added `create-si` command to initially create a `bleve` search index for repositories
- added `search` command to search for TMs by a text search query using `bleve` syntax
- added a `s3` (AWS S3) repository type
- added a check for directory or repo before serving
- added importing attachments in a directory of TMs when importing TMs

### Changed

- return error on attachment import when it already exists and add a flag to override
- `check`: removed subcommands of `check` command, unifying both into the parent command
- `repo`: reorganized commands that change repo config: renamed/created `config auth`, `config description`, and `config headers` commands
- `list`, `copy`, `export`: removed text search query parameter
- removed `search` parameter from `/authors`, `/manufacturers`, and `/mpns` API endpoints
- `/inventory`: make `search` parameter use `bleve` query syntax and make it mutually exclusive with filter parameters

### Fixed

- return `"application/tm+json"` as MIME type when fetching TMs via API

## [v0.1.2]

### Removed

- Dockerfile: removed creation of a named catalog in the docker image

## [v0.1.1]

### Added

- `import`: added flag `--ignore-existing` to ignore TMs that have conflicts with existing TMs instead of returning an error code

## [v0.1.0]

### Added

- REST API: added query parameter `force` to POST endpoint `/thing-models` to enforce pushing TMs with same TM name, semantic version and digest
- REST API: added query parameter `optPath` to POST endpoint `/thing-models` to append optional path parts to the target path (and id)
- added command `copy` to copy TMs between repositories
- REST API: added GET endpoint `/repos` to list available repositories
- `repo`: added flag for setting a repository description when adding/changing repo config
- REST API: added source repository name to inventory responses

### Changed

- `push`: renamed `push` to `import`
- `pull`: renamed `pull` to `export`
- `push`: always reject pushing TMs with same TM name, semantic version and digest by default, but can be enforced by flag `--force`
- `push`: shows a warning if there is a timestamp collision (retrying after a second has been removed)
- `list, pull, versions`: return exit code 1 if at least one repo returns an error
- `check index`: do not return error if repo does not contain any TM's and index

### Fixed

- `import`: restore the printout of error to stdout when import was not successful

## [v0.0.0-alpha.7]

### Added

- Added verb `check` with sub-commands `index` and `resources`
- `check index`: validates the index wrt stored Thing Models
- `check resources`: validates stored Thing Models (path, name, syntax etc.)
- Put a limit on importable TM name length at 255 characters

### Changed

- Removed the concept of official TMs
- Force all TM ids and key fields in imported TMs to be sanitized and lower case

## [v0.0.0-alpha.6]

### Changed

- Renamed the go module to `github.com/wot-oss/tmc`
- Renamed command `remote` to `repo`
- Renamed command `update-toc` to `index`
- Tab-completions now only complete a path segment instead of the full name to resemble shell completion in a file system
- Removed mockery dependency from final binary

## [v0.0.0-alpha.5]

### Added

- Implemented REST API authentication with JWT tokens and JWKS validation
- Implemented `version` command to show the version of the tm-catalog-cli
- Implemented autocompletion for most flags and arguments for the shell autocompletion script
- Added an optional flag to fetch to restore original external id to the fetched TM

### Changed

- Request results from multiple remotes concurrently instead of sequentially

### Fixed

- handle timestamp collisions on push by retrying after one second, forcing generation of new id, or reporting the error if all else fails
- Display errors when accessing remotes for list/versions instead of silently ignoring them

## [v0.0.0-alpha.4]

### Added

- REST API: added `meta.page.elements` to inventory response, reflecting number of entries in current result page

### Changed

- REST API: renamed inventory endpoint `/versions` to `/.versions`
- REST API: removed `meta.created` from inventory response
- Removed '--exact' flag to `list` and `pull`
- `list` and `pull`: match given name pattern as a prefix by complete path parts
- `list`: changed output format: put NAME column first, renamed PATH column to MPN
- `versions`: changed output format: renamed PATH column to ID

## [v0.0.0-alpha.3]

### Added

- Building docker base image for releases to enable catalog hosting
- Implemented file locking to avoid data races (guaranteed for Linux/MacOSX)
- Implemented `pull` command to fetch and save multiple thing models at once
- Implemented setting CORS options for API
- Implemented fetching a TM by a \[partial\] semantic version also in REST API
- Print information about used config file in `help`
- Implemented a `tmc` remote type, which uses our own REST API as the underlying TM storage
- Added `filter.name` parameter to REST API `/inventory` listing
- Added `--exact` flag to `list` and `pull`

### Changed

- `create-toc`: renamed to `update-toc` and allow for partial updates
- `list`: allows now listing by name pattern
- `serve`: separate configuration of the remote(s) to be served from the target remote for push
- `fetch`: `--output` now accepts only a target folder to save TM to, `--with-path` has been removed
- `list, pull`: removed filter flag `filter.externalID`, search for externalID has now to be done by query search `-s`
- REST API:  removed filter parameter filter.externalID from `/inventory`, `/authors`, `/manufacturers`, `/mpns`,
  search for externalID has now to be done by query parameter `search`
- enable/disable logging is now done only by setting a loglevel

### Fixed

- count only enabled remotes when checking if empty remote specification is unambiguous
- make fetch by partial semantic version match the most recent version beginning with given string
- (BREAKING!) `push`: file hash calculation has been made more reliable and idempotent. Consequently, some files if pushed to TMC, may receive a new version hash, despite no change in contents
- `fetch`: fixed `"Unable to parse TMID..."` error when fetching an official TM by content hash
- prevent `serve` from using one of remotes from config as push target when `-r` or `-d` are given
- print the actual error if updating TOC after `push` fails

## [v0.0.0-alpha.2] - 2023-01-15

### Fixed

- config is now created if not existing
- Adding `.exe` to Windows binaries

## [v0.0.0-alpha.1] - 2023-01-15

This the first alpha release, which implements the most basic verbs to create and interact with a thing model catalog.

### Added

- Verbs: create-toc, fetch, list, push, remote, serve, validate, versions
- Target local catalogs with the `--directory` flag
- fetch can now create a file instead of printing to stdout
- serve now exposes a REST API

See README.md for a description of all current features.
