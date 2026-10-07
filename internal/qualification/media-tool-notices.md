# Native extractor qualification inputs

The [media tool lock](media-tools.json) pins archive SHA-256 values for ExifTool 13.59 distributions and the ffprobe binaries published in [ffmpeg-static release b6.1.1](https://github.com/eugeneware/ffmpeg-static/releases/tag/b6.1.1). The helper also verifies ExifTool's npm-maintainer SHA-512 integrity. Package names, exact extractor version output and archive identities appear in qualification receipts.

ExifTool is distributed under the same terms as Perl (Artistic License or GPL). The [PhotoStructure vendored packaging](https://github.com/photostructure/exiftool-vendored.exe) uses the MIT License. Qualification retains every package support file and notice, verifies all support-file digests, disables inherited ExifTool configuration and Perl injection options, and explicitly hashes/selects the Unix Perl executable.

The ffprobe binary distributions carry their platform-specific GPL notices and build/source README. The helper checksum-verifies and retains both beside the selected binary. The upstream release is a distribution identity; its platform binaries can report different exact version suffixes. The generated runtime manifest binds each binary's digest and exact emitted version rather than assuming a shared suffix.

These are controlled native qualification inputs. Installable release assembly remains responsible for shipping the elected distributions and notices with the product.
