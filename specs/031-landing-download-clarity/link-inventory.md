# S031 landing-link inventory

Version context: current v1.0.1. Latest guidance stays on stable aliases; all-version navigation is separate. Shared chrome uses the same native anchors in snapshot/released views. Browser reports record actual postbuild relative hrefs and fully resolved local verification URLs.

| Visible label or accessible name | Destination | Intent/type |
| --- | --- | --- |
| Skip to content | `#main` | In-page bypass |
| insonic home | `/` | Home |
| Documentation | `/docs/latest/` | Current guidance |
| Repository | `https://github.com/shruggietech/insonic` | Source repository |
| Choose a download | `#downloads` | In-page choice jump; single primary action |
| Read the documentation | `/docs/latest/` | Current guidance; intentional hero/header repetition |
| Media and metadata | `/docs/latest/ingestion/` | Import contract |
| Pipelines and adapters | `/docs/latest/pipelines/` | Processing contract |
| Speaker audio and models | `/docs/latest/voice-models/` | Speaker/model contract |
| Queries and exploration | `/docs/latest/graph/` | Query contract |
| Explore the JSON contracts | `/docs/latest/contracts/` | JSON contract viewer |
| Read the architecture | `/docs/latest/architecture/` | Architecture guidance; subordinate link |
| All documentation versions | `/docs/versions/` | Version archive |
| Download Windows MSI | Release asset below | Selectable installer |
| Windows installation, updates and removal | `/docs/latest/windows-installation/` | MSI lifecycle and ZIP overview |
| Download Windows Desktop + CLI ZIP | Release asset below | Portable Windows application/CLI |
| Download Windows CLI-only ZIP | Release asset below | Portable Windows CLI |
| Windows ZIP installation and maintenance | `/docs/latest/desktop/#install-a-windows-zip` | ZIP lifecycle guidance |
| Download Linux Desktop + CLI tar.gz | Release asset below | Portable Linux application/CLI |
| Download Linux CLI-only tar.gz | Release asset below | Portable Linux CLI |
| Linux prerequisites and installation | `/docs/latest/desktop/#linux-prerequisites-and-installation` | Linux prerequisites/install; no update/removal promise |
| ShruggieTech | `https://shruggie.tech/` | Company affiliation |

The redundant bottom `Installation and packaging requirements` link is removed. Each platform keeps direct guidance; general package information remains reachable through documentation. The preview's three tabs and theme switch are buttons, not inventory anchors; they remain in geometry/focus coverage.

## Exact release assets

Every URL begins `https://github.com/shruggietech/insonic/releases/download/v1.0.1/`. The complete destination is that prefix plus the exact filename below. These five assets appear only in released rendering.

| Exact filename | Platform/architecture | Client choice | Format/purpose |
| --- | --- | --- | --- |
| `insonic_1.0.1_windows_amd64_installer.msi` | Windows amd64 | Desktop, CLI or both; both default | MSI installation/maintenance |
| `insonic_1.0.1_windows_amd64_desktop.zip` | Windows amd64 | Desktop + CLI | Portable ZIP |
| `insonic_1.0.1_windows_amd64_cli.zip` | Windows amd64 | CLI only | Portable ZIP |
| `insonic_1.0.1_linux_amd64_desktop.tar.gz` | Linux amd64 | Desktop + CLI | Portable tar.gz |
| `insonic_1.0.1_linux_amd64_cli.tar.gz` | Linux amd64 | CLI only | Portable tar.gz |

Fresh official release API asset names/URLs agree with the original release manifest. The cached original manifest's SHA-256 `ce885efde09f7ba9d4b12d3b60ec9e87f918921e2db5fcc6ec2021fab743b8d8` matches the fresh API digest, binding its four package receipts and selectable MSI metadata to the published manifest. No live binary download/installation is claimed by link-activation tests.
