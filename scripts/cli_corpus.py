#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Small maintained fixture index and value-free installed CLI coverage."""
import ast
from collections import Counter
import hashlib
import json
from pathlib import Path, PurePosixPath
import platform
import re
from threading import Lock

ROOT = Path(__file__).resolve().parents[1]
CATALOG = Path('tests/fixtures/cli-corpus.json')
ID = re.compile(r'[a-z][a-z0-9-]{0,63}')
FLAG = re.compile(r'(?:--[a-z][a-z0-9-]{0,63}|-h)')
ASSERTIONS = {'materialized-bytes', 'durable-completion', 'owned-runtime-ready'}
ERROR_FIELDS = json.loads((ROOT / 'schemas' / ('v' + (ROOT / 'VERSION').read_text().strip()) / 'runtime-response.schema.json').read_text(encoding='utf-8'))['properties']['error']['properties']


def failure_class(error):
    if error is None: return {'code': 'unclassified'}
    result = {}
    for key in ['code', 'reason']:
        if key not in error and key == 'reason': continue
        if error.get(key) not in ERROR_FIELDS[key]['enum']: raise ValueError('Unknown failure classification')
        result[key] = error[key]
    return result


def unique_object(pairs):
    value = {}
    for key, item in pairs:
        if key in value: raise ValueError('Duplicate corpus member')
        value[key] = item
    return value


def shape(value, required, optional=()):
    if not isinstance(value, dict) or not set(required) <= value.keys() or value.keys() - set(required) - set(optional):
        raise ValueError('Invalid corpus shape')


def label(value, pattern=ID):
    if not isinstance(value, str) or not pattern.fullmatch(value): raise ValueError('Invalid corpus identifier')


def description(value):
    if not isinstance(value, str) or not value.strip() or len(value) > 512 or any(ord(c) < 32 for c in value):
        raise ValueError('Invalid corpus description')


def owner(value, root):
    shape(value, ['path'], ['symbol', 'sha256'])
    name = value['path']
    if not isinstance(name, str) or len(name) > 256 or not re.fullmatch(r'[A-Za-z0-9_./-]+', name) or PurePosixPath(name).is_absolute() or any(p in ('', '.', '..') for p in name.split('/')):
        raise ValueError('Invalid corpus source path')
    target = (root / name).resolve()
    if not target.is_relative_to(root.resolve()) or not target.is_file(): raise ValueError('Corpus source escapes or is missing')
    if 'sha256' in value:
        label(value['sha256'], re.compile('[a-f0-9]{64}'))
        with target.open('rb') as stream: actual = hashlib.file_digest(stream, 'sha256').hexdigest()
        if actual != value['sha256']: raise ValueError('Pinned corpus source differs')
    if 'symbol' in value:
        label(value['symbol'], re.compile(r'[A-Za-z_][A-Za-z0-9_]{0,63}'))
        if target.suffix != '.py' or target.stat().st_size > 2 << 20: raise ValueError('Invalid corpus assertion owner')
        tree = ast.parse(target.read_text(encoding='utf-8'))
        if not any(isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef)) and n.name == value['symbol'] for n in ast.walk(tree)):
            raise ValueError('Corpus assertion owner is missing')


