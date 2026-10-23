# Acoustic reference fixtures

These five original LibriSpeech dev-clean FLACs and their corpus transcript/reader labels are licensed CC BY 4.0. Attribution: Vassil Panayotov, Guoguo Chen, Daniel Povey and Sanjeev Khudanpur, [LibriSpeech](https://www.openslr.org/12/). See [the license](https://creativecommons.org/licenses/by/4.0/) and LICENSE-UPSTREAM.txt.

The manifest pins the original archive and each retained file. Reader IDs identify audiobook readers, not characters. references.txt preserves exact upstream transcript lines; source-metadata.txt retains relevant supplied metadata.

Selection is frozen before inference: readers 84 and 174 enroll on the first eligible 6–20-second utterance of the first numeric chapter; held-out speech uses a different chapter when available. Reader 251 is never enrolled. The derived sequence concatenates both held-out clips with 8000 zero samples at 16 kHz. Original files are unchanged; decoded/concatenated WAVs exist only in owned runtime scratch.

To reproduce, acquire the manifest-pinned official archive and run `python tests/fixtures/acoustic/admit.py ARCHIVE NEW_OUTPUT_DIRECTORY`. The generator verifies SHA-256 before reading archive members. It copies only selected regular original files and supplied references, never arbitrary archive paths.

Book-derived transcript and reader labels are independent of this run. Utterance boundaries include internal silence; they are not manually annotated speech boundaries and do not justify a diarization error-rate claim. Enrollment/evaluation source separation does not establish absence from pretrained model training data. Small clean read-speech measurements are a baseline, not broad accuracy guarantees.
