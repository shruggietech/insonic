# Native extractor qualification inputs

The [media tool lock](media-tools.json) pins archive SHA-256 values for ExifTool 13.59 distributions, Windows/Linux FFmpeg/ffprobe binaries from [ffmpeg-static release b6.1.1](https://github.com/eugeneware/ffmpeg-static/releases/tag/b6.1.1), and the exact FFmpeg n7.0.2 source/configuration used on macOS ARM64. The helper also verifies ExifTool's npm-maintainer SHA-512 integrity. Exact extractor/decoder versions, archive identities and source-build receipts accompany qualification.

ExifTool is distributed under the same terms as Perl (Artistic License or GPL). The [PhotoStructure vendored packaging](https://github.com/photostructure/exiftool-vendored.exe) uses the MIT License. Qualification retains every package support file and notice, verifies all support-file digests, disables inherited ExifTool configuration and Perl injection options, and explicitly hashes/selects the Unix Perl executable.

Windows/Linux FFmpeg/ffprobe distributions retain their GPL notices, build/source README and exact core source archive. The upstream release is a distribution identity; platform binaries can report different exact versions. Complete corresponding source for enabled external libraries remains a prerequisite for public binary distribution under release delivery (#14); notices and the FFmpeg archive alone do not fulfill that step. CI uploads receipts/inventories/checksums, not package archives.

The macOS build disables GPL/nonfree options, automatic external-library discovery and networking. Its exact-source static companions use built-in codecs and the system VideoToolbox framework. Packages retain source, configuration and emitted build/version information. The runtime manifest binds every actual binary digest/version. Deterministic qualification uses bounded probing, PCM extraction and preview encoding, with no transcription, diarization, model loading or weight downloads.

Native package assembly includes elected companion bytes/support trees and notices. Qualification is separate from official signing, release promotion and public corresponding-source distribution.
