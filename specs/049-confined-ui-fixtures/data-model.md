# S049 Fixture Lifecycle

A temporary parent has an absolute non-root configured path and a verified canonical directory identity. A fixture has a unique prefix, canonical parent, directory path, allocation identity (device/inode), and lifecycle state. Initialization may add the capture ownership marker and isolated profile directories.

Transitions: unallocated -> allocated/owned -> initialized -> children released -> removed. Initialization failure transitions directly to owned cleanup. A replacement, redirect, failed child release or removal failure transitions to a diagnostic failure with residual path. Repeated cleanup after verified removal is harmless; it must not delete a newly created replacement. Marker failure never cancels responsibility for the allocated directory. Forced termination has no finalization guarantee.

Detailed parent/path and removal evidence belongs to disposable qualification diagnostics/artifacts. Published screenshot provenance uses non-private allocation metadata and verified cleanup, retaining source and binary identities.
