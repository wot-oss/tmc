# Why We Don’t Derive Repo Types from URI Schemes

## Status

accepted

## Context

A question repeatedly comes up in discussions why don't we use URI scheme to define the type of repository instead of making user specify the type explicitly. 
An example of a seemingly useful scheme is 'tmc' to define a repository based on a remote instance of our own API. 

This ADR records the reasons for why we don't do that.

## Decision

List of reasons in no particular order:

1. W3C discourages creating new URI schemes, especially private schemes, not registered with IANA. 
See references [1] and [2]
2. Even for a single repo type 'tmc', one URI scheme would not be enough. 
We would need at least to define a TLS variant of the same scheme, like 'tmcs' or 'tmc+tls'.
3. After a user started our API with `tmc serve` she would need to differentiate between URIs used to access the
API from a browser or a REST client like Postman ("http://thingmodels.example.com") and from another tmc CLI 
("tmc://thingmodels.example.com").
4. If we ever decide to offer another API, e.g. gRPC, that would confuse the matters even more as it would require some 
kind of differentiation in the repo config between the REST API and a gRPC API types.

## Consequences

We will continue to require specifying repo type independently of URI scheme.


[1]: https://www.w3.org/wiki/UriSchemes
[2]: http://infomesh.net/2001/09/urischemes