def load_corpus(root=ROOT, path=None):
    root = Path(root)
    path = Path(path) if path is not None else root / CATALOG
    if not path.resolve().is_relative_to(root.resolve()) or path.stat().st_size > 65536: raise ValueError('Invalid corpus file bound')
    with path.open('rb') as stream: data = stream.read(65537)
    if len(data) > 65536: raise ValueError('Invalid corpus file bound')
    try: value = json.loads(data.decode('utf-8'), object_pairs_hook=unique_object)
    except (UnicodeError, json.JSONDecodeError): raise ValueError('Invalid corpus encoding or JSON') from None
    shape(value, ['kind', 'version', 'fixtures', 'scenarios'])
    if value['kind'] != 'cli-qualification-corpus' or type(value['version']) is not int or value['version'] != 1: raise ValueError('Unsupported corpus version')
    for key, limit in [('fixtures', 100), ('scenarios', 32)]:
        if not isinstance(value[key], list) or not 1 <= len(value[key]) <= limit: raise ValueError('Invalid corpus inventory bound')
        seen = set()
        for item in value[key]:
            if not isinstance(item, dict): raise ValueError('Invalid corpus item')
            label(item.get('id'))
            if item['id'] in seen: raise ValueError('Duplicate corpus identifier')
            seen.add(item['id'])
    fixtures = {f['id'] for f in value['fixtures']}
    for item in value['fixtures']:
        shape(item, ['id', 'owners', 'license', 'provenance', 'expectations'])
        if not isinstance(item['owners'], list) or not 1 <= len(item['owners']) <= 8: raise ValueError('Invalid fixture owners')
        for reference in item['owners']: owner(reference, root)
        for key in ['license', 'provenance']: description(item[key])
        if not isinstance(item['expectations'], list) or not 1 <= len(item['expectations']) <= 16: raise ValueError('Invalid fixture expectations')
        for expectation in item['expectations']: description(expectation)
    for item in value['scenarios']:
        shape(item, ['id', 'owner', 'fixtures'])
        owner(item['owner'], root)
        if not isinstance(item['fixtures'], list) or not item['fixtures'] or not all(isinstance(f, str) and f in fixtures for f in item['fixtures']) or len(set(item['fixtures'])) != len(item['fixtures']):
            raise ValueError('Invalid scenario fixture references')
    return value


def installed_authority(help_pages):
    if not isinstance(help_pages, dict) or not help_pages: raise ValueError('Missing installed help authority')
    authority = {}
    for name, page in help_pages.items():
        if not isinstance(name, str) or len(name) > 128 or not re.fullmatch(r'[a-z][a-z0-9-]*(?: [a-z][a-z0-9-]*)*', name): raise ValueError('Invalid installed command')
        options, headers, section = {}, set(), None
        for line in page.splitlines():
            if line in ('Global options:', 'Command options:'):
                if line in headers: raise ValueError('Duplicate installed options section')
                headers.add(line); section = line
            elif section and line.startswith('  -'):
                if ';' not in line: raise ValueError('Malformed installed option metadata')
                head, detail = line.strip().split(';', 1)
                flag = head.split()[0]
                if not FLAG.fullmatch(flag) or flag in options: raise ValueError('Invalid or duplicate installed option')
                options[flag] = {'value': len(head.split()) > 1, 'required': bool(re.search(r'Status: required\b', detail)),
                                 'repeatable': 'Repeat: repeatable' in detail, 'scope': 'global' if section == 'Global options:' else 'command'}
            elif line and not line.startswith(' '): section = None
        if headers != {'Global options:', 'Command options:'}: raise ValueError('Incomplete installed help authority')
        authority[name] = options
    return authority


def example_outcome(result, error, *, completed=None, materialized=False):
    failure = error or (completed or {}).get('error')
    if failure:
        return {'outcome': 'refusal-observed', 'assertion': 'allowed-refusal-category', 'failure_code': failure['code']}
    if completed is not None:
        if completed.get('result', {}).get('state') != 'succeeded': raise ValueError('Example work has no asserted terminal success')
        return {'outcome': 'asserted-success', 'assertion': 'durable-completion'}
    if materialized: return {'outcome': 'asserted-success', 'assertion': 'materialized-bytes'}
    return {'outcome': 'invocation-only', 'assertion': 'stream-exit-envelope'}


def finish_server(server, thread):
    primary = None
    for cleanup in [server.shutdown, server.server_close, lambda: thread.join(timeout=5)]:
        try: cleanup()
        except BaseException as error:
            if primary is None: primary = error
    if thread.is_alive(): raise ValueError('Owned fixture server did not stop') from primary
    if primary is not None: raise primary


def observation_row(**fields):
    return {**fields, 'invocations': 0, 'successful_exits': 0, 'failed_exits': 0, 'parser_rejections': 0, 'help_checks': 0, 'failures': Counter(), 'assertions': set(), 'scenarios': set()}


