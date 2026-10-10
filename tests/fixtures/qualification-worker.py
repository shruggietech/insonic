#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Deterministic maintainer journey adapter. No acoustic model is loaded."""
import json
import sys
import wave

request = json.load(sys.stdin)
with wave.open(request['audio_path'], 'rb') as audio:
    duration = audio.getnframes() * 1000000 // audio.getframerate()
end = min(duration, 5000000)
provenance = {'context_digest': request.get('context_digest', ''), 'offline': True}
operation = request['operation']
if operation == 'transcribe':
    result = {'srt': '1\n00:00:00,000 --> 00:00:05,000\nControlled qualification speech.\n\n',
              'no_speech': False, 'diagnostics': [], 'provenance': provenance}
elif operation == 'diarize':
    result = {'turns': [{'label': 'controlled-voice', 'start_us': 0, 'end_us': end}],
              'no_speech': False, 'diagnostics': [], 'provenance': provenance}
elif operation in ('embed', 'embed-batch'):
    result = {'vectors': [[1.0, 0.0] for _ in request['audio_paths']]} if operation == 'embed-batch' else {'vector': [1.0, 0.0]}
    result.update(provenance={**provenance, 'input_count': len(request['audio_paths']) if operation == 'embed-batch' else 1}, diagnostics=[])
else:
    raise ValueError('unsupported controlled operation')
print(json.dumps(result, allow_nan=False))
