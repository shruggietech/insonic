# Native extractor qualification inputs

The [media tool lock](media-tools.json) pins ExifTool 13.59 distributions and the exact FFmpeg n7.0.2 source/configuration for Windows and Linux x86-64. ExifTool packaging integrity and every retained support-file digest are verified. Qualification records actual extractor/decoder versions, binary identities and corresponding-source build receipts.

ExifTool is distributed under the same terms as Perl (Artistic License or GPL). The [PhotoStructure vendored packaging](https://github.com/photostructure/exiftool-vendored.exe) uses the MIT License. Qualification retains support files and notices, disables inherited ExifTool configuration and Perl injection options, and hashes/selects the Unix Perl executable.

Media companions are built from the pinned source closure with automatic external-library discovery, networking, GPL and nonfree options disabled. Elected external libraries are built from exact retained source archives; their notices, configuration, commands and receipts accompany the binaries. Static companion inventories bind every actual binary digest/version. Deterministic qualification uses bounded probing, PCM extraction and preview encoding, without transcription, diarization, model loading or weight downloads.

Native package assembly includes elected companion support trees, notices and complete corresponding sources. Qualification remains separate from configured publisher signing and official release promotion.
