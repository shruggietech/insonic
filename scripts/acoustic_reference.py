# SPDX-License-Identifier: Apache-2.0
"""Frozen acoustic references and independent metrics; imports no engines."""
import hashlib
import json
import math
from pathlib import Path
import re
import stat
import unicodedata

ARCHIVE_SHA256 = "76f87d090650617fca0cac8f88b9416e0ebf80350acb97b343a85fa903728ab3"
FROZEN_MANIFEST_SHA256 = "5abce92ea60ff0de4639161c85af383457f4e5b36c3ff6c00cfa018631dd6ada"
NORMALIZATION = "NFC, Unicode casefold, Unicode word characters"


def sha(path):
    with Path(path).open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("Duplicate JSON field")
        result[key] = value
    return result


def shape(value, fields):
    if not isinstance(value, dict) or set(value) != set(fields.split()):
        raise ValueError("Reference field shape differs")


def integer(value, minimum, maximum):
    return type(value) is int and minimum <= value <= maximum


def file_bytes(root, name, size, digest, maximum=2 << 20):
    if not isinstance(name, str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*", name):
        raise ValueError("Reference path escapes its corpus")
    if not integer(size, 1, maximum) or not isinstance(digest, str) or not re.fullmatch("[0-9a-f]{64}", digest):
        raise ValueError("Reference file identity differs")
    target = root / name
    try:
        if target.is_symlink() or target.resolve().parent != root or not stat.S_ISREG(target.stat().st_mode):
            raise ValueError("Reference path is not an owned regular corpus file")
        if target.stat().st_size != size:
            raise ValueError("Reference file size differs")
        data = target.read_bytes()
    except OSError as error:
        raise ValueError("Reference file identity is unavailable") from error
    if len(data) != size or hashlib.sha256(data).hexdigest() != digest:
        raise ValueError("Reference file hash differs")
    return data


def flac_identity(data):
    if len(data) < 42 or data[:4] != b"fLaC" or data[4] & 127 or int.from_bytes(data[5:8], "big") != 34:
        raise ValueError("Original FLAC sample identity differs")
    packed = int.from_bytes(data[18:26], "big")
    rate, channels, bits, samples = packed >> 44, ((packed >> 41) & 7) + 1, ((packed >> 36) & 31) + 1, packed & ((1 << 36) - 1)
    if (rate, channels, bits) != (16000, 1, 16):
        raise ValueError("Original FLAC sample format differs")
    return samples, rate


def load_frozen_corpus(path):
    return load_corpus(path, frozen=True)


def load_corpus(path, *, frozen=False):
    try:
        path = Path(path).resolve(strict=True)
        with path.open("rb") as stream:
            data = stream.read(65537)
        if len(data) > 65536:
            raise ValueError("Reference manifest exceeds its limit")
        if frozen and hashlib.sha256(data).hexdigest() != FROZEN_MANIFEST_SHA256:
            raise ValueError("Reference manifest differs from the pinned frozen corpus")
        corpus = json.loads(data.decode("utf-8"), object_pairs_hook=unique_object)
    except (OSError, UnicodeError, json.JSONDecodeError) as error:
        raise ValueError("Reference manifest is unavailable or malformed") from error
    shape(corpus, "format source selection references clips sequence matching")
    if corpus["format"] != "insonic-acoustic-reference-v1":
        raise ValueError("Reference format differs")
    source = corpus["source"]
    shape(source, "url size_bytes sha256 md5 license license_url authors")
    if source != {
        "url": "https://www.openslr.org/resources/12/dev-clean.tar.gz",
        "size_bytes": 337926286, "sha256": ARCHIVE_SHA256,
        "md5": "42e2234ba48799c1f50f24a7926300a1", "license": "CC-BY-4.0",
        "license_url": "https://creativecommons.org/licenses/by/4.0/",
        "authors": ["Vassil Panayotov", "Guoguo Chen", "Daniel Povey", "Sanjeev Khudanpur"],
    }:
        raise ValueError("Reference upstream source identity differs")
    selection = corpus["selection"]
    shape(selection, "enrolled_speakers unseen_speaker min_samples max_samples policy")
    enrolled = selection["enrolled_speakers"]
    if (not isinstance(enrolled, list) or len(enrolled) != 2
            or any(not isinstance(s, str) or not re.fullmatch(r"\d+", s) for s in enrolled) or len(set(enrolled)) != 2
            or not isinstance(selection["unseen_speaker"], str) or not re.fullmatch(r"\d+", selection["unseen_speaker"])
            or selection["unseen_speaker"] in enrolled or selection["min_samples"] != 96000
            or selection["max_samples"] != 320000 or not isinstance(selection["policy"], str) or not selection["policy"]):
        raise ValueError("Reference reader selection differs")
    refs = corpus["references"]
    shape(refs, "path size_bytes sha256")
    root = path.resolve().parent
    text = file_bytes(root, refs["path"], refs["size_bytes"], refs["sha256"], 65536).decode("utf-8")
    references = {}
    for line in text.splitlines():
        parts = line.split(" ", 1)
        if len(parts) != 2 or parts[0] in references or not parts[1].strip():
            raise ValueError("Independent reference lines differ")
        references[parts[0]] = parts[1]
    clips = corpus["clips"]
    if not isinstance(clips, list) or len(clips) != 5:
        raise ValueError("Reference case inventory differs")
    seen = {key: set() for key in ("id", "path", "archive_member", "sha256")}
    roles = {"enrollment": [], "held-out": [], "unseen": []}
    for clip in clips:
        shape(clip, "id path size_bytes samples sample_rate speaker archive_member role transcript sha256")
        match = re.fullmatch(r"(\d+)-(\d+)-(\d+)", clip["id"]) if isinstance(clip["id"], str) else None
        if not isinstance(clip["path"], str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*", clip["path"]):
            raise ValueError("Reference path escapes its corpus")
        if not isinstance(clip["sha256"], str) or not re.fullmatch("[0-9a-f]{64}", clip["sha256"]):
            raise ValueError("Reference file identity differs")
        if (not match or clip["speaker"] != match[1] or clip["path"] != clip["id"] + ".flac"
                or clip["archive_member"] != f"LibriSpeech/dev-clean/{match[1]}/{match[2]}/{clip['id']}.flac"):
            raise ValueError("Reference source identity differs")
        for key in seen:
            if clip[key] in seen[key]:
                raise ValueError("Duplicate reference source or enrollment/evaluation overlap")
            seen[key].add(clip[key])
        if not isinstance(clip["role"], str) or clip["role"] not in roles:
            raise ValueError("Reference role differs")
        roles[clip["role"]].append(clip["speaker"])
        if not integer(clip["samples"], 96000, 320000) or clip["sample_rate"] != 16000:
            raise ValueError("Reference sample identity differs")
        data = file_bytes(root, clip["path"], clip["size_bytes"], clip["sha256"])
        if flac_identity(data) != (clip["samples"], clip["sample_rate"]):
            raise ValueError("Reference sample identity differs from original bytes")
        if references.get(clip["id"]) != clip["transcript"]:
            raise ValueError("Independent reference transcript differs")
    if (set(references) != seen["id"] or sorted(roles["enrollment"]) != sorted(enrolled)
            or sorted(roles["held-out"]) != sorted(enrolled) or roles["unseen"] != [selection["unseen_speaker"]]):
        raise ValueError("Reference role inventory differs")
    sequence = corpus["sequence"]
    shape(sequence, "clips gap_samples")
    held = [clip["id"] for clip in clips if clip["role"] == "held-out"]
    if sequence["clips"] != held or sequence["gap_samples"] != 8000:
        raise ValueError("Reference sequence mapping differs")
    matching = corpus["matching"]
    shape(matching, "threshold ambiguity_margin min_evidence_us")
    for name in ("threshold", "ambiguity_margin"):
        value = matching[name]
        if type(value) not in (float, int) or not math.isfinite(value) or not 0 <= value <= 1:
            raise ValueError("Reference matching setting differs")
    if not integer(matching["min_evidence_us"], 1, 60000000):
        raise ValueError("Reference matching evidence setting differs")
    return corpus


def words(text):
    if not isinstance(text, str) or len(text) > 1 << 20:
        raise ValueError("Reference comparison text limit exceeded")
    result = re.findall(r"\w+", unicodedata.normalize("NFC", text).casefold())
    if len(result) > 8192:
        raise ValueError("Reference comparison word limit exceeded")
    return result


def word_counts(reference, generated):
    expected, actual = words(reference), words(generated)
    if len(expected) * len(actual) > 1000000:
        raise ValueError("Reference comparison cell limit exceeded")
    # Deterministic ties prefer diagonal, deletion, insertion. Retain components
    # rather than estimating them from the final edit distance.
    previous = [(j, 0, 0, j) for j in range(len(actual) + 1)]
    for i, token in enumerate(expected, 1):
        current = [(i, 0, i, 0)]
        for j, value in enumerate(actual, 1):
            diagonal, deletion, insertion = previous[j - 1], previous[j], current[j - 1]
            different = int(token != value)
            choices = [(diagonal[0] + different, diagonal[1] + different, diagonal[2], diagonal[3]),
                       (deletion[0] + 1, deletion[1], deletion[2] + 1, deletion[3]),
                       (insertion[0] + 1, insertion[1], insertion[2], insertion[3] + 1)]
            current.append(min(choices, key=lambda entry: entry[0]))
        previous = current
    distance, substitutions, deletions, insertions = previous[-1]
    return {"reference_words": len(expected), "generated_words": len(actual), "edit_distance": distance,
            "substitutions": substitutions, "deletions": deletions, "insertions": insertions,
            "word_error_rate": distance / len(expected) if expected else None}


def compare_decision(expected, decision, enrolled):
    state = decision.get("state")
    score = decision.get("score")
    if state not in ("accepted", "unknown", "ambiguous"):
        raise ValueError("Speaker decision state differs")
    if score is not None and (type(score) not in (int, float) or not math.isfinite(score)):
        raise ValueError("Speaker decision score is not finite")
    selected = decision.get("speaker_id")
    if state == "accepted" and (not selected or selected not in enrolled.values()):
        raise ValueError("Accepted speaker decision has no elected reader")
    if state == "accepted":
        outcome = "correct_match" if enrolled.get(expected) == selected else "incorrect"
    elif state == "unknown":
        outcome = "unknown" if expected in enrolled else "correct_unknown"
    else:
        outcome = "ambiguous"
    return {"expected_reader": expected, "observed_state": state, "selected_speaker_id": selected,
            "score": score, "outcome": outcome}


def reference_readers(intervals, windows):
    spans = []
    for start, end in intervals:
        if not integer(start, 0, 60000000) or not integer(end, 1, 60000000) or end <= start:
            raise ValueError("Reference interval differs")
        spans.append((start, end))
    merged = []
    for start, end in sorted(spans):
        if merged and start <= merged[-1][1]:
            merged[-1] = (merged[-1][0], max(end, merged[-1][1]))
        else:
            merged.append((start, end))
    previous_end = 0
    result = {}
    for window in sorted(windows, key=lambda entry: entry["start_us"]):
        start, end, speaker = window["start_us"], window["end_us"], window["speaker"]
        if not integer(start, 0, 60000000) or not integer(end, 1, 60000000) or end <= start or start < previous_end:
            raise ValueError("Reference window interval differs")
        previous_end = end
        overlap = sum(max(0, min(right, end) - max(left, start)) for left, right in merged)
        if overlap:
            result[speaker] = result.get(speaker, 0) + overlap
    return result


if __name__ == "__main__":
    import sys
    corpus = load_corpus(sys.argv[1])
    print(json.dumps({"format": corpus["format"], "clips": len(corpus["clips"]), "manifest_sha256": sha(sys.argv[1])}))
