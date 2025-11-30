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

## Create a Repository

To create an empty repository named 'my-catalog' in the folder 'tm-catalog' under your user home directory, execute

```bash
tmc repo add --type file my-catalog ~/tm-catalog
```

If the directory does not exist, it will be created when you import the first TM into the repository.

See [`repo add`][5] for more details on how to create repositories.

## Manage The List of Repositories

Most subcommands of `tmc` operate by default on the list of named repositories stored in its `config.json` file, unless a single one of them is selected
by `--repo` flag or a local repository is defined by `--directory` flag. The default location of the `config.json` file is `~/.tm-catalog`. You can
override the default config directory with the `--config` flag. There must be at least one repository to serve specified in `config.json` file or with the `--repo` or `--directory` flags.

To view and modify the list of repositories in the config file, use command `repo` and its subcommands. For example,

```bash
tmc repo list
tmc repo show my-catalog
tmc repo toggle-enaled my-catalog
tmc repo remove my-catalog
```

You can also use any directory as a storage space for an unnamed local repository. You will need to pass a `--directory` flag to most commands.

## Import Thing Models

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
tmc import ./my-tms
```

### Attachments

When importing a folder, the `import` command can be used with the `--with-attachments` flag to import attachments along with the TMs. An attachment is linked to a TM by placing it into a subfolder whose name exactly matches the TM's filename (including its extension). 
For example:
    *   If your TM file is: `../example-catalog/.tmc/omniuser/omnicorp/senseall/v1.0.0-20241008124326-15af48381cf7.tm.json`
    *   Then an attachment (e.g., `readme.md`) for this TM would be placed at: `../example-catalog/.tmc/omniuser/omnicorp/senseall/.attachments/v1.0.0-20241008124326-15af48381cf7.tm.json/readme.md`

### Input Sanitization

Please pay attention to the values of `manufacturer`, `author`, and `mpn` as they will be sanitized following the rules below:

- spaces will be removed
- all letters will become lowercase
- characters below will be replaced with `-`:
  - `_`
  - `+`
  - `&`
  - `=`
  - `:`
  - `/`
- all characters with an accent will be replaced with their versions without an accent, e.g., `ö` will become `oe`, `à` will become `a`.

In the end, there will be only letters, numbers and `-` remaining.
You can refer to `SanitizeName` function at [internal/utils/util.go](https://github.com/wot-oss/tmc/blob/main/internal/utils/util.go).

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

## Publish a Catalog to a Git Forge

Initialize the directory where your file repository is located as a git repository and use the git workflows to commit and push it to
your git forge, like GitHub or GitLab.

You may want to add the `*.lock` files to your `.gitignore`, but it's not mandatory
```bash
echo "*.lock" >> .gitignore
```

To use the published catalog as a "http" repository on the consumer side, you have to use the URL, under which all files
of the git repository can be retrieved by HTTP GET request by either appending their relative paths to the URL or
substituting a placeholder with the relative path. This URL will differ between different git forges.
For example, for GitHub, it has this form ```https://raw.githubusercontent.com/<group>/<repository>/refs/heads/main```,
where you should substitute `<group>` and `<repository>` with actual names. For GitLab, you can use the REST API
endpoint
`https://gitlab.example.com/api/v4/projects/<project-id>/repository/files/{% raw %}{{ID}}{% endraw %}?ref=main`, where
you replace `<project-id>` with the numeric id of the GitLab project.

If the git repository is private, an access token needs to be configured using ```tmc remote set-auth```

This method has the advantage that the infrastructure of the forges is used and no custom infrastructure needs to be
maintained by the creator. The downside is that any contribution has to go through a git workflow, which might not be an
accessible option for system integrators. In addition, products will most likely want to deploy a private catalog with a
curated list of TMs, without relying on a forge.

## Expose a Catalog for HTTP Clients

To expose a catalog over HTTP, start a server:
```bash
tmc serve
```
An OpenAPI description of the API [is available][3] for ease of integration.

Once a catalog is exposed with `tmc serve`, it can be configured as a repository of type 'tmc' on other clients. Users
can push to a hosted catalog using the REST API, without using git workflow and hosting can happen on the edge within a
product.

To make things easier, we build a ```tmc``` [container image][4] which runs the cli as a server. That image doesn't
have any TMs inside it. A creator can then simply serve a 'file' or local repository, by mapping its directory
or volume into the container as follows:

```bash
docker run --rm --name tm-catalog -p 8080:8080 -v$(pwd):/thingmodels ghcr.io/wot-oss/tmc:latest
```

[1]: https://github.com/w3c/wot-thing-description/blob/main/validation/tm-json-schema-validation.json
[2]: https://schema.org
[3]: https://github.com/wot-oss/tmc/blob/main/api/tm-catalog.openapi.yaml
[4]: https://github.com/wot-oss/tmc/pkgs/container/tmc
[5]: ./commands#repo-add
