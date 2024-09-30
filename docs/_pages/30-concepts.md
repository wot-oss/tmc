---
layout: post
permalink: /concepts
title: Concepts
---

## Glossary

A short glossary of concepts and abbreviations used throughout Thing Model Catalog documentation and code

**TM** - Thing Model

**TM name** - a path-like string identifying a set of different versions of TMs for the same device from the same author. The first three path segments are always the author, manufacturer and mpn (manufacturer part number) has the format of `<author>/<manufacturer>/<mpn>[/<optional-extra-path-segments>`

**TM version** - a single specific TM in context of the set of TMs with the same TM name. Also, the string identifying this TM, i.e. the last segment of the TM's ID. See [TM IDs](#tm-ids-and-structure-of-repositories) 

**Catalog**, **TM Catalog**, **TMC** - a set of TMs accessible via a single entry point: CLI or REST API

**Named Repository** - a Repository, the configuration of which is stored in a config file

**Local Repository** - an "ad-hoc" unnamed Repository that is stored in a local directory, which is passed to the TMC 
Binary on the command line

## Working With Repositories

Most of the subcommands of `tmc` need to know which repository or repositories to operate on. There are three possible 
cases of how the repos could be defined.

### 1. A Single Repository Is Stored in the Config File 
When the config file contains only one repository, or it contains multiple repositories, but only one of them is enabled, 
then a command operates on this one repository. See `tmc help repo`.

### 2. A Local Repository Is Defined by a Flag
If you pass a directory name via the `--directory` flag, then that directory is used as a storage location for an 
unnamed local repository, which is then used by the command.
When a repository is passed via this flag, any repository configuration in the config file is disregarded, even if the 
same directory is stored as a named repository there. 

### 3. Multiple Repositories Are Stored in the Config File
When there are multiple repositories defined in the config file, then, depending on the command, it either uses the entire 
list of repositories to operate on (e.g. `tmc list`), or it will require that you specify one of the
repositories by name using the `--repo` flag (e.g. `tmc import`).
You can still constrain the commands like `tmc list` or `tmc versions` to operate on a single repository using the same 
`--repo` flag even though it's not mandatory.


## TM IDs and Structure of Repositories

When you import TMs into a repository, they are given a generated ID, which is based on the key fields, optional 
additional path (see `tmc help import`), timestamp, and a hash of the TM contents. See the [proposal][1] for details and 
the reasoning behind.
If a TM already has an ID when it's imported, the original ID will be moved to a field 'externalID' and can be restored 
when the TM is fetched or exported from TMC. See `tmc help fetch` and `tmc help export`.

The IDs given to TMs define the storage structure of the files in file-based repositories.

Do *not* change the TM files inside the repository's directory structure except via TMC Binary, for this will cause the
index files to be out of sync. You can check if the index corresponds to the contents of a repository with `tmc check`.
To repair a broken index, use `tmc index`.


[1]: https://github.com/wot-oss/proposal/issues/10