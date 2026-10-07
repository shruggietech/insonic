#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Offline fixture integrity and bounded decode/probe; never invokes inference."""
import argparse
from decimal import Decimal
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import tempfile

from process_tree import ProcessTree

ROOT = Path(__file__).resolve().parent.parent
FIXTURES = ROOT / 'tests/fixtures/media'


def sha(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def fixture_path(value):
    path = PurePosixPath(value)
    if path.is_absolute() or '..' in path.parts or '\\' in value or ':' in value:
        raise ValueError('fixture path escapes committed directory')
    target = FIXTURES.joinpath(*path.parts)
    if not target.resolve().is_relative_to(FIXTURES.resolve()) or target.is_symlink():
        raise ValueError('fixture path escapes committed directory')
    return target


def checked_file(value):
    path = fixture_path(value['path'])
    if not re.fullmatch(r'[0-9a-f]{64}', value['sha256']) or sha(path) != value['sha256']:
        raise ValueError('fixture/reference checksum mismatch')
    if path.stat().st_size != value['size_bytes']:
        raise ValueError('fixture/reference byte length mismatch')
    with path.open('rb') as source:
        if source.read(100).startswith(b'version https://git-lfs.github.com/spec/v1'):
            raise ValueError('fixture is an LFS pointer')
    return path


def timestamp(milliseconds, vtt=False):
    hours, value = divmod(milliseconds, 3600000)
    minutes, value = divmod(value, 60000)
    seconds, fraction = divmod(value, 1000)
    return f'{hours:02}:{minutes:02}:{seconds:02}{"." if vtt else ","}{fraction:03}'


def subtitle_bytes(reference, vtt=False):
    output = 'WEBVTT\n\n' if vtt else ''
    output += '\n\n'.join(f"{cue['id']}\n{timestamp(cue['start_ms'], vtt)} --> {timestamp(cue['end_ms'], vtt)}\n{cue['text']}"
                          for cue in reference['cues']) + '\n\n'
    return output.encode('utf-8')


def validate(manifest=None):
    if manifest is None:
        manifest = json.loads((FIXTURES / 'manifest.json').read_text(encoding='utf-8'))
    if manifest['kind'] != 'licensed-media-fixtures' or len(manifest['assets']) != 2:
        raise ValueError('two licensed real fixtures required')
    total = 0
    if {asset['id'] for asset in manifest['assets']} != {'speech', 'sintel-dialogue'}:
        raise ValueError('missing fixture identity')
    for asset in manifest['assets']:
        checked_file(asset)
        total += asset['size_bytes']
        if asset['license']['spdx'] != 'CC-BY-3.0' or not asset['creator'] or not asset['attribution']:
            raise ValueError('fixture redistribution attribution missing')
        if not asset['source']['page_revision_url'] or not re.fullmatch(r'[0-9a-f]{64}', asset['source']['sha256']):
            raise ValueError('source rendition identity missing')
        duration = asset['media']['duration_ns']
        if type(duration) is not int or duration <= 0:
            raise ValueError('fixture duration must be measured integer nanoseconds')
        media = asset['media']
        if type(media['sample_rate']) is not int or media['sample_rate'] <= 0 or type(media['channels']) is not int or not 1 <= media['channels'] <= 6:
            raise ValueError('fixture measured stream is invalid')
        if media['samples_per_channel'] * 1000000000 != duration * media['sample_rate']:
            raise ValueError('fixture sample clock differs')
        clock = asset['source_clock']
        if clock['rate_numerator'] != 1 or clock['rate_denominator'] != 1:
            raise ValueError('fixture crop must preserve clock rate')
        if clock['sample_tick_rate'] != media['sample_rate'] or clock['sample_tick_origin'] * 1000000000 != clock['origin_ns'] * media['sample_rate']:
            raise ValueError('fixture source sample origin differs')
        if asset['id'] == 'speech':
            if clock['origin_ns'] != 0 or asset['sha256'] != asset['source']['sha256']:
                raise ValueError('speech sample source bytes/clock changed')
        else:
            adaptation = asset['adaptation']
            if clock['origin_ns'] != adaptation['source_start_ns'] or adaptation['source_end_ns'] - adaptation['source_start_ns'] != duration:
                raise ValueError('crop origin/duration mismatch')
        reference = None
        for item in asset['references']:
            path = checked_file(item)
            if path.suffix == '.json':
                reference = json.loads(path.read_text(encoding='utf-8'))
        if reference is None or reference.get('test_oracle_only') is not True or not reference['source_evidence']:
            raise ValueError('independent reference evidence missing')
        for cue in reference['cues']:
            if any(type(cue[key]) is not int for key in ['start_ms', 'end_ms']):
                raise ValueError('reference cue time must be an integer')
            if not 0 <= cue['start_ms'] < cue['end_ms'] <= duration // 1000000:
                raise ValueError('reference cue outside fixture')
            if asset['id'] == 'sintel-dialogue':
                for key in ['start', 'end']:
                    if cue[key + '_ms'] * 1000000 + clock['origin_ns'] != cue['source_' + key + '_ms'] * 1000000:
                        raise ValueError('published subtitle clock mismatch')
        for item in asset['references']:
            path = fixture_path(item['path'])
            if path.suffix in ['.srt', '.vtt'] and path.read_bytes() != subtitle_bytes(reference, path.suffix == '.vtt'):
                raise ValueError('supplied subtitles differ from independent oracle')
    if total > 10 << 20:
        raise ValueError('committed media exceeds ten MiB')
    return {'fixtures': 2, 'media_bytes': total, 'references': 'passed', 'inference': 'not invoked'}


def child(args):
    with ProcessTree([str(arg) for arg in args], cwd=ROOT, env=os.environ.copy()) as tree:
        try:
            output, errors = tree.process.communicate(timeout=60)
        except subprocess.TimeoutExpired:
            tree.kill()
            tree.process.communicate(timeout=5)
            raise
        if tree.process.returncode:
            raise RuntimeError('bounded media fixture child failed: ' + errors.decode(errors='replace')[:4000])
        if len(output) > 1 << 20:
            raise ValueError('media fixture child output exceeds bound')
        return output


def native(tools_path):
    result = validate()
    tools = json.loads(Path(tools_path).read_text(encoding='utf-8'))
    for name in ['ffmpeg', 'ffprobe']:
        tool = tools[name]
        executable = Path(tool['path'])
        if not executable.is_absolute() or sha(executable) != tool['sha256']:
            raise ValueError('native fixture tool identity mismatch')
        version = child([executable, '-version']).decode().splitlines()[0].split()[2]
        if version != tool['version']:
            raise ValueError('native fixture tool version mismatch')
    manifest = json.loads((FIXTURES / 'manifest.json').read_text(encoding='utf-8'))
    measurements = []
    with tempfile.TemporaryDirectory(prefix='insonic-fixture-') as scratch:
        for asset in manifest['assets']:
            media = asset['media']
            source = fixture_path(asset['path'])
            probe = json.loads(child([tools['ffprobe']['path'], '-v', 'error', '-protocol_whitelist', 'file,pipe', '-count_frames', '-show_streams', '-show_format', '-of', 'json', source]))
            if Decimal(probe['format']['duration']) * 1000000000 != media['duration_ns']:
                raise ValueError('native measured duration differs')
            selected = next(stream for stream in probe['streams'] if stream['index'] == media['audio_stream_index'])
            if selected['codec_name'] != media['codec'] or int(selected['sample_rate']) != media['sample_rate'] or selected['channels'] != media['channels']:
                raise ValueError('native audio stream differs')
            if 'video' in media:
                video = next(stream for stream in probe['streams'] if stream['codec_type'] == 'video')
                expected = media['video']
                if video['codec_name'] != expected['codec'] or video['width'] != expected['width'] or video['height'] != expected['height'] or int(video['nb_read_frames']) != expected['decoded_frames']:
                    raise ValueError('native video stream differs')
            pcm = Path(scratch) / (asset['id'] + '.pcm')
            args = [tools['ffmpeg']['path'], '-hide_banner', '-nostdin', '-loglevel', 'error', '-y', '-threads', '1',
                    '-protocol_whitelist', 'file,pipe', '-i', source, '-map', '0:' + str(media['audio_stream_index']),
                    '-t', str(Decimal(media['duration_ns']) / 1000000000), '-c:a', 'pcm_s16le', '-f', 's16le', pcm]
            child(args)
            expected_size = media['samples_per_channel'] * media['channels'] * 2
            if pcm.stat().st_size != expected_size:
                raise ValueError('native exact PCM sample count differs')
            if asset['id'] == 'sintel-dialogue' and sha(pcm) != asset['adaptation']['pcm_verification']['sha256']:
                raise ValueError('native source-clock PCM differs')
            full_digest = sha(pcm)
            channel = 2 if media['channels'] == 6 else 0
            mapped = Path(scratch) / (asset['id'] + '-mapped.pcm')
            child(args[:-5] + ['-af', 'pan=mono|c0=c' + str(channel), '-ar', '16000', '-c:a', 'pcm_s16le', '-f', 's16le', mapped])
            if mapped.stat().st_size * 1000000000 != media['duration_ns'] * 16000 * 2:
                raise ValueError('native mapped extraction sample count differs')
            measurements.append({'id': asset['id'], 'pcm_sha256': full_digest, 'samples_per_channel': media['samples_per_channel'],
                                 'selected_channel': channel, 'mapped_sample_rate': 16000, 'mapped_pcm_sha256': sha(mapped)})
    result.update({'native_media': 'passed', 'measurements': measurements, 'tools': {name: tools[name]['version'] for name in ['ffmpeg', 'ffprobe']}})
    receipt = ROOT / 'build/native/media-fixture-receipt.json'
    receipt.parent.mkdir(parents=True, exist_ok=True)
    receipt.write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--native', action='store_true', help='bounded pinned decode/probe, without inference')
    parser.add_argument('--tools', default=str(ROOT / 'build/native/media-tools.json'))
    args = parser.parse_args()
    print(json.dumps(native(args.tools) if args.native else validate()))