class Coverage:
    def __init__(self, help_pages, executable, version, artifact_identity=None, *, root_help=''):
        self.authority = installed_authority(help_pages)
        self.corpus = load_corpus()
        self.identity = dict(artifact_identity or {'scope': 'source'})
        shape(self.identity, ['scope'], ['archive_sha256', 'inventory_sha256', 'source_revision', 'executable_sha256'])
        if self.identity['scope'] not in ('source', 'package'): raise ValueError('Invalid qualification scope')
        for key, value in self.identity.items():
            if key != 'scope': label(value, re.compile('[a-f0-9]{40}' if key == 'source_revision' else '[a-f0-9]{64}'))
        if self.identity['scope'] == 'package' and not all(k in self.identity for k in ['archive_sha256', 'inventory_sha256']): raise ValueError('Incomplete package inventory identity')
        if not isinstance(version, str) or not re.fullmatch(r'\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?', version) or len(version) > 64: raise ValueError('Invalid product version')
        with Path(executable).open('rb') as stream: digest = hashlib.file_digest(stream, 'sha256').hexdigest()
        if self.identity.get('executable_sha256', digest) != digest: raise ValueError('CLI image differs from elected identity')
        self.identity.update(executable_sha256=digest, version=version, platform=platform.system().lower(), architecture=platform.machine().lower())
        self.rows = {name: observation_row(command=name, options={key: observation_row(**meta) for key, meta in options.items()}) for name, options in self.authority.items()}
        self.arity = {}
        for options in self.authority.values():
            for key, meta in options.items():
                if key in self.arity and self.arity[key] != meta['value']: raise ValueError('Installed option arity differs')
                self.arity[key] = meta['value']
        self.version_alias = '--version/version' in root_help and 'version' in self.rows
        if self.version_alias:
            self.arity['--version'] = False
            self.rows['version']['options']['--version'] = observation_row(value=False, required=False, repeatable=False, scope='alias')
        self.controls = Counter({'help-only': 0, 'parser-rejections': 0, 'literal-separator': 0, 'equals-value': 0, 'repeated-option': 0})
        self.failures = Counter()
        self.seen, self.total, self.lock = Counter(), 0, Lock()

    def parse(self, argv, code, *, help_only=False):
        words, flags, literal, equals = [], [], False, False
        args = [str(a) for a in argv]; index = 0
        while index < len(args):
            arg = args[index]; index += 1
            if arg == '--' and not literal: literal = True; continue
            if literal: words.append(arg); continue
            if not arg.startswith('-'): words.append(arg); continue
            key, sep, _ = arg.partition('=')
            if key not in self.arity:
                if code == 2: continue
                raise ValueError('Observation uses an unknown accepted option')
            flags.append(key); equals |= bool(sep)
            if self.arity[key] and not sep: index += 1
            if key == '--version': words.append('version')
        help_only |= words[:1] == ['help'] or any(k in flags for k in ['--help', '-h'])
        if words[:1] == ['help']: words = words[1:]
        matches = [name for name in self.rows if words[:len(name.split())] == name.split()]
        if not matches:
            if help_only: return '', sorted(set(flags)), literal, equals, False, True
            if code == 2: return None
            raise ValueError('Observation uses an unknown accepted command')
        command = max(matches, key=lambda name: len(name.split()))
        if any(key not in self.rows[command]['options'] for key in flags):
            if code == 2: return None
            raise ValueError('Observation option is not accepted by command')
        return command, sorted(set(flags)), literal, equals, len(flags) != len(set(flags)), help_only

    def observe(self, argv, code, *, scenario='core', failure=None, help_only=False):
        if scenario not in {s['id'] for s in self.corpus['scenarios']}: raise ValueError('Unknown observation scenario')
        category = tuple(sorted(failure_class(failure).items())) if code else None
        parsed = self.parse(argv, code, help_only=help_only)
        with self.lock:
            if parsed is not None and parsed[-1]:
                command, flags, *_ = parsed
                self.controls['help-only'] += 1
                if command:
                    self.rows[command]['help_checks'] += 1
                    self.rows[command]['scenarios'].add(scenario)
                    for flag in flags:
                        self.rows[command]['options'][flag]['help_checks'] += 1
                        self.rows[command]['options'][flag]['scenarios'].add(scenario)
                return
            if self.total >= 20000: raise ValueError('CLI coverage observation bound exceeded')
            self.total += 1
            if code == 2: self.controls['parser-rejections'] += 1
            if category is not None: self.failures[category] += 1
            if parsed is None: return
            command, flags, literal, equals, repeated, _ = parsed
            self.controls['literal-separator'] += int(literal)
            self.controls['equals-value'] += int(equals)
            self.controls['repeated-option'] += int(repeated)
            self.seen[(command, tuple(flags), code == 0, scenario)] += 1
            kind = 'successful_exits' if code == 0 else 'parser_rejections' if code == 2 else 'failed_exits'
            self.rows[command]['scenarios'].add(scenario)
            if category is not None: self.rows[command]['failures'][category] += 1
            self.rows[command]['invocations'] += 1
            self.rows[command][kind] += 1
            for flag in flags:
                self.rows[command]['options'][flag]['scenarios'].add(scenario)
                if category is not None: self.rows[command]['options'][flag]['failures'][category] += 1
                self.rows[command]['options'][flag]['invocations'] += 1
                self.rows[command]['options'][flag][kind] += 1

    def assert_example(self, entry):
        parsed = self.parse(entry['example'], 0)
        if parsed is None or parsed[0] != entry['command']: raise ValueError('Example assertion command differs')
        command, flags, *_ = parsed
        assertion = entry.get('assertion')
        if entry['outcome'] != 'asserted-success': return
        if assertion not in ASSERTIONS: raise ValueError('Unsupported semantic success assertion')
        if assertion == 'owned-runtime-ready' and command == 'runtime serve': self.observe(entry['example'], 0, scenario='examples')
        if not any(name == command and succeeded and set(flags) <= set(observed) for name, observed, succeeded, scenario in self.seen if scenario == 'examples'): raise ValueError('Semantic assertion has no successful invocation')
        self.rows[command]['assertions'].add(assertion)
        for flag in flags: self.rows[command]['options'][flag]['assertions'].add(assertion)

    def finish(self, completed):
        scenarios = {s['id']: s for s in self.corpus['scenarios']}
        if set(completed) - scenarios.keys(): raise ValueError('Unknown completed corpus scenario')
        def rendered(row):
            return {**{k: v for k, v in row.items() if k not in ('options', 'assertions', 'scenarios', 'failures')},
                    'status': 'asserted-success' if row['assertions'] else 'invocation-only' if row['successful_exits'] else 'refusal-observed' if row['failed_exits'] else 'parser-only' if row['parser_rejections'] else 'help-only' if row['help_checks'] else 'unverified',
                    'failures': [{**dict(key), 'count': count} for key, count in sorted(row['failures'].items())],
                    'scenarios': sorted(row['scenarios']), 'assertions': sorted(row['assertions']), 'assertion_scenarios': ['examples'] if row['assertions'] else []}
        commands = []
        for row in self.rows.values():
            commands.append({**rendered(row), 'options': {key: rendered(option) for key, option in row['options'].items()}})
        value = {'kind': 'installed-cli-coverage', 'version': 1, 'identity': self.identity,
                 'corpus_sha256': hashlib.sha256((ROOT / CATALOG).read_bytes()).hexdigest(),
                 'controls': dict(self.controls), 'observations': self.total,
                 'failures': [{**dict(key), 'count': count} for key, count in sorted(self.failures.items())],
                 'scenarios': [{'id': name, 'state': 'passed' if completed.get(name) else 'unverified', 'owner': scenario['owner'], 'fixtures': scenario['fixtures']} for name, scenario in scenarios.items()],
                 'commands': commands, 'limits': 'Synthetic contracts only; inference, external authorization and owner-installed behavior are unverified.'}
        if len(json.dumps(value, ensure_ascii=False).encode('utf-8')) > 1 << 20: raise ValueError('CLI coverage ledger bound exceeded')
        return value


if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser(description=__doc__);parser.add_argument('--check', action='store_true', required=True);parser.parse_args()
    corpus = load_corpus()
    print(json.dumps({'version': corpus['version'], 'fixtures': len(corpus['fixtures']), 'scenarios': len(corpus['scenarios'])}))
