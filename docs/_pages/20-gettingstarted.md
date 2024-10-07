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

### Browse the Default Catalog

1. Configure default repository
    ```bash
    tmc repo add --type http thingmodels 'https://example.com/thingmodels'
    ```
   At the moment, a default catalog should be provided by your organization. We envision providing an example 
ThingModel catalog available to everyone in the near future. 

> ##### TIP
>
> If you want to host a catalog for your organization or project, you have two options: 
> 1. [A simple read-only catalog hosted by your favorite git forge][1]
> 2. [A catalog served by the TMC REST API][2]
> You can configure those as a repo of type 'http' or 'tmc', respectively
{: .block-tip }

2. List the contents of the default repository
    ```bash
    tmc list
    ```

The listed names are formatted as follows

```
<author>/<manufacturer>/<model>[/<optional-path>]
```

You can specify a part of that path after the ```list``` command to filter the list for only parts of the list tree (use tab to auto-complete path parts):

```
tmc list nexus-x/siemens
```

## List Versions

Every model entry in the list may contain multiple versions, reflecting the evolution of the Thing Model (bugfixes, additions, changes in the device itself ...). List the available versions with the ```versions``` command:

```bash
tmc versions <name>
```

## Fetch a Thing Model

Like what you see? Fetch and store locally using the ```fetch``` command. It will print the Thing Model to stdout to enable unix-like piping:

```bash
tmc fetch <ID>
```

If you specify just the name, the CLI will fetch the latest version automatically. 

```bash
tmc fetch <NAME>
```

If you want to fetch a specific semantic version, append the version string to the name, separated by a colon:

```bash
tmc fetch <NAME>:<SEMVER>
```

To store the Thing Model locally instead of printing to stdout, specify the ```-o``` flag and point it to a directory:

```bash
tmc fetch <NAME> -o .
```

[1]: ./workflows#publish-a-catalog-to-a-git-forge
[2]: ./workflows#expose-a-catalog-for-http-clients