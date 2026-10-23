# SPDX-License-Identifier: Apache-2.0
"""Reproduce the frozen corpus from one verified official archive snapshot."""
from contextlib import contextmanager
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import stat
import sys
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[3]
spec = importlib.util.spec_from_file_location('admission_reference', ROOT / 'scripts/acoustic_reference.py')
reference = importlib.util.module_from_spec(spec)
spec.loader.exec_module(reference)
ARCHIVE_SHA256 = reference.ARCHIVE_SHA256
ARCHIVE_SIZE = 337926286


@contextmanager
def verified_snapshot(path):
    # The caller's path is opened once. Both parsing passes consume our private
    # verified snapshot, so replacing or modifying the source path cannot swap it.
    with tempfile.TemporaryFile() as snapshot:
        with Path(path).open('rb') as source:
            identity = os.fstat(source.fileno())
            if not stat.S_ISREG(identity.st_mode) or identity.st_size != ARCHIVE_SIZE:
                raise ValueError('Archive size or regular-file identity differs')
            digest, size = hashlib.sha256(), 0
            while block := source.read(1 << 20):
                size += len(block)
                if size > ARCHIVE_SIZE:
                    raise ValueError('Archive exceeds its pinned byte bound')
                digest.update(block)
                snapshot.write(block)
        if size != ARCHIVE_SIZE or digest.hexdigest() != ARCHIVE_SHA256:
            raise ValueError('Archive SHA-256 differs from the pinned official source')
        snapshot.seek(0)
        yield snapshot


def member_bytes(archive, member, maximum):
    if not member.isfile() or not 0 <= member.size <= maximum:
        raise ValueError('Archive member type or size differs')
    with archive.extractfile(member) as stream:
        data = stream.read(maximum + 1)
    if len(data) != member.size:
        raise ValueError('Archive member byte count differs')
    return data


