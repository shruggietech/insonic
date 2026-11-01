# Original receipt retention

The [manifest](manifest.json) names each gzip-compressed original receipt and records SHA256 and byte size for both the retained gzip and its uncompressed original. Decompress with a standard gzip reader, verify the original SHA256, then parse the JSON. Every retained file was read back and matched its original bytes before commit.

The receipts distinguish qualification, candidate retention, public asset downloads, publication reconciliation, deployed-tree/build reconciliation, live URL equality and sampled rendering. An initial publisher transport disconnect and initial Pages command failure remain recorded; successful independent reconciliation does not relabel those attempts as clean executions.

Full binary archives and the eight original PNG captures remain in the local retention paths named by the manifest. The official release independently retains all ten accepted publication assets. No source or artifact is rebuilt by this evidence record.
