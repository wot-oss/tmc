---
layout: post
permalink: /architecture
title: Architecture
---

While the [OpenAPI description][1] is enough to get started, this page contains more architectural information about the Thing Model Catalog.
Make sure to read the [concepts][2] page first.

![Thing Model Catalog Basic Architecture]({{site.baseurl}}/media/architecture.png)

## Inventory and Repositories

A TMC allows managing an inventory (a catalog).
This inventory can contain different repositories, which can be of different types such as file, http (git urls), TMC REST API, or AWS S3 buckets.
This allows repositories to be hosted by different parties, while a single TMC client managing them from one place.

## Authors and Namespaces

A repository typically contains multiple authors of TMs, each separated by their namespace.
An author can be the manufacturer of the device but not necessarily, i.e. one can write and serve TMs for other manufacturer's devices.
In productive deployments, you may want to apply authorization based on the namespace.

## Devices and Thing Models

Each author has a publishes TMs for their devices, where a device has TMs with different versions.
Across one model part number (referred to as tmName), the TM versions get incremented as needed.
For example, a typo fix can be a version increment or a new firmware can result in a different TM.
TMC does not mandate a specific versioning scheme regarding how the content of the TM changes.
However, a TM cannot be changed in place, i.e. each change must be published as a new TM.
As the id of a TM is generated based on the content, each change results in a new TM id.
In the filesystem, all TMs of a device are found under the same folder.

## Attachments

In addition to a TM, a device can contain attachments such as images, manuals or even binaries.
An attachment can be linked to a device or to a specific TM version.
In the filesystem, an `.attachment` folder is created under the folder of the device.

## Device Metadata

When a TM is added for a device, TMC creates metadata that contains the following information:

- Repository of the TM
- Author, manufacturer and MPN information
- Links to the all known TM versions

TMC uses the metadata information for providing search, querying and indexing purposes.

[1]: https://github.com/wot-oss/tmc/blob/main/api/tm-catalog.openapi.yaml
[2]: ./30-concepts.md

