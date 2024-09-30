---
layout: post
permalink: /workflows
title: Workflows
---

Below you can find some typical workflows for using the TMC and example command lines. Refer to the detailed documentation of each
command for a complete list of available flags and arguments.

```bash
tmc <command> --help
```

## Import a TM or a folder with multiple TMs

To be imported into a catalog, a TM must be valid according to the [W3C Thing Model schema][1]. In addition to that,
some minimal key fields defined by [schema.org][2] are required.
The fields are:
- `schema:author/schema:name` (https://schema.org/author)
- `schema:manufacturer/schema:name` (https://schema.org/manufacturer)
- `schema:mpn` (https://schema.org/mpn)

These fields together build the mandatory part of TM name.

You may also want to set the field `version/model` (https://www.w3.org/TR/wot-thing-description11/#versioninfo) to track 
and communicate the extent of changes between versions of TMs with the same TM name.

You can check if a TM can be imported by validating it beforehand:
```bash
tmc validate my-tm.json 
```

Import a TM or a folder with multiple TMs into the catalog:
```bash
tmc import my-tm.json 
```

## Find and fetch a TM

```bash
tmc list --filter.mpn poc1000
tmc fetch siemens/siemens/poc1000/v1.0.1-20240407094932-5a3840060b05.tm.json
```

You can fetch a specific version of a TM by fetching by ID as above, or you can fetch the latest TM that matches a given name and, optionally, a part of semantic version.   
Examples:
```bash
tmc fetch siemens/siemens/poc1000
tmc fetch siemens/siemens/poc1000:v1
tmc fetch siemens/siemens/poc1000:v1.0.1
```

## Set Up a List of TM Repositories

Most subcommands of `tmc` operate by default on the list of named repositories stored in its `config.json` file, unless a single one of them is selected
by `--repo` flag or a local repository is defined by `--directory` flag. The default location of the `config.json` file is `~/.tm-catalog`. You can 
override the default config directory with the `--config` flag.

To view and modify the list of repositories in the config file, use command `repo` and its subcommands. For example:
```bash
tmc repo list
tmc repo add --type file my-catalog ~/tm-catalog
tmc repo show my-catalog
tmc repo toggle-enabled my-catalog
tmc repo remove my-catalog
```

You can also use any directory as a storage space for an unnamed local repository. You will need to pass a `--directory` flag to most commands.

## Expose a Catalog for HTTP Clients

To expose a catalog over HTTP, start a server:
```bash
tmc serve
```
An OpenAPI description of the API [is available][4] for ease of integration. (Raw source [here][3].)


[1]: https://github.com/w3c/wot-thing-description/blob/main/validation/tm-json-schema-validation.json
[2]: https://schema.org
[3]: https://github.com/wot-oss/tmc/blob/main/api/tm-catalog.openapi.yaml
[4]: https://editor.swagger.io/?url=https://raw.githubusercontent.com/wot-oss/tmc/refs/heads/main/api/tm-catalog.openapi.yaml

