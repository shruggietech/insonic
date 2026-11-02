# S067 Maintainer CLI contract

`python scripts/release.py pages-promote --candidate DIRECTORY [--source-root DIRECTORY] [--output RECEIPT]`

Output defaults to candidate/pages-promotion.json; an existing receipt is refused before destination mutation. New attempts require a distinct output. Failure preserves sanitized phase/identity evidence. Existing success fields remain available.

`python scripts/release.py pages-reconcile --candidate DIRECTORY --attempt RECEIPT --output RESULT [--source-root DIRECTORY]`

All remote operations are GET. Output must be distinct from attempt and must not exist. Revalidates source, publication, archive, destination, original and staged trees/parent, exact completed build and public content. Does not stage, patch, request a build, roll back or change release bytes. A missing or changed required identity fails.
