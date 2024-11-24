---
layout: post
permalink: /commands
title: Commands
---

For most `tmc` subcommands, their help page (`tmc help <command>`) contains enough information for usage.

This page contains additional or detailed documentation for those subcommands or options where help page would not fit 
all the necessary information.

## `repo add`

> Usage:
>
>   tmc repo add <name> [--type &lt;type>] ((&lt;location> [--description &lt;description>]) | --file &lt;config-file> |
> --json &lt;config-json>)
> 

All repos have two mandatory fields: 'type' and 'loc' (short for location), and an optional 'description' field.
The 'type' is assigned from '--type' flag and the 'loc' is assigned from \<location> argument.
Depending on the repo type, the exact meaning of 'loc' field is different and also other fields may be provided or may be mandatory.

When adding a repository, the entire config may be provided in JSON form, either by giving a file name in
\<config-file> or the entire JSON as string in \<config-json. See `repo show` for example.

### File Repositories

File repo is the primary repo type.

For repos of type "file", 'loc' field is the path to filesystem directory where the repository is located.

File repos do not define any additional fields.

### HTTP Repositories

HTTP repo type allows accessing a file repo exposed over HTTP by simple directory hosting. It also permits
accessing a repo committed to some git forge, e.g. GitHub or GitLab.

For repos of type "http", 'loc' field is the URL of the repository.

HTTP repos are read-only.

The structure of HTTP directory tree is expected to be exactly as the one stored by a file repo.
That means, that normally, the relative file paths are simply appended to the 'loc' URL. If the server requires putting
the requested file's path anywhere other than at the end of the URL, you can use a placeholder `{% raw %}{{ID}}{% endraw %}` in the URL, both in the path and in query part.

E.g. for an HTTP repo served by a GitLab installation, you may use ```https://example.com/api/v4/projects/<project-id>/repository/files/{% raw %}{{ID}}{% endraw %}?ref=main``` 
as the URL.

### TMC Repositories

TMC repo allows to access a remotely hosted repository via TMC REST API, which is provided by `tmc serve`.

For repos of type "tmc", 'loc' field is the URL of the REST API.

## `attachment fetch`

Basic usage of `attachment fetch` is straightforward, however the `--concat` flag requires some elaboration.

### `--concat`

The `--concat` flag allows automating the production of documentation of changes between versions of TM. For example, you may put
your general description of a device into a file "README.md" and attach it to the TM name. Then, for each version of 
TM you import into the catalog, you may put just the version-specific documentation or changelog in an attachment with the same
name "README.md" and attach it to the corresponding TM id. Given this setup, if you then use execute `tmc attachment fetch --concat <tm-name> README.md`
you will get a file, which is a concatenation of the "README.md" attached to TM name and all "README.md" files attached to
individual TMs with this TM name. 

The order of TM id attachments is the same as you get from `tmc versions <tm-name>`.
If a version does not have an attachment named "README.md", it is skipped.

The flag is intended to concatenate simple text files. It does not verify whether concatenating attachments produces a valid file for its media type.
E.g. concatenating HTML or PDF files is not going to produce useful results.

## `check`

When a file repository is [published to a git forge][1], there exists the risk that contributions from multiple people
will produce conflicts which won't get detected in time. Hence, there is the `check` command to verify a repository for
internal consistency and integrity of the storage. `check` will fail if you store any files in the repository other than
those that were added by using `tmc`. Exception to that is that files on top level and directories starting with a dot,
are always ignored. In fact, when you run `tmc check` for the first time, a `.tmcignore` file with defaults is created
in `.tmc` folder under file repository's root. If you do want to store some other files along with TMs and attachments
under the repo's root, you should add corresponding lines to `.tmcignore`. It has the same pattern format as
[`.gitignore`][2], but the paths are always relative to repo's root, instead of to directory where `.tmcignore` resides.

[1]: ./workflows#publish-a-catalog-to-a-git-forge
[2]: https://git-scm.com/docs/gitignore#_pattern_format
