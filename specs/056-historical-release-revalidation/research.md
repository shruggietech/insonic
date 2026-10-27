# Research: Original release authority

2026-10-27: Confirmed #185 and the published v1.0.1 release source 136f5c0307a6fce00ba8240ef69c907a944cd8c0, tree 1c4ae59c053a3d8d751cde5c736f67f2bbeecd3c. GitHub lists nine assets: four product archives, one MSI, two documentation archives, manifest and checksums. There is no setup EXE.

The downloaded original manifest matches the published SHA-256 ce885efde09f7ba9d4b12d3b60ec9e87f918921e2db5fcc6ec2021fab743b8d8. Its original MSI SHA-256 is beff9343561e160e157ed2ce3f8f41a287b6bdd9f88beec0779ae408fceae22d. These immutable anchors cover all original MSI tables and payload streams without executing old code or rebuilding the current model.

Current release.py requires modern terminal receipts and installer setup, while windows_msi.py reconstructs a later table model. Existing archive checks already accept a complete expected_source_root and validate original source helpers, worker bytes, media inventories and original qualification. Supplying schemas alone is insufficient.

Rejected alternatives: generic historical version exemption (accepts unknown authorities), executing old verifiers (changes trusted code boundary), copying legacy MSI authoring (duplicates logic). Select the pinned profile only after original manifest and complete source verification; require all applicable original qualification independently. Modern and unknown candidates remain strict. Existing v1.0.0 handling remains separately maintained.
