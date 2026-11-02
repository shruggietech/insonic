# Original receipt retention

The [manifest](manifest.json) records byte sizes and SHA-256 digests for every gzip-compressed original receipt and its uncompressed bytes. Decompress with a standard gzip reader, verify the original digest, then parse JSON where indicated. Each retained file was read back and matched its original bytes before commit.

These records distinguish accepted product source and qualification, unchanged-byte publication, independent public downloads, exact Pages deployment, live byte equality and sampled rendering. Full original release-manifest qualification receipts are retained alongside run/job/step metadata and terminal timing. Final PR CI and review closure are retained in the PR conversation after the final source commit.

Original workflow ZIPs, all ten accepted assets, their independent public downloads, the exact Pages export and eight PNG captures remain in the ignored local paths named by the manifest and artifact/image inventories. The official release independently retains all ten publication assets. No release asset is rebuilt by this evidence record.