def reproduce(path, output):
    output = Path(output)
    if output.exists():
        raise ValueError('Corpus output already exists')
    rows, texts, metadata = {speaker: [] for speaker in ('84', '174', '251')}, {}, {}
    with verified_snapshot(path) as snapshot:
        with tarfile.open(fileobj=snapshot, mode='r|gz') as archive:
            for member in archive:
                if not member.isfile():
                    continue
                parts = member.name.split('/')
                match = re.fullmatch(r'LibriSpeech/dev-clean/(84|174|251)/(\d+)/(\d+-\d+-\d+)\.flac', member.name)
                if match:
                    speaker, chapter, identifier = match.groups()
                    if not identifier.startswith(speaker + '-' + chapter + '-'):
                        raise ValueError('Archive clip source identity differs')
                    with archive.extractfile(member) as stream:
                        samples, rate = reference.flac_identity(stream.read(42))
                    if rate * 6 <= samples <= rate * 20:
                        rows[speaker].append({'id': identifier, 'path': identifier + '.flac', 'size_bytes': member.size,
                                              'samples': samples, 'sample_rate': rate, 'speaker': speaker,
                                              'archive_member': member.name})
                elif member.name.endswith('.trans.txt') and len(parts) >= 4 and parts[-3] in rows:
                    for line in member_bytes(archive, member, 2 << 20).decode('utf-8').splitlines():
                        key, value = line.split(' ', 1)
                        if key in texts:
                            raise ValueError('Duplicate archive reference identity')
                        texts[key] = value
                elif parts[-1] in ('SPEAKERS.TXT', 'CHAPTERS.TXT', 'LICENSE.TXT', 'README.TXT'):
                    if parts[-1] in metadata:
                        raise ValueError('Duplicate archive metadata identity')
                    metadata[parts[-1]] = member_bytes(archive, member, 2 << 20)
        selected = []
        numeric = lambda row: tuple(map(int, row['id'].split('-')))
        for speaker in ('84', '174'):
            candidates = sorted(rows[speaker], key=numeric)
            if len(candidates) < 2:
                raise ValueError('Archive has insufficient disjoint reader clips')
            enrollment = candidates[0]
            held = next((item for item in candidates if item['id'].split('-')[1] != enrollment['id'].split('-')[1]), candidates[1])
            selected.extend([{**enrollment, 'role': 'enrollment'}, {**held, 'role': 'held-out'}])
        if not rows['251']:
            raise ValueError('Archive has no unseen reader clip')
        selected.append({**sorted(rows['251'], key=numeric)[0], 'role': 'unseen'})
        for row in selected:
            row['transcript'] = texts[row['id']]
        by_name, files = {row['archive_member']: row for row in selected}, {}
        snapshot.seek(0)
        with tarfile.open(fileobj=snapshot, mode='r|gz') as archive:
            for member in archive:
                if member.name in by_name:
                    row = by_name[member.name]
                    if row['path'] in files:
                        raise ValueError('Duplicate selected archive member')
                    data = member_bytes(archive, member, 2 << 20)
                    if len(data) != row['size_bytes']:
                        raise ValueError('Selected clip byte count differs')
                    row['sha256'] = hashlib.sha256(data).hexdigest()
                    files[row['path']] = data
        if len(files) != len(selected):
            raise ValueError('Selected archive inventory is incomplete')
    references = ''.join(row['id'] + ' ' + row['transcript'] + '\n' for row in selected).encode('utf-8')
    manifest = {
        'format': 'insonic-acoustic-reference-v1',
        'source': {'url': 'https://www.openslr.org/resources/12/dev-clean.tar.gz', 'size_bytes': ARCHIVE_SIZE,
                   'sha256': ARCHIVE_SHA256, 'md5': '42e2234ba48799c1f50f24a7926300a1', 'license': 'CC-BY-4.0',
                   'license_url': 'https://creativecommons.org/licenses/by/4.0/',
                   'authors': ['Vassil Panayotov', 'Guoguo Chen', 'Daniel Povey', 'Sanjeev Khudanpur']},
        'selection': {'enrolled_speakers': ['84', '174'], 'unseen_speaker': '251', 'min_samples': 96000,
                      'max_samples': 320000, 'policy': 'ascending numeric reader/chapter/utterance; first eligible enrollment, first eligible different chapter held-out when available, otherwise next distinct utterance; frozen before inference'},
        'references': {'path': 'references.txt', 'size_bytes': len(references), 'sha256': hashlib.sha256(references).hexdigest()},
        'clips': selected, 'sequence': {'clips': [row['id'] for row in selected if row['role'] == 'held-out'], 'gap_samples': 8000},
        'matching': {'threshold': .75, 'ambiguity_margin': .1, 'min_evidence_us': 1000000},
    }
    lines, chapters = [], {row['id'].split('-')[1] for row in selected}
    for name in ('SPEAKERS.TXT', 'CHAPTERS.TXT'):
        wanted = set(rows) if name == 'SPEAKERS.TXT' else chapters
        for line in metadata[name].decode('utf-8').splitlines():
            if line.startswith(';') or line.split('|', 1)[0].strip() in wanted:
                lines.append(line)
    output.mkdir(parents=True, exist_ok=False)
    for name, data in files.items():
        (output / name).write_bytes(data)
    (output / 'references.txt').write_bytes(references)
    (output / 'LICENSE-UPSTREAM.txt').write_bytes(metadata.get('LICENSE.TXT', b'CC BY 4.0: https://creativecommons.org/licenses/by/4.0/\n'))
    (output / 'source-metadata.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
    (output / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + '\n', encoding='utf-8', newline='\n')
    return manifest


if __name__ == '__main__':
    if len(sys.argv) != 3:
        raise ValueError('Expected official archive and new corpus output directory')
    result = reproduce(Path(sys.argv[1]), Path(sys.argv[2]))
    print(json.dumps({'clips': [row['id'] for row in result['clips']], 'source_sha256': result['source']['sha256']}))
