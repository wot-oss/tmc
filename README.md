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
go install github.com/wot-oss/tmc@v0.1.0
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

### Browse the Canonical Catalog

1. Configure the canonical repository
    ```bash
    tmc repo add --type http thingmodels 'https://example.com/thingmodels'
    ```
2. List the contents of the canonical catalog
    ```bash
    tmc list
    ```

The listed names are formatted as follows

```
<author>/<manufacturer>/<model>
```

You can specify a part of that path after the ```list``` command to filter the list for only parts of the list tree (use tab to auto-complete path parts):

```
tmc list nexus-x/siemens
```

### List Versions

Every model entry in the list may contain multiple versions, reflecting the evolution of the Thing Model (bugfixes, additions, changes in the device itself ...). List the available versions with the ```versions``` command:

```bash
tmc versions <name>
```

### Fetch a Thing Model

Like what you see? Fetch and store locally using the ```fetch``` command. It will print the Thing Model to stdout to enable unix-like piping:

```bash
tmc fetch <ID>
```

You can also fetch a TM by specifying only the name part, optionally with a semantic version: 
```bash
tmc fetch <NAME>[:<SEMVER]
```
Doing so will fetch the latest version of the TM that matches the given name and semver.

To store the Thing Model to a file instead of printing to stdout, specify the ```-o``` flag and point it to a directory:

```bash
tmc fetch <NAME> -o .
```


[1]: https://www.w3.org/TR/wot-thing-description11/
[2]: https://github.com/wot-oss/tmc/releases
[3]: https://wot-oss.github.io/tmc/

## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fwot-oss%2Ftmc.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fwot-oss%2Ftmc?ref=badge_large)