# S028 documentation and delivery contracts

## Current and historical routes

/docs/ is a current-first documentation entry. /docs/latest/ and /docs/latest/TOPIC/ render full manifest-latest content with truthful version/status and canonical fixed-version identity. /docs/versions/ is the explicit archive chooser. /docs/vVERSION/... retains that version. Missing current topics do not silently return archived content. Trailing slash/index/file routes, query and fragment behavior remain useful in exported HTML, with no server dependency.

## Navigation and reading

Shared article/reference frames provide labelled sidebar/contents disclosures, one principal heading, current-page/version context, main skip target and previous/next where applicable. Narrow layouts begin compact; native keyboard/touch behavior and visible focus remain available without custom menu roles. Complex content scrolls within labelled focusable containers where necessary, with semantic table structure and readable diagram/source alternatives.

## CI and release boundary

Owner-approved active CI window is660seconds. Backend lease exclusion/takeover/fencing/renewal and every elected security/integrity/package lane remain mandatory. S028 publishes branch/PR only, resolves findings in at most two requested rounds, and stops after final exact-head green CI for owner review/merge. #97 publication/tag/site deployment stays after merge.
