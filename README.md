# Thing Model Catalog CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/wot-oss/tmc)](https://goreportcard.com/report/github.com/wot-oss/tmc) [![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/wot-oss/tmc)](https://github.com/wot-oss/tmc/releases) [![PkgGoDev](https://img.shields.io/badge/go.dev-docs-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/wot-oss/tmc)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fwot-oss%2Ftmc.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fwot-oss%2Ftmc?ref=badge_shield)

![Thing Model Catalog Logo](https://raw.githubusercontent.com/wot-oss/tmc/main/docs/media/tm-catalog-logo.svg)
---
Find, use and contribute device descriptions for industrial IoT devices!

⚠ This software is **experimental** and may not be fit for any purpose. 

The Thing Model Catalog Command Line Client, or ```tmc``` for short, is a tool for browsing, consuming, contributing and serving Thing Models.

Read our [Documentation][3] for more.

---

## Installation

Download binary from [releases][2] page or

```bash
go install github.com/wot-oss/tmc@v0.1.2
```

## Quick Start

### Configure Autocompletion (Optional)

1. Read the help of the ```completion``` command to find out which shells are supported
    ```bash
    tmc completion -h
    ```

2. Follow the instructions of the shell specific help text
    ```bash
    tmc completion <shell> -h
    ```

### Browse the Example Catalog

We provide an [example repository][4] for you to get acquainted with `tmc`. The following commands assume that you use
the example repository. If your organization hosts a TM catalog for use as a default, you will need to change the
commands accordingly.

#### Configure the Example Repository

```bash
tmc repo add -t http example https://raw.githubusercontent.com/wot-oss/example-catalog/refs/heads/main
```

#### List the Contents of the Example Repository

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

#### List Versions

Every model entry in the list may contain multiple versions, reflecting the evolution of the Thing Model (bugfixes,
additions, changes in the device itself ...). List the available versions with the ```versions``` command:

```bash
tmc versions omnicorp/omnicorp/lightall
```

#### Fetch a Thing Model

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

[1]: https://www.w3.org/TR/wot-thing-description11/
[2]: https://github.com/wot-oss/tmc/releases
[3]: https://wot-oss.github.io/tmc/
[4]: https://github.com/wot-oss/example-catalog

## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fwot-oss%2Ftmc.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fwot-oss%2Ftmc?ref=badge_large)
