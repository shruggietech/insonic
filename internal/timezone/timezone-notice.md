# Timezone data

`zoneinfo.zip` is the unmodified public-domain IANA rules archive shipped in Go 1.27.1. SHA-256: `b2d18a7c8fa8142097a48c99609fb3c92db5ee98bc740294e57eab8ae9f94779`.

`windows-zones.json` contains the world (`territory="001"`) Windows-to-IANA mappings from [Unicode CLDR release 48](https://github.com/unicode-org/cldr/blob/release-48/common/supplemental/windowsZones.xml). It is covered by the [Unicode License](UNICODE-LICENSE.txt), retained beside the mapped data. Its exact bytes are embedded with the rules, and every interpretation records `CLDR-release-48` plus the rules digest. An unavailable platform mapping leaves dates unresolved.
