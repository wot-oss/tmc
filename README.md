# Thing Model Catalog CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/wot-oss/tmc)](https://goreportcard.com/report/github.com/wot-oss/tmc) [![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/wot-oss/tmc)](https://github.com/wot-oss/tmc/releases) [![PkgGoDev](https://img.shields.io/badge/go.dev-docs-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/wot-oss/tmc)

![Thing Model Catalog Logo](https://github.com/wot-oss/tmc/raw/main/docs/media/tm-catalog-logo.svg)
---
Find, use and contribute device descriptions for industrial IoT devices!

⚠ This software is **experimental** and may not be fit for any purposes. 

The Thing Model Catalog Command Line Client, or ```tmc``` for short, is a tool for browsing, consuming, contributing and serving Thing Models.

Thing Models are simple device descriptions specified in the [W3C Thing Description][1] standard. Thing Models are to Thing Descriptions what classes are to instances in programming languages.

Thing Models let you describe industrial devices using a simple standardized JSON-based format, which is independent of the communication protocol. This enables a uniform access layer to the fragmented industrial protocol landscape we encounter today.

Thing Descriptions are to Modbus, BACnet, MQTT, DNP3 ... what HTML is to HTTP.

---

## Installation

1. Download the latest [release][2] for your operating system and architecture
2. Optionally rename to ```tmc``` to remove os/arch postfixes
3. Give it execution rights and move to a folder that is in your ```PATH```

## Quick Start

The ```tmc``` helps you to interact with a Thing Model catalog, which may be hosted on any git forge like github or create your own catalog in a git repository of your choosing. 

To enable a culture of sharing, we provide a canonical repository at [], but feel free to create your own open or private catalog as well.

To integrate publicly available and your own private Thing Models into your product, the ```tmc``` can be run as a server, exposing a REST API that can be protected with JWT tokens.

### Configure Autocompletion

1. Read the help of the ```completion``` command to find out which shells are supported
```bash
tmc completion -h
```

2. Follow the instructions of the shell specific help text
```bash
tmc completion <shell> -h
```

### Browse the canoncial Catalog

1. Configure the canonical repository
```bash
tmc repo add --type http thingmodels 'https://raw.githubusercontent.com/wot-oss/thingmodels'
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
tmc fetch <NAME>
```

If you just specify the name, the cli will fetch the latest version automatically. If you want to fetch a specific version, append the version string to the name, separated by a colon:

```bash
tmc fetch <NAME>:<SEMVER>
```

To store the Thing Model locally instead of printing to stdout, specify the ```-o``` flag and point it to a directory:

```bash
tmc fetch <NAME> -o .
```


[1]: https://www.w3.org/TR/wot-thing-description11/
[2]: https://github.com/wot-oss/tmc/releases
