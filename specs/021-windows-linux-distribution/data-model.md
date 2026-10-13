# S021 data model

Current targets are exactly windows_amd64 and linux_amd64, each with cli and desktop variants. A current candidate owns four distinct qualified package receipts, complete source/content inventory, exact revision and aligned 1.0.1 version. Historical documents use their immutable version contracts.

Installation attempts transition absent destination -> uniquely owned copy attempt -> completed new installation, or failed owned attempt -> verified cleanup. Existing or substituted destinations never acquire cleanup ownership. Library/media/model/credential state is independent of program-directory ownership.
