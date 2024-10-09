---
layout: post
permalink: /gettingstarted
title: Getting Started
---

The ```tmc``` helps you to interact with a Thing Model catalog, which may be hosted on any git forge like GitHub or create your own catalog in a git repository of your choosing.

To integrate publicly available and your own private Thing Models into your product, the ```tmc``` can be run as a server, exposing a REST API that can be protected with JWT tokens.

## Configure Autocompletion (Optional)

1. Read the help of the ```completion``` command to find out which shells are supported
    ```bash
    tmc completion -h
    ```

2. Follow the instructions of the shell specific help text
    ```bash
    tmc completion <shell> -h
    ```

## Browse the Example Catalog

We provide an [example repository][3] for you to get acquainted with `tmc`. The following commands assume that you use the example repository.
If your organization hosts a TM catalog for use as a default, you will need to change the commands accordingly.

### Configure the Example Repository

```bash
tmc repo add -t http example https://raw.githubusercontent.com/wot-oss/example-catalog/refs/heads/main
```

### List the Contents of the Example Repository

```bash
tmc list
```

The listed names are formatted as follows

```
<author>/<manufacturer>/<model>[/<optional-path>]
```

You can specify a part of that path after the ```list``` command to filter the list for only parts of the list tree (
use tab to auto-complete path parts):

```
tmc list omnicorp/omnicorp
```

There are also other options to list a subset of available TMs from a catalog. See `tmc list -h` for details.

### List Versions

Every model entry in the list may contain multiple versions, reflecting the evolution of the Thing Model (bugfixes,
additions, changes in the device itself ...). List the available versions with the ```versions``` command:

```bash
tmc versions omnicorp/omnicorp/lightall
```

### Fetch a Thing Model

Like what you see? Fetch and store locally using the ```fetch``` command. It will print the Thing Model to stdout to
enable unix-like piping:

```bash
tmc fetch omnicorp/omnicorp/lightall/v2.0.1-20241008124326-8c75753996b3.tm.json
```

If you specify just the name, the CLI will fetch the latest version automatically.

```bash
tmc fetch omnicorp/omnicorp/lightall
```

You can fetch the latest TM matching a specific semantic version or part of it by adding the version to the TM name,
separated by a colon. For example, all the following commands fetch the same version 'v2.0.1'.

```bash
tmc fetch omnicorp/omnicorp/lightall:v2
tmc fetch omnicorp/omnicorp/lightall:v2.0
tmc fetch omnicorp/omnicorp/lightall:v2.0.1
```

To store the Thing Model locally instead of printing to stdout, specify the ```-o``` flag and point it to a
directory:

```bash
tmc fetch omnicorp/omnicorp/lightall-mk2 -o .
```

## Host Your Own Catalog

If you want to host a catalog for your organization or project you should create (See [Create a Repository][4]) and populate (See [Import Thing Models][5]) a repository 
and then host it using one of the two options:
1. [A simple read-only catalog hosted by your favorite git forge][1]
2. [A catalog served by the TMC REST API][2]

You can configure those as a repo of type 'http' or 'tmc', respectively

[1]: ./workflows#publish-a-catalog-to-a-git-forge
[2]: ./workflows#expose-a-catalog-for-http-clients
[3]: https://github.com/wot-oss/example-catalog
[4]: ./workflows#create-a-repository
[5]: ./workflows#import-thing-models
