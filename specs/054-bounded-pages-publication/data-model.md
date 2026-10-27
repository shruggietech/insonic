# Publication data model

Original export maps canonical relative paths to bytes. Each expected file has mode 100644, byte size and Git blob SHA-1. A staged tree has an immutable SHA and complete recursive inventory; intermediate trees are unreferenced. The final commit has the originally observed destination head as its sole parent. Only a verified non-forced branch update makes it public. Diagnostic context retains method, query-free API path, status, request bytes and bounded redacted server fields, never headers or submitted content.
