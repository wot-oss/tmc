# Handle Conflicts on TM Push

## Status
13.05.2024 proposed

## Context

In context of pushing a TM with the same TM name and semantic version as already exists in repository,
there are two kinds of conflict that can occur:

1. A TM with the same content (hash) is already the latest by timestamp. In this case, the push is rejected. 
    (Only the latest version by timestamp is considered, so that it remains possible to override the latest version with
    one of the older versions)
2. A TM with different hash exists, but with same timestamp, making it impossible to sequentially order these two versions, except randomly (e.g. by hash)

Currently, in the second case, the push is rejected by the repository, but the command waits a second and pushes again, resolving 
the conflict nominally, but still producing a somewhat random result ordering of the two conflicting versions. 
In the sense that it is most probably not intended or expected by the user.

If the user performs a directory push where the directory contains two TMs for the same device, they are pushed in this 
quasi-random order. Then, if the same push is performed again, two more versions of the TM are created. 
None of the two TMs are recognized as an already existing TM, because at each moment the other one is the latest by timestamp and 
thus no conflict of the first kind is encountered. Normally, a repeated push of the same folder is expected to be
rejected, because all files are already imported into repository.

## Decision

1. Extend the check for the existence of the same content to all existing versions, not only the latest by timestamp.
2. Add an option to the push command to force pushing a TM which would otherwise be rejected with the "TM exists" error. This retains the ability to override the latest version by timestamp with a given TM
3. In case of the conflict of the second kind, accept the conflicting TM and alert the user to the existence of the conflict by a warning message

## Consequences

Pushing directories may now produce warnings which may cause confusion among users as to how they are to be resolved.
