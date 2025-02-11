---
layout: post
permalink: /commands
title: Commands
---

For most `tmc` subcommands, their help page (`tmc help <command>`) contains enough information for usage.

This page contains additional or detailed documentation for those subcommands or options where the help page would not
fit all the necessary information.

## `repo add`

> Usage:
>
>   tmc repo add <name> [--type &lt;type>] ((&lt;location> [--description &lt;description>]) | --file &lt;config-file> |
> --json &lt;config-json>)
> 

All repos have a mandatory field 'type' and an optional 'description' field.  
The 'type' is assigned from '--type' flag.  
Depending on the repo type, also the field 'loc' (short for location) may be required and is then assigned from \<location> argument.  
The exact meaning of 'loc' field may different and also other fields may be provided or may be mandatory.  

When adding a repository, the entire config may be provided in JSON form, either by giving a file name in
\<config-file> or the entire JSON as string in \<config-json>. See `repo show` for example.

Some configuration parameters can be defined by environment variables. To refer to an environment variable, set the
value of a parameter to the variable's name prefixed by a '\$', e.g. '\$PROD_TOKEN'. This expansion of env variables is
supported in the following fields of repository config: 'enabled', 'loc', 'auth' (leaf fields, like 'username'), 
'headers' (both header names and values).

### File Repositories

File repo is the primary repo type.

For repos of type "file", 'loc' field is the path to filesystem directory where the repository is located.

File repos do not define any additional fields.

### S3 Repositories

S3 repo allows accessing a repository stored in an AWS S3 bucket.  
Like a file repo, this type of repository is writable.

S3 repos do not have a 'loc' field, the entire configuration must be provided in JSON form.  
The config json requires at least the field 'aws_bucket'.  

Depending on how AWS client access is configured where the 'tmc' binary is running (e.g. via IAM role or shared config and credential files),   
additional fields are necessary for the S3 repo.   

A complete config json may look like this:
```
{
  "aws_bucket": "catalog",
  "aws_region": "eu-central-1",
  "aws_access_key_id": "$AWS_AKI", 
  "aws_secret_access_key": "$AWS_SAK",
  "description": "Repository stored on AWS",
  "type": "s3"
}
```



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
