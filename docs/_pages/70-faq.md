---
layout: post
permalink: /faq
title: Frequently Asked Questions
---

## Q: How to Import Multiple TMs for the Same Device With Different Protocol Bindings?

When you have a device that supports several protocol bindings, and you want to keep them in separate TMs, you have
several options of how to import the TMs and still be able to distinguish between the protocols when looking for a TM.

### Import With Optional Path

The recommended way is to use the optional path (`--opt-path`) option when importing the TMs to append the protocol name
to the TM name (and thus include it in the ID). For example,

```bash
tmc import lightall-coap.json --opt-path coap
tmc import lightall-http.json --opt-path http
tmc import lightall-modbus.json --opt-path modbus
```

This will result in each TM having a different TM name (e.g. `omnicorp/omnicorp/lightall/modbus`) and thus having
independent lists of versions. You can then select the right TM for you based on the TM name, fetch the latest version
for the TM name, etc.

Note that you can achieve the same effect when importing multiple TMs from a directory, if you prepare the folder
structure accordingly and use the `--opt-tree` flag. I.e. if you have prepared the following files:

```bash
$ find . -type f
./coap/lightall.json
./coap/senseall.json
./http/lightall.json
./http/senseall.json
./modbus/lightall.json
./modbus/senseall.json
```

you can import them in one go:

```bash
tmc import --opt-tree .
```

### Use Conventional Versions or Descriptions

For completeness' sake, the option of defining and using a convention in your TMs' versions or descriptions should be
mentioned.

You can include the protocol name in the semantic version (e.g. `v1.0.0-modbus`) or prefix the TM's description with
it (e.g. `"modbus: Omnicorp Lightall..."`). The TMs in this case will all have the same TM name and thus the different
versions will all be listed together. Listing the versions by `tmc versions` will allow to visually recognize which
version has which protocol. However, automatically checking for latest version of a TM for a certain protocol will
require some scripting on your part.
