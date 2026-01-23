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

-  If your TM file is: `../example-catalog/.tmc/omniuser/omnicorp/senseall/v1.0.0-20241008124326-15af48381cf7.tm.json`
-  Then an attachment (e.g., `readme.md`) for this TM would be placed at: `../example-catalog/.tmc/omniuser/omnicorp/senseall/.attachments/v1.0.0-20241008124326-15af48381cf7.tm.json/readme.md`

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

### Catalog as S3 bucket

In order to quickly getting started with S3, we recommend to use [localstack][6] (requires docker) and [awslocal][7] for local developments. Once installed:

1. start localstack:
```bash
localstack start
```
2. create a bucket in s3 by running:
```bash
awslocal s3api create-bucket --bucket tmc-bucket --region eu-central-1 --create-bucket-configuration LocationConstraint=eu-central-1
```
3. copy the tmc into the newly created bucket:
```bash
awslocal s3 cp <local_repo_folder> s3://tmc-bucket --recursive --endpoint-url=http://localhost:4566
```
4. create the S3 repo configuration in config.json:
```json
{
  "s3repo": {
    "description": "",
    "aws_bucket": "tmc-bucket",
    "aws_region": "eu-central-1",
    "aws_endpoint": "http://localhost:4566",
    "aws_access_key_id":"some access key",
    "aws_secret_access_key":"some secret",
    "type": "s3"
  }
}
```
5. run tmc. the s3 repo should be accessible just as any other repo, you've been using before.

## JWT Validation for API Requests

The `serve` command can be configured with the `--jwtValidation` flag to enforce security by requiring valid JWTs for incoming API requests. 

### Configuration

1. **`--jwtValidation`** 
  - Enables JWT-based access control for the API server.

2. **`--jwksURL=<url>`** 
  - Specifies the JWKS URL to retrieve public keys for verifying JWT signatures.
  - Example: `http://127.0.0.1:8100/.well-known/jwks.json`.

3. **`--jwtServiceID=<serviceID>`** 
  - String that represents the **audience** (`aud` claim) required in valid JWTs.
  - Example: `"myServiceID"`.

### Behavior with JWT Validation

When the `--jwtValidation` flag is provided:

#### 1. Bearer Token Requirement 
- All incoming requests **must include a valid Bearer token** in the `Authorization` header. 

#### 2. JWKS Validation 
- Incoming tokens are validated against the JSON Web Key Sets (JWKS) at the URL specified in `--jwksURL`. 
- The server checks the following:
- JWT signature is valid and matches the public key(s) defined in the JWKS.
- JWT is issued by the **issuer URL** corresponding to `--jwksURL`.
- JWT audience (`aud` claim) matches the value specified in `--jwtServiceID`.

#### 3. Scope-Based Access Control 
- The JWT must include a `scope` claim, which is an array of strings defining user permissions. 
- Each scope string determines the user's access rights to specific endpoints, as defined in the **Scope Table** (explained below). 
- Example Scope Claim:
```json
"scope": ["tmc.ns.myNamespace.read", "tmc.ns.myNamespace.write"]
```

#### 4. Token Validation Details

The token is confirmed to be valid if:

Its signature is verified using a public key from the JWKS.
The token has not expired (checks the exp claim).
It is issued by the expected issuer URL.
The aud claim equals the value of --jwtServiceID.
The scope claim contains sufficient permissions for the requested endpoint.

#### 5. Error Handling

Requests without a valid Bearer token will result in an HTTP 401 Unauthorized error.

#### 6. Scope Table

<div style="overflow-x: auto; width: 100%;">

<table style="border-collapse: collapse; width: 100%;">
  <thead>
    <tr>
      <th>Scope</th>
      <th>/thing-models/{tmID}/attachments/{attachmentFileName} (GET)</th>
      <th>/thing-models/{tmID}/attachments/{attachmentFileName} (PUT)</th>
      <th>/thing-models/{tmID}/attachments/{attachmentFileName} (DELETE)</th>
      <th>/thing-models/{tmName}/{tmVersion}/attachments/{attachmentFileName} (GET)</th>
      <th>/thing-models/{tmName}/{tmVersion}/attachments/{attachmentFileName} (PUT)</th>
      <th>/thing-models/{tmName}/{tmVersion}/attachments/{attachmentFileName} (DELETE)</th>
      <th>Inventory (GET)</th>
      <th>/thing-models/{tmID} (GET)</th>
      <th>/thing-models/{tmID} (DELETE)</th>
      <th>/thing-models/.latest/{fetchName} (GET)</th>
      <th>/thing-models/.latest/{fetchName} (POST)</th>
      <th>/thing-models (POST)</th>
      <th>/repos (GET)</th>
      <th>/info* (GET)</th>
      <th>/health* (GET)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><b>tmc.ns.{namespace}.read: Reading TMs, their metadata and attachments</b></td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>no</td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>no</td>
      <td>if tmID == namespace</td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
    </tr>
    <tr>
      <td><b>tmc.ns.{namespace}.write: Adding new TMs and attachments</b></td>
      <td>no</td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>no</td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>if tmID == namespace</td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
    </tr>
    <tr>
      <td><b>tmc.ns.{namespace}.attachments.delete: Deleting attachments</b></td>
      <td>no</td>
      <td>no</td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>no</td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
    </tr>
    <tr>
      <td><b>tmc.ns.{namespace}.thingmodels.delete: Deleting TMs (not desired, thus separate scope)</b></td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>if tmID == namespace</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
    </tr>
    <tr>
      <td><b>tmc.repos.read: Reading /repos</b></td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>yes</td>
      <td>no</td>
      <td>no</td>
    </tr>
    <tr>
      <td><b>tmc.internal.read: Reading everything under /info</b></td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>yes</td>
      <td>no</td>
    </tr>
    <tr>
      <td><b>tmc.health.read: Reading everything under /healthz</b></td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>no</td>
      <td>yes</td>
    </tr>
    <tr>
      <td><b>tmc.admin: everything's allowed</b></td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
      <td>yes</td>
    </tr>
  </tbody>
</table>

</div>

[1]: https://github.com/w3c/wot-thing-description/blob/main/validation/tm-json-schema-validation.json
[2]: https://schema.org
[3]: https://github.com/wot-oss/tmc/blob/main/api/tm-catalog.openapi.yaml
[4]: https://github.com/wot-oss/tmc/pkgs/container/tmc
[5]: ./commands#repo-add
[6]: https://docs.localstack.cloud/aws/getting-started
[7]: https://github.com/localstack/awscli-local
