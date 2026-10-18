#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""Prepare owned versions and verify, publish and promote exact-source releases.

Nothing is published by preparation or candidate validation. Publication and
promotion require explicit commands; their transports are injectable for tests.
"""
import argparse
import datetime
import hashlib
import http.client
import importlib.util
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
REPOSITORY = 'shruggietech/insonic'
PLATFORMS = ('windows_amd64', 'linux_amd64')
HISTORICAL_PLATFORMS = ('windows_amd64', 'linux_amd64', 'darwin_arm64')
VARIANTS = ('cli', 'desktop')
VERSION_PATTERN = r'(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)'
SHA_PATTERN = r'[a-f0-9]{64}'
DEPLOYMENT_ENVIRONMENT = 'documentation-deployment'
AUTOMATION_CREDENTIAL_NAMES = {'GH_TOKEN', 'GITHUB_TOKEN', 'GH_ENTERPRISE_TOKEN', 'GITHUB_PAT',
                               'ACTIONS_RUNTIME_TOKEN', 'ACTIONS_ID_TOKEN_REQUEST_TOKEN', 'SYSTEM_ACCESSTOKEN'}
AUTOMATION_ENVIRONMENT_PREFIXES = ('GH_', 'GITHUB_', 'ACTIONS_', 'RUNNER_')


def write_json(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8', newline='\n')


def read_json(path):
    return json.loads(path.read_text(encoding='utf-8'))


def sha(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def child(args, root=ROOT, environment=None):
    result = subprocess.run([str(a) for a in args], cwd=root, stdin=subprocess.DEVNULL,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=120,
                            creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0,
                            env={**(os.environ if environment is None else environment), 'GH_PROMPT_DISABLED': '1', 'GIT_TERMINAL_PROMPT': '0'})
    if result.returncode:
        raise ValueError('release child operation failed: ' + Path(str(args[0])).name)
    return result.stdout.decode('utf-8').strip()


def versions(root=ROOT):
    version = (root / 'VERSION').read_text(encoding='utf-8').strip()
    if not re.fullmatch(VERSION_PATTERN, version):
        raise ValueError('release version must be a canonical semantic version')
    for directory in ('', 'site', 'desktop/frontend'):
        base = root / directory
        package, lock = read_json(base / 'package.json'), read_json(base / 'package-lock.json')
        if package['version'] != version or lock['version'] != version or lock['packages']['']['version'] != version:
            raise ValueError('software package and lock versions differ')
    if re.search(r'const Version = "([^"]+)"', (root / 'internal/contracts/contracts.go').read_text())[1] != version:
        raise ValueError('runtime contract version differs')
    manifest = read_json(root / 'docs/versions.json')
    if manifest['latest'] != version or not any(v['version'] == version for v in manifest['versions']):
        raise ValueError('documentation version differs')
    schema_root = root / 'schemas' / ('v' + version)
    for path in schema_root.glob('*.schema.json'):
        value = read_json(path)
        if value['$id'] != f'https://raw.githubusercontent.com/{REPOSITORY}/v{version}/schemas/v{version}/{path.name}':
            raise ValueError('schema identity differs from release')
        if value.get('properties', {}).get('kind', {}).get('const') and value.get('properties', {}).get('schema_version', {}).get('const') != version:
            raise ValueError('schema envelope differs from release')
    if not (schema_root / 'master.schema.json').is_file():
        raise ValueError('release master schema is missing')
    return version


def exact_source(revision, root=ROOT, runner=child, tag=None):
    if not re.fullmatch(r'[a-f0-9]{40}', revision or '') or runner(['git', 'rev-parse', 'HEAD'], root) != revision:
        raise ValueError('release source must be the exact full current commit SHA')
    if runner(['git', 'status', '--porcelain', '--untracked-files=normal'], root):
        raise ValueError('release source must be clean')
    if tag is not None and runner(['git', 'rev-parse', tag + '^{commit}'], root) != revision:
        raise ValueError('release tag does not identify the candidate source')


def trusted_history(revision, api, root=ROOT, runner=child):
    if not re.fullmatch(r'[a-f0-9]{40}', revision or ''):
        raise ValueError('trusted release revision must be a full commit SHA')
    repository = api.call('GET', f'repos/{REPOSITORY}')
    branch = api.call('GET', f'repos/{REPOSITORY}/branches/main')
    if (not repository or repository.get('default_branch') != 'main' or not branch
            or branch.get('protected') is not True or branch.get('name') != 'main'
            or not re.fullmatch(r'[a-f0-9]{40}', branch.get('commit', {}).get('sha', ''))):
        raise ValueError('release tooling requires the protected default main branch')
    trusted = branch['commit']['sha']
    exact_source(trusted, root, runner)
    try:
        runner(['git', 'merge-base', '--is-ancestor', revision, trusted], root)
    except ValueError:
        raise ValueError('candidate revision is outside trusted main history') from None
    return {'trusted_revision': trusted, 'revision': revision}


def deployment_environment(api):
    endpoint = f'repos/{REPOSITORY}/environments/{DEPLOYMENT_ENVIRONMENT}'
    environment = api.call('GET', endpoint)
    policy = environment.get('deployment_branch_policy', {}) if environment else {}
    rules = api.call('GET', endpoint + '/deployment-branch-policies') if policy.get('custom_branch_policies') is True else None
    branches = rules.get('branch_policies', []) if rules else []
    if (policy.get('protected_branches') is not False or policy.get('custom_branch_policies') is not True
            or not rules or rules.get('total_count') != 1 or len(branches) != 1
            or branches[0].get('name') != 'main' or branches[0].get('type') != 'branch'):
        raise ValueError('documentation deployment environment must restrict deployment branches to main only')
    return {'deployment_environment': 'verified'}


def rewrite_owned_schema(node, old, new, fields=None, context=None):
    fields = fields or {'schema_version'}
    if isinstance(node, dict):
        return {key: rewrite_owned_schema(value, old, new, fields, key if key in fields else context) for key, value in node.items()}
    if isinstance(node, list):
        return [rewrite_owned_schema(value, old, new, fields, context) for value in node]
    if isinstance(node, str):
        if context in fields and node == old: return new
        if context == 'tag' and node == 'v' + old: return 'v' + new
        if re.fullmatch(r'insonic_' + VERSION_PATTERN + r'_windows_amd64_installer\.msi', node):
            return f'insonic_{new}_windows_amd64_installer.msi'
        if re.fullmatch(r'insonic_' + VERSION_PATTERN + r'_windows_amd64_setup\.exe', node):
            return f'insonic_{new}_windows_amd64_setup.exe'
        package = re.fullmatch(r'insonic_' + VERSION_PATTERN + r'_(windows_amd64|linux_amd64|darwin_arm64)_(cli|desktop)\.(?:zip|tar\.gz)', node)
        if package:
            extension = '.tar.gz' if package[1] == 'linux_amd64' else '.zip'
            return f'insonic_{new}_{package[1]}_{package[2]}{extension}'
        return node.replace(f'https://raw.githubusercontent.com/{REPOSITORY}/v{old}/schemas/v{old}/', f'https://raw.githubusercontent.com/{REPOSITORY}/v{new}/schemas/v{new}/')
    return node


def prepare_schema(value, filename, old, version):
    fields = {'schema_version'} | ({'version', 'product_version', 'documentation_version', 'tag'} if filename == 'release-candidate.schema.json' else set())
    value = rewrite_owned_schema(value, old, version, fields)
    if filename == 'common.schema.json':
        value['description'] = re.sub(r'^Shared logical types for the v' + VERSION_PATTERN + r' contract(?: baseline)?\.',
                                      f'Shared logical types for the v{version} contract.', value['description'])
    return value


def glossary_scan(directory):
    glossary = (directory / 'glossary.md').read_text(encoding='utf-8')
    rows = [line for line in glossary.splitlines() if line.startswith('| ') and 'https://' in line]
    if not rows or any(not re.search(r'\]\([^):]+\.md(?:#[^)]*)?\)', line) for line in rows):
        raise ValueError('glossary entries need learning references and local usage links')
    terms = sorted(set(re.findall(r'`([A-Za-z][A-Za-z0-9_./-]{2,})`|\b([A-Z][A-Z0-9]{1,})\b', '\n'.join(p.read_text(encoding='utf-8') for p in directory.glob('*.md')))))
    return {'documentation_pages': len(list(directory.glob('*.md'))), 'defined_terms': len(rows),
            'technical_terms': sorted(set(a or b for a, b in terms))}


def prepare(version, date, highlights, root=ROOT):
    old = versions(root)
    if not re.fullmatch(VERSION_PATTERN, version) or tuple(map(int, version.split('.'))) <= tuple(map(int, old.split('.'))):
        raise ValueError('preparation requires a newer canonical semantic version')
    datetime.date.fromisoformat(date)
    if not highlights.strip() or len(highlights.split()) > 180:
        raise ValueError('release highlights must be concise and nonempty')
    docs, schemas = root / f'docs/v{version}', root / f'schemas/v{version}'
    if docs.exists() or schemas.exists():
        raise ValueError('version preparation never overwrites an existing version')
    # Build every changed byte before activating the new navigation/version.
    changes = {}
    for directory in ('', 'site', 'desktop/frontend'):
        for filename in ('package.json', 'package-lock.json'):
            target = root / directory / filename
            value = read_json(target); value['version'] = version
            if filename == 'package-lock.json': value['packages']['']['version'] = version
            changes[target] = json.dumps(value, indent=2) + '\n'
    changelog = (root / 'CHANGELOG.md').read_text(encoding='utf-8')
    if '## [Unreleased]\n' not in changelog: raise ValueError('master changelog has no Unreleased section')
    changelog = changelog.replace('## [Unreleased]\n', f'## [Unreleased]\n\n## [{version}] - {date}\n', 1)
    changelog = re.sub(r'(?m)^\[Unreleased\]:.*$', f'[Unreleased]: https://github.com/{REPOSITORY}/compare/v{version}...HEAD', changelog)
    changelog += f'\n[{version}]: https://github.com/{REPOSITORY}/compare/v{old}...v{version}\n'
    changes[root / 'CHANGELOG.md'] = changelog
    # Limit version rewriting to project-owned runtime/UI/tests and fixtures.
    for prefix in ('internal', 'cmd', 'desktop/frontend/src', 'desktop/frontend/tests', 'schemas', 'tests', 'scripts/media-tools.py'):
        base = root / prefix
        paths = [base] if base.is_file() else list(base.rglob('*'))
        for path in paths:
            if not path.is_file() or path.suffix not in ('.go', '.mjs', '.tsx', '.ts', '.py', '.json'): continue
            if path.relative_to(root).as_posix() in ('tests/check.test.mjs', 'tests/ci-scope.test.mjs', 'tests/brand-sync.test.mjs'): continue
            if path.is_relative_to(root / 'tests/fixtures/cueson') or path.is_relative_to(root / 'internal/subtitles'): continue
            if 'schemas' in path.parts and path.parent.name.startswith('v'): continue
            if path.suffix == '.json' and not path.is_relative_to(root / 'tests'): continue
            text = path.read_text(encoding='utf-8')
            # Exact quoted version tokens only; upstream versions and git IDs stay unchanged.
            rewritten = text.replace('"' + old + '"', '"' + version + '"').replace("'" + old + "'", "'" + version + "'").replace(f'https://raw.githubusercontent.com/{REPOSITORY}/v{old}/schemas/v{old}/', f'https://raw.githubusercontent.com/{REPOSITORY}/v{version}/schemas/v{version}/').replace('schemas/v' + old + '/', 'schemas/v' + version + '/').replace('"v' + old + '/', '"v' + version + '/').replace("'v" + old + '/', "'v" + version + '/')
            if rewritten != text: changes[path] = rewritten
    manifest = read_json(root / 'docs/versions.json')
    entry = json.loads(json.dumps(next(e for e in manifest['versions'] if e['version'] == old)))
    entry.update(version=version, status='candidate'); manifest['latest'] = version; manifest['versions'].append(entry)
    changes[root / 'docs/versions.json'] = json.dumps(manifest, indent=2) + '\n'
    frozen = {}
    source_docs = root / f'docs/v{old}'
    for path in source_docs.rglob('*'):
        if not path.is_file() or path.is_symlink(): continue
        target = docs / path.relative_to(source_docs)
        frozen[target] = path.read_bytes()
        if path.suffix == '.md' and 'references' not in path.relative_to(source_docs).parts:
            text = path.read_text(encoding='utf-8')
            # Only owned routes and the product index advance. Upstream tags
            # can share the same numeric version and keep their own authority.
            text = text.replace(f'https://raw.githubusercontent.com/{REPOSITORY}/v{old}/schemas/v{old}/', f'https://raw.githubusercontent.com/{REPOSITORY}/v{version}/schemas/v{version}/')
            text = text.replace(f'schemas/v{old}/', f'schemas/v{version}/').replace(f'docs/v{old}/', f'docs/v{version}/')
            if path.name == 'index.md':
                text = re.sub(r'(?m)^(# insonic v|v)' + re.escape(old) + r'(?=\s|$)', lambda match: match[1] + version, text)
            text = re.sub(r'("schema_version"\s*:\s*")' + re.escape(old) + r'(")', lambda match: match[1] + version + match[2], text)
            if path.name == 'index.md':
                text = re.sub(r'(?m)^v' + re.escape(version) + r' (?:is a specification baseline|documents (?:this release|the prepared release candidate))[^\n]*', f'v{version} documents the prepared release candidate. Official product downloads have not been published; package availability and executed qualification are recorded in its candidate manifest.', text)
            frozen[target] = text.encode('utf-8')
            # Images have a separate versioned capture authority; they are not repository text references.
            reference_text = re.sub(r'!\[[^\]]*\]\([^)]*\)', '', text)
            for relative in re.findall(r'\]\((\.\./[^)#? ]+)', reference_text):
                source = (source_docs / relative).resolve()
                if not source.is_relative_to(root.resolve()) or not source.is_file(): raise ValueError('documentation reference cannot be frozen')
                destination = docs / 'references' / source.relative_to(root.resolve())
                frozen[destination] = source.read_bytes()
    frozen[docs / 'references/CHANGELOG.md'] = changelog.encode('utf-8')
    for source in (root / f'schemas/v{old}').glob('*.json'):
        value = prepare_schema(read_json(source), source.name, old, version)
        frozen[schemas / source.name] = (json.dumps(value, indent=2) + '\n').encode('utf-8')
    changes[root / 'VERSION'] = version + '\n'
    # Roll back all activation files on an I/O or final consistency failure.
    saved = {path: path.read_bytes() for path in changes}
    try:
        for path, raw in frozen.items(): path.parent.mkdir(parents=True, exist_ok=True); path.write_bytes(raw)
        for path, text in changes.items(): path.write_text(text, encoding='utf-8', newline='\n')
        versions(root)
        scan = glossary_scan(docs) if version.split('.')[0] != old.split('.')[0] else None
        report = {'version': version, 'previous_version': old, 'glossary_scan': scan, 'published': False}
        write_json(root / 'build/release/version-preparation.json', report)
        (root / 'build/release/highlights.md').write_text(highlights.strip() + f'\n\n[Full changelog](https://github.com/{REPOSITORY}/blob/v{version}/CHANGELOG.md)\n', encoding='utf-8', newline='\n')
        return report
    except BaseException:
        for path, raw in saved.items(): path.write_bytes(raw)
        for target in (docs, schemas):
            if target.resolve().is_relative_to(root.resolve()) and target.name == 'v' + version and target.exists(): shutil.rmtree(target)
        raise


def file_entry(path):
    if path.is_symlink() or not path.is_file(): raise ValueError('release asset must be a regular file')
    return {'name': path.name, 'size_bytes': path.stat().st_size, 'sha256': sha(path)}


def asset_name(name):
    if not isinstance(name, str) or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]*', name): raise ValueError('candidate asset name is invalid')
    return name


def verify_signing(signing, *, historical=False):
    if not isinstance(signing.get('configured'), bool) or not isinstance(signing.get('verified'), bool): raise ValueError('package signing disposition is incomplete')
    if signing['configured']:
        statuses = ('signed', 'notarized') if historical else ('signed',)
        if signing['verified'] is not True or signing.get('status') not in statuses: raise ValueError('configured package signing was not verified')
    elif signing['verified'] is not False or signing.get('status') != 'unconfigured': raise ValueError('unconfigured package signing cannot claim publisher verification')


def verify_qualification(receipt):
    required = {'extracted_inventory': 'passed', 'relocation_path_spaces': 'passed', 'cli_real_audio_video_import': 'passed', 'development_native_libraries': 'isolated-and-restored'}
    if receipt.get('schema_version') != '1.0.0' and receipt.get('platform') == 'windows_amd64':
        required.update(windows_copy_install='passed', windows_first_use='passed')
    expected_gui = 'passed' if receipt.get('variant') == 'desktop' else 'not-included'
    required.update(gui_bridge=expected_gui, native_webview=expected_gui)
    if any(receipt.get(field) != expected for field, expected in required.items()): raise ValueError('package relocation and native qualification are incomplete')


def package_module():
    spec = importlib.util.spec_from_file_location('release_package', ROOT / 'scripts/package-desktop.py')
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


def verify_package(path, receipt, source_root=ROOT):
    module = package_module()
    module.verify_archive_receipt(path, receipt, expected_source_root=source_root)


def verify_installer(path, receipt, source_root=ROOT):
    spec = importlib.util.spec_from_file_location('release_windows_msi', ROOT / 'scripts/windows_msi.py')
    module = importlib.util.module_from_spec(spec); sys.modules[spec.name] = module; spec.loader.exec_module(module)
    return module.verify_installer(path, receipt, source_root=source_root)


def installer_identity(receipt, version, revision, packages):
    if (receipt.get('kind') != 'windows-installer-receipt' or receipt.get('format') != 'msi'
            or receipt.get('platform') != 'windows_amd64' or receipt.get('variant') != 'selectable'):
        raise ValueError('installer format or platform identity differs')
    if (receipt.get('revision') != revision or receipt.get('source_dirty') is not False
            or any(receipt.get(field) != version for field in ('schema_version', 'product_version', 'documentation_version'))):
        raise ValueError('installer source or version identity differs')
    expected_name = f'insonic_{version}_windows_amd64_installer.msi'
    if receipt.get('artifact') != expected_name:
        raise ValueError('installer artifact format or version differs')
    desktop = next((item for item in packages if item.get('platform') == 'windows_amd64' and item.get('variant') == 'desktop'), None)
    if (desktop is None or receipt.get('payload_receipt') != desktop
            or receipt.get('payload_archive') != desktop['archive']
            or receipt.get('payload_archive_sha256') != desktop['archive_sha256']):
        raise ValueError('installer payload source authority differs from qualified desktop archive')
    required = ['client_selections','license_and_prerequisites','maintenance','upgrade_rollback','downgrade_rejection','active_work_preserved','user_data_preserved','launch_contract','native_ui_capture','cancellation_rollback','user_context']
    if (not isinstance(receipt.get('qualification'), dict)
            or any(receipt['qualification'].get(key) != 'passed' for key in required)):
        raise ValueError('installer qualification is missing or incomplete')
    distribution = receipt.get('distribution', {})
    if (distribution.get('source_complete') is not True or distribution.get('binary_publication') != 'eligible'
            or distribution.get('corresponding_source') != 'included'):
        raise ValueError('installer corresponding sources are incomplete')
    signing = receipt.get('signing', {})
    verify_signing(signing)
    if signing.get('configured') is not False or signing.get('verified') is not False or signing.get('status') != 'unconfigured':
        raise ValueError('MSI is newly authored unsigned; payload signatures cannot establish installer signing')
    setup_identity(receipt, version)


def setup_identity(receipt, version):
    spec=importlib.util.spec_from_file_location('release_windows_setup', ROOT / 'scripts/windows_setup.py')
    windows_setup=importlib.util.module_from_spec(spec);spec.loader.exec_module(windows_setup)
    setup = receipt.get('setup', {})
    expected = {'product_code': receipt.get('product_code'), 'upgrade_code': receipt.get('upgrade_code'),
                'version': version, 'sha256': receipt.get('artifact_sha256'), 'size_bytes': receipt.get('artifact_size_bytes')}
    if (setup.get('kind') != 'windows-setup-receipt' or setup.get('format') != 'exe'
            or setup.get('artifact') != f'insonic_{version}_windows_amd64_setup.exe'
            or setup.get('embedded_msi') != expected or not windows_setup.identity_valid(expected)
            or setup.get('manifest') != 'asInvoker' or setup.get('scope') != 'per-user'
            or setup.get('signing') != {'configured': False, 'verified': False, 'status': 'unconfigured'}):
        raise ValueError('coordinated setup artifact identity differs from the raw MSI')
    if any(setup.get('qualification', {}).get(key) != 'passed' for key in windows_setup.QUALIFICATION_KEYS):
        raise ValueError('coordinated setup recovery qualification is incomplete')


def verify_selected_package(path, receipt, verifier, source_root):
    # Execute the trusted verifier; candidate checkout inputs are data only.
    if verifier is verify_package:
        if receipt.get('kind') == 'windows-installer-receipt':
            verify_installer(path, receipt, source_root)
        else:
            verifier(path, receipt, source_root)
    else:
        verifier(path, receipt)


def verify_documentation_build(marker, version, revision, require_release=False):
    fields = {'version', 'revision', 'source_dirty'}
    if not isinstance(marker, dict) or set(marker) not in (fields, fields | {'documentation_target'}) or marker.get('version') != version or marker.get('revision') != revision or marker.get('source_dirty') is not False:
        raise ValueError('documentation build source differs from candidate')
    target = marker.get('documentation_target', 'snapshot')
    if target not in ('snapshot', 'release'):
        raise ValueError('invalid documentation publication target')
    if require_release and target != 'release':
        raise ValueError('publication requires release-target documentation, not a candidate snapshot')
    return target


def verify_documentation(path, version, revision, require_release=False):
    with tarfile.open(path, 'r:gz') as archive:
        names = set()
        for member in archive.getmembers():
            relative = PurePosixPath(member.name)
            if member.name in names or not member.name or relative.is_absolute() or '..' in relative.parts or '\\' in member.name or ':' in member.name or not member.isfile(): raise ValueError('documentation archive has unsafe members')
            names.add(member.name)
        if not {'release-documentation.json', 'documentation-build.json'}.issubset(names): raise ValueError('documentation identity is missing')
        identity = json.load(archive.extractfile('release-documentation.json'))
        if identity != {'version': version, 'revision': revision, 'kind': path.name.removesuffix('.tar.gz')}: raise ValueError('documentation identity differs from candidate')
        marker = json.load(archive.extractfile('documentation-build.json'))
        return verify_documentation_build(marker, version, revision, require_release)


def candidate(receipts, assets, output, revision, root=ROOT, verifier=verify_package, runner=child):
    version = versions(root); exact_source(revision, root, runner)
    if output.exists(): raise ValueError('candidate output must be new')
    selected = set(); packages = []; installers = []; files = []
    for receipt_file in receipts:
        receipt = read_json(receipt_file)
        if receipt.get('kind') == 'windows-installer-receipt':
            installers.append(receipt); continue
        key = (receipt.get('platform'), receipt.get('variant'))
        if key not in {(p, v) for p in PLATFORMS for v in VARIANTS} or key in selected: raise ValueError('missing, duplicate or unqualified package variant')
        selected.add(key)
        if receipt.get('kind') != 'native-package-receipt' or receipt.get('revision') != revision or receipt.get('source_dirty') is not False:
            raise ValueError('package source identity differs or is dirty')
        if any(receipt.get(field) != version for field in ('schema_version', 'product_version', 'documentation_version')): raise ValueError('package versions differ')
        verify_qualification(receipt)
        distribution = receipt.get('distribution', {})
        if distribution.get('binary_publication') != 'eligible' or distribution.get('source_complete') is not True or distribution.get('corresponding_source') != 'included': raise ValueError('corresponding sources are incomplete')
        signing = receipt.get('signing', {})
        verify_signing(signing)
        name = asset_name(receipt.get('archive'))
        path = assets / name; entry = file_entry(path)
        if entry['sha256'] != receipt.get('archive_sha256') or entry['size_bytes'] != receipt.get('archive_size_bytes'): raise ValueError('package archive differs from receipt')
        verify_selected_package(path, receipt, verifier, root); files.append(entry); packages.append(receipt)
    if selected != {(p, v) for p in PLATFORMS for v in VARIANTS}: raise ValueError('candidate requires all four Windows/Linux package variants')
    if len(installers) != selected_installer_count(root, version):
        raise ValueError('candidate requires exactly one MSI installer; missing or duplicate receipt')
    for receipt in installers:
        installer_identity(receipt, version, revision, packages)
        path = assets / asset_name(receipt['artifact']); entry = file_entry(path)
        if entry['sha256'] != receipt.get('artifact_sha256') or entry['size_bytes'] != receipt.get('artifact_size_bytes'):
            raise ValueError('installer artifact differs from receipt')
        verify_selected_package(path, receipt, verifier, root); files.append(entry)
        setup = receipt['setup']; setup_path = assets / asset_name(setup['artifact']); setup_entry = file_entry(setup_path)
        if setup_entry['sha256'] != setup.get('artifact_sha256') or setup_entry['size_bytes'] != setup.get('artifact_size_bytes'):
            raise ValueError('setup artifact differs from receipt')
        files.append(setup_entry)
    for filename in ('documentation.tar.gz', 'offline-help.tar.gz'):
        files.append(file_entry(assets / filename))
    for filename in ('documentation.tar.gz', 'offline-help.tar.gz'):
        verify_documentation(assets / filename, version, revision)
    value = {'kind': 'release-candidate', 'schema_version': version, 'version': version, 'revision': revision,
             'tag': 'v' + version, 'repository': REPOSITORY, 'packages': packages, 'assets': files,
             'capabilities': {'variants': list(VARIANTS), 'platforms': list(PLATFORMS), 'advanced_cli': ['backup'], 'inference_qualification': 'not-executed'}}
    if installers: value['installers'] = installers
    output.mkdir(parents=True)
    for entry in files:
        shutil.copyfile(assets / entry['name'], output / entry['name'])
        if file_entry(output / entry['name']) != entry: raise ValueError('candidate copy changed during preparation')
    manifest_bytes = (json.dumps(value, indent=2) + '\n').encode('utf-8')
    checksum_files = [*files, {'name': 'release-manifest.json', 'size_bytes': len(manifest_bytes), 'sha256': hashlib.sha256(manifest_bytes).hexdigest()}]
    (output / 'SHA256SUMS').write_text(''.join(f"{entry['sha256']}  {entry['name']}\n" for entry in sorted(checksum_files, key=lambda e: e['name'])), encoding='utf-8', newline='\n')
    # Completion authority is last. An interrupted copy never has a manifest.
    with (output / 'release-manifest.json').open('xb') as completion: completion.write(manifest_bytes)
    return value


def document_archives(output, revision, root=ROOT, runner=child):
    version = versions(root); exact_source(revision, root, runner)
    output.mkdir(parents=True, exist_ok=True)
    for name, directory in [('documentation', root / 'site/out'), ('offline-help', root / 'site/offline')]:
        if not (directory / 'index.html').is_file(): raise ValueError('documentation export is missing')
        verify_documentation_build(read_json(directory / 'documentation-build.json'), version, revision)
        identity = {'version': version, 'revision': revision, 'kind': name}
        with tempfile.TemporaryDirectory() as temporary:
            marker = Path(temporary) / 'release-documentation.json'; write_json(marker, identity)
            with tarfile.open(output / (name + '.tar.gz'), 'w:gz') as archive:
                archive.add(marker, arcname=marker.name)
                for path in sorted(directory.rglob('*')):
                    if path.is_symlink(): raise ValueError('documentation export contains a link')
                    if path.is_file(): archive.add(path, arcname=path.relative_to(directory).as_posix())


def collect(input_directory, output, revision, root=ROOT, verifier=verify_package, runner=child, *, summary=False):
    receipts = [path for path in input_directory.rglob('package-*-receipt.json') if path.name in ('package-cli-receipt.json', 'package-desktop-receipt.json') and read_json(path).get('kind') == 'native-package-receipt']
    receipts += [path for path in input_directory.rglob('windows-installer-receipt.json') if read_json(path).get('kind') == 'windows-installer-receipt']
    with tempfile.TemporaryDirectory(prefix='insonic release aggregation ') as temporary:
        assets = Path(temporary)
        for receipt in receipts:
            value = read_json(receipt)
            name = asset_name(value['artifact'] if value['kind'] == 'windows-installer-receipt' else value['archive'])
            matches = list(input_directory.rglob(name))
            if len(matches) != 1: raise ValueError('downloaded native asset is missing or ambiguous')
            shutil.copyfile(matches[0], assets / name)
            if value['kind']=='windows-installer-receipt' and value.get('setup'):
                setup_name=asset_name(value['setup']['artifact']);setup_matches=list(input_directory.rglob(setup_name))
                if len(setup_matches)!=1:raise ValueError('downloaded setup asset is missing or ambiguous')
                shutil.copyfile(setup_matches[0],assets/setup_name)
        for filename in ('documentation.tar.gz', 'offline-help.tar.gz'):
            matches = list(input_directory.rglob(filename))
            if len(matches) != 1: raise ValueError('downloaded documentation is missing or ambiguous')
            shutil.copyfile(matches[0], assets / filename)
        value = candidate(receipts, assets, output, revision, root, verifier, runner)
        if not summary:
            return value
        return {'kind': 'release-candidate-summary', 'version': value['version'], 'revision': value['revision'],
                'package_count': len(value['packages']), 'asset_count': len(value['assets']),
                'manifest': file_entry(output / 'release-manifest.json')}


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs): return None


class GitHub:
    def __init__(self, token, *, read_only=False):
        if not token and not read_only: raise ValueError('GitHub publication token is required')
        self.token = token; self.read_only = read_only
        self.opener = urllib.request.build_opener(NoRedirect())

    def call(self, method, endpoint, value=None, path=None):
        if self.read_only and (method != 'GET' or value is not None or path is not None):
            raise ValueError('read-only GitHub verification permits GET only')
        url = endpoint if urllib.parse.urlsplit(endpoint).scheme else 'https://api.github.com/' + endpoint
        parsed = urllib.parse.urlsplit(url)
        if parsed.scheme != 'https' or parsed.hostname not in ('api.github.com', 'uploads.github.com') or parsed.username is not None or parsed.port not in (None, 443):
            raise ValueError('GitHub publication endpoint is invalid')
        body = json.dumps(value).encode() if value is not None else None
        headers = {'Accept': 'application/vnd.github+json', 'X-GitHub-Api-Version': '2022-11-28', 'Content-Type': 'application/octet-stream' if path else 'application/json'}
        if self.token: headers['Authorization'] = 'Bearer ' + self.token
        if path:
            headers['Content-Length'] = str(path.stat().st_size)
            connection = http.client.HTTPSConnection(parsed.hostname, timeout=120)
            try:
                with path.open('rb') as source:
                    connection.request(method, parsed.path + ('?' + parsed.query if parsed.query else ''), body=source, headers=headers)
                    response = connection.getresponse()
                    if response.status not in (200, 201): raise ValueError(f'GitHub asset upload failed ({response.status})')
                    return json.load(response)
            finally: connection.close()
        try:
            with self.opener.open(urllib.request.Request(url, data=body, headers=headers, method=method), timeout=120) as response: return json.load(response)
        except urllib.error.HTTPError as error:
            if error.code == 404: return None
            raise ValueError(f'GitHub publication operation failed ({error.code})') from None

    def download_asset(self, asset_id, path, size, digest):
        if type(asset_id) is not int or asset_id <= 0 or type(size) is not int or size < 0 or not re.fullmatch(SHA_PATTERN, digest):
            raise ValueError('published asset metadata is invalid')
        url = f'https://api.github.com/repos/{REPOSITORY}/releases/assets/{asset_id}'
        headers = {'Authorization': 'Bearer ' + self.token, 'Accept': 'application/octet-stream',
                   'X-GitHub-Api-Version': '2022-11-28'}
        started = time.monotonic()
        created = False
        try:
            for attempt in range(4):
                try:
                    response = self.opener.open(urllib.request.Request(url, headers=headers), timeout=30)
                    break
                except urllib.error.HTTPError as error:
                    if error.code not in (301, 302, 303, 307, 308):
                        raise ValueError(f'published asset download failed ({error.code})') from None
                    destination = error.headers.get('Location', '')
                    parsed = urllib.parse.urlsplit(destination)
                    if (parsed.scheme != 'https' or parsed.hostname not in ('release-assets.githubusercontent.com', 'objects.githubusercontent.com')
                            or parsed.username is not None or parsed.port not in (None, 443) or parsed.fragment):
                        raise ValueError('published asset redirect destination is invalid') from None
                    # Signed official CDN URLs need no GitHub token. NoRedirect
                    # prevents urllib from forwarding Authorization implicitly.
                    url, headers = destination, {'Accept': 'application/octet-stream'}
                    error.close()
            else:
                raise ValueError('published asset download has too many redirects')
            with response, path.open('xb') as output:
                created = True
                total = 0
                while True:
                    if time.monotonic() - started > 120:
                        raise ValueError('published asset download exceeded its time budget')
                    data = response.read(min(1024 * 1024, size - total + 1))
                    if not data:
                        break
                    total += len(data)
                    if total > size:
                        raise ValueError('published asset download exceeds its declared size')
                    output.write(data)
            if total != size or sha(path) != digest:
                raise ValueError('published asset bytes differ from their declared identity')
        except (OSError, urllib.error.URLError):
            if created: path.unlink(missing_ok=True)
            raise ValueError('published asset download failed') from None
        except BaseException:
            if created: path.unlink(missing_ok=True)
            raise


def verify_tag(api, tag, revision):
    ref = api.call('GET', f'repos/{REPOSITORY}/git/ref/tags/{tag}')
    if not ref: raise ValueError('release tag must already exist')
    obj = ref['object']
    for _ in range(8):
        if obj['type'] == 'commit':
            if obj['sha'] != revision: raise ValueError('remote tag differs from candidate source')
            return
        if obj['type'] != 'tag': break
        obj = api.call('GET', f'repos/{REPOSITORY}/git/tags/{obj["sha"]}')['object']
    raise ValueError('release tag does not resolve to a commit')


def remote_assets(api, release, expected):
    actual = api.call('GET', f'repos/{REPOSITORY}/releases/{release["id"]}/assets?per_page=100')
    if not isinstance(actual, list) or len(actual) != len(expected): raise ValueError('remote release asset set is incomplete or unexpected')
    for entry in expected:
        matches = [asset for asset in actual if asset['name'] == entry['name']]
        if len(matches) != 1 or matches[0].get('state') != 'uploaded' or matches[0].get('size') != entry['size_bytes'] or matches[0].get('digest') != 'sha256:' + entry['sha256']:
            raise ValueError('remote release asset digest or size differs')


def selected_installer_count(source_root, version):
    schema = read_json(source_root / f'schemas/v{version}/release-candidate.schema.json')
    installers = schema.get('properties', {}).get('installers')
    if installers is None: return 0
    if (installers.get('minItems') != 1 or installers.get('maxItems') != 1
            or installers.get('items', {}).get('properties', {}).get('format', {}).get('const') != 'msi'
            or 'installers' not in schema.get('required', [])):
        raise ValueError('selected release installer format contract differs')
    return 1


def selected_setup_count(source_root, version):
    schema = read_json(source_root / f'schemas/v{version}/release-candidate.schema.json')
    installer = schema.get('properties', {}).get('installers', {}).get('items', {})
    setup = installer.get('properties', {}).get('setup')
    if setup is None: return 0
    if ('setup' not in installer.get('required', [])
            or setup.get('properties', {}).get('format', {}).get('const') != 'exe'):
        raise ValueError('selected release setup format contract differs')
    return selected_installer_count(source_root, version)


def selected_release_platforms(source_root, version):
    # Selected historical checkouts are data; never execute their validation code.
    schema = read_json(source_root / f'schemas/v{version}/release-candidate.schema.json')
    packages = schema['properties']['packages']
    platforms = packages['items']['properties']['platform']['enum']
    allowed = HISTORICAL_PLATFORMS if version == '1.0.0' else PLATFORMS
    count = len(allowed) * len(VARIANTS)
    installer_count = selected_installer_count(source_root, version) + selected_setup_count(source_root, version)
    if (not isinstance(platforms, list) or len(platforms) != len(set(platforms))
            or set(platforms) != set(allowed)
            or packages.get('minItems') != count or packages.get('maxItems') != count
            or schema['properties']['assets'].get('minItems') != count + installer_count + 2
            or schema['properties']['assets'].get('maxItems') != count + installer_count + 2):
        raise ValueError('selected release platform contract differs')
    return tuple(platforms)


def load_candidate(directory, verifier=verify_package, source_root=ROOT):
    value = read_json(directory / 'release-manifest.json')
    if value.get('kind') != 'release-candidate' or value.get('repository') != REPOSITORY or not re.fullmatch(VERSION_PATTERN, value.get('version', '')) or value.get('tag') != 'v' + value['version'] or not re.fullmatch(r'[a-f0-9]{40}', value.get('revision', '')):
        raise ValueError('invalid release candidate identity')
    platforms = selected_release_platforms(source_root, value['version'])
    package_count = len(platforms) * len(VARIANTS)
    capabilities = value.get('capabilities', {})
    if (value.get('schema_version') != value['version']
            or not isinstance(capabilities.get('platforms'), list) or len(capabilities['platforms']) != len(platforms)
            or set(capabilities['platforms']) != set(platforms)
            or not isinstance(capabilities.get('variants'), list) or len(capabilities['variants']) != len(VARIANTS)
            or set(capabilities['variants']) != set(VARIANTS)
            or capabilities.get('inference_qualification') != 'not-executed'):
        raise ValueError('candidate advertised capability matrix differs')
    packages = value.get('packages', [])
    if len(packages) != package_count or {(p.get('platform'), p.get('variant')) for p in packages} != {(p, v) for p in platforms for v in VARIANTS}: raise ValueError('candidate package matrix is incomplete')
    for receipt in packages:
        if receipt.get('revision') != value['revision'] or receipt.get('source_dirty') is not False or any(receipt.get(field) != value['version'] for field in ('schema_version', 'product_version', 'documentation_version')): raise ValueError('candidate package authority differs')
        verify_qualification(receipt)
        distribution = receipt.get('distribution', {})
        if distribution.get('source_complete') is not True or distribution.get('binary_publication') != 'eligible' or distribution.get('corresponding_source') != 'included': raise ValueError('candidate corresponding sources are incomplete')
        signing = receipt.get('signing', {})
        verify_signing(signing, historical=value['version'] == '1.0.0')
    installers = value.get('installers', [])
    installer_count = selected_installer_count(source_root, value['version'])
    if not isinstance(installers, list) or len(installers) != installer_count:
        raise ValueError('candidate MSI installer inventory is missing or duplicate')
    for receipt in installers: installer_identity(receipt, value['version'], value['revision'], packages)
    setup_names = {i['setup']['artifact'] for i in installers} if selected_setup_count(source_root, value['version']) else set()
    asset_names = {p['archive'] for p in packages} | {i['artifact'] for i in installers} | setup_names | {'documentation.tar.gz', 'offline-help.tar.gz'}
    if len(value.get('assets', [])) != package_count + installer_count + len(setup_names) + 2 or {a['name'] for a in value['assets']} != asset_names: raise ValueError('candidate asset matrix differs')
    expected = value['assets'] + [file_entry(directory / 'release-manifest.json'), file_entry(directory / 'SHA256SUMS')]
    if len({entry['name'] for entry in expected}) != len(expected): raise ValueError('duplicate candidate assets')
    for entry in expected:
        asset_name(entry['name'])
        if file_entry(directory / entry['name']) != entry: raise ValueError('candidate bytes changed')
    for receipt in packages:
        archive_entry = next(a for a in value['assets'] if a['name'] == receipt['archive'])
        if archive_entry['sha256'] != receipt.get('archive_sha256') or archive_entry['size_bytes'] != receipt.get('archive_size_bytes'): raise ValueError('candidate receipt differs from archive identity')
        verify_selected_package(directory / receipt['archive'], receipt, verifier, source_root)
    for receipt in installers:
        entry = next(a for a in value['assets'] if a['name'] == receipt['artifact'])
        if entry['sha256'] != receipt.get('artifact_sha256') or entry['size_bytes'] != receipt.get('artifact_size_bytes'):
            raise ValueError('candidate MSI receipt differs from artifact identity')
        verify_selected_package(directory / receipt['artifact'], receipt, verifier, source_root)
        setup = receipt['setup']; entry = next(a for a in value['assets'] if a['name'] == setup['artifact'])
        if entry['sha256'] != setup.get('artifact_sha256') or entry['size_bytes'] != setup.get('artifact_size_bytes'):
            raise ValueError('candidate setup receipt differs from artifact identity')
    for name in ('documentation.tar.gz', 'offline-help.tar.gz'): verify_documentation(directory / name, value['version'], value['revision'])
    sums = ''.join(f"{entry['sha256']}  {entry['name']}\n" for entry in sorted(expected[:-1], key=lambda e: e['name']))
    if (directory / 'SHA256SUMS').read_text() != sums: raise ValueError('candidate checksums differ')
    return value, expected


def published_candidate(output, revision, api, root=ROOT, runner=child, verifier=verify_package, trusted_root=None):
    trusted_history(revision, api, trusted_root or root, runner)
    version = versions(root)
    if version == '0.0.0':
        raise ValueError('published candidate requires a prepared release version')
    tag = 'v' + version
    exact_source(revision, root, runner, tag)
    verify_tag(api, tag, revision)
    release = api.call('GET', f'repos/{REPOSITORY}/releases/tags/{tag}')
    if (not release or release.get('draft') is not False or release.get('tag_name') != tag
            or release.get('target_commitish') != revision):
        raise ValueError('published candidate requires the exact existing published release')
    assets = api.call('GET', f'repos/{REPOSITORY}/releases/{release["id"]}/assets?per_page=100')
    asset_count = len(selected_release_platforms(root, version)) * len(VARIANTS) + selected_installer_count(root, version) + selected_setup_count(root, version) + 4
    if (not isinstance(assets, list) or len(assets) != asset_count or len({a.get('name') for a in assets}) != asset_count
            or not {'release-manifest.json', 'SHA256SUMS', 'documentation.tar.gz', 'offline-help.tar.gz'}.issubset({a.get('name') for a in assets})):
        raise ValueError('published candidate asset set is incomplete or unexpected')
    for asset in assets:
        asset_name(asset.get('name'))
        if (asset.get('state') != 'uploaded' or type(asset.get('id')) is not int or asset['id'] <= 0
                or type(asset.get('size')) is not int or asset['size'] < 0
                or not re.fullmatch('sha256:' + SHA_PATTERN, asset.get('digest', ''))):
            raise ValueError('published candidate asset metadata is invalid')
    if output.exists():
        raise ValueError('published candidate output must be new')
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='published-candidate-', dir=output.parent) as temporary:
        staged = Path(temporary) / 'candidate'; staged.mkdir()
        for asset in assets:
            path = staged / asset['name']
            api.download_asset(asset['id'], path, asset['size'], asset['digest'].removeprefix('sha256:'))
            if file_entry(path) != {'name': asset['name'], 'size_bytes': asset['size'], 'sha256': asset['digest'].removeprefix('sha256:')}:
                raise ValueError('downloaded published candidate bytes changed')
        value, expected = load_candidate(staged, verifier, root)
        for name in ('documentation.tar.gz', 'offline-help.tar.gz'):
            verify_documentation(staged / name, value['version'], value['revision'], require_release=True)
        if value['version'] != version or value['revision'] != revision or value['tag'] != tag:
            raise ValueError('published candidate identity differs from the selected release')
        remote_assets(api, release, expected)
        verify_tag(api, tag, revision)
        if output.exists():
            raise ValueError('published candidate output must be new')
        staged.replace(output)
    return {'candidate': 'retrieved-and-verified', 'version': version, 'revision': revision}


def is_release_workflow_run(run, identifier):
    return (isinstance(run, dict) and type(run.get('id')) is int and run['id'] == identifier
            and isinstance(run.get('repository'), dict) and run['repository'].get('full_name') == REPOSITORY
            and run.get('path') in ('.github/workflows/release.yml', '.github/workflows/release.yml@main',
                                    '.github/workflows/release.yml@refs/heads/main')
            and run.get('event') == 'workflow_dispatch' and run.get('head_branch') == 'main'
            and type(run.get('workflow_id')) is int and run['workflow_id'] > 0
            and type(run.get('check_suite_id')) is int and run['check_suite_id'] > 0
            and isinstance(run.get('head_sha'), str) and re.fullmatch(r'[a-f0-9]{40}', run['head_sha']) is not None)


def current_release_run(api, context=None):
    selected = os.environ if context is None else context
    run_id = selected.get('GITHUB_RUN_ID', '')
    if not run_id:
        return None
    if not re.fullmatch(r'[1-9][0-9]*', run_id) or selected.get('GITHUB_REPOSITORY') != REPOSITORY:
        raise ValueError('current release workflow context does not identify this repository')
    run = api.call('GET', f'repos/{REPOSITORY}/actions/runs/{run_id}')
    if (not is_release_workflow_run(run, int(run_id))
            or (selected.get('GITHUB_SHA') and selected['GITHUB_SHA'] != run['head_sha'])):
        raise ValueError('current release workflow suite identity could not be verified')
    return run


def current_release_suite(api, revision, context=None):
    run = current_release_run(api, context)
    return run['check_suite_id'] if run and run['head_sha'] == revision else None


def external_source_checks(api, revision, checks, context=None):
    current = current_release_run(api, context)
    if current is None:
        return checks
    excluded = {current['check_suite_id']} if current['head_sha'] == revision else set()
    previous_runs = {}
    job_url = re.compile(r'https://github\.com/' + re.escape(REPOSITORY) + r'/actions/runs/([1-9][0-9]*)/job/[1-9][0-9]*')
    for check in checks:
        suite = check.get('check_suite', {})
        if not isinstance(suite, dict) or suite.get('id') in excluded:
            continue
        link = check.get('details_url')
        match = job_url.fullmatch(link) if isinstance(link, str) else None
        if match is None or int(match[1]) >= current['id']:
            continue
        run_id = int(match[1])
        if run_id not in previous_runs:
            previous_runs[run_id] = api.call('GET', f'repos/{REPOSITORY}/actions/runs/{run_id}')
        previous = previous_runs[run_id]
        if (is_release_workflow_run(previous, run_id) and previous['workflow_id'] == current['workflow_id']
                and previous['head_sha'] == revision and previous['check_suite_id'] == suite.get('id')
                and previous.get('status') == 'completed'
                and previous.get('conclusion') in ('action_required', 'cancelled', 'failure', 'neutral',
                                                       'skipped', 'stale', 'success', 'timed_out')):
            # A fully qualified newer release attempt supersedes earlier attempts
            # of this workflow. CI/security workflows and active/newer runs remain gates.
            excluded.add(previous['check_suite_id'])
    return [check for check in checks if check.get('check_suite', {}).get('id') not in excluded]


def source_checks(api, revision):
    runs = []; seen = set(); total = None; page = 1
    while True:
        response = api.call('GET', f'repos/{REPOSITORY}/commits/{revision}/check-runs?per_page=100&page={page}')
        count = response.get('total_count') if isinstance(response, dict) else None
        checks = response.get('check_runs') if isinstance(response, dict) else None
        if (type(count) is not int or count < 0 or not isinstance(checks, list) or len(checks) > 100
                or (total is not None and count != total)):
            raise ValueError('exact-source check inventory is invalid or changed')
        total = count
        for check in checks:
            identifier = check.get('id') if isinstance(check, dict) else None
            if type(identifier) is not int or identifier <= 0 or identifier in seen:
                raise ValueError('exact-source check inventory contains invalid or duplicate entries')
            seen.add(identifier); runs.append(check)
        if len(runs) == total:
            return runs
        if len(runs) > total or len(checks) != 100:
            raise ValueError('exact-source check inventory is incomplete')
        page += 1


def publish(directory, highlights, api, root=ROOT, runner=child, verifier=verify_package, trusted_root=None):
    trusted_history(read_json(directory / 'release-manifest.json').get('revision'), api, trusted_root or root, runner)
    value, expected = load_candidate(directory, verifier, root)
    if versions(root) != value['version'] or value['version'] == '0.0.0': raise ValueError('publication requires prepared release versions')
    for name in ('documentation.tar.gz', 'offline-help.tar.gz'):
        verify_documentation(directory / name, value['version'], value['revision'], require_release=True)
    exact_source(value['revision'], root, runner, value['tag'])
    verify_tag(api, value['tag'], value['revision'])
    runs = source_checks(api, value['revision'])
    required = read_json(root / '.github/bootstrap.json')['required_checks']
    external_runs = external_source_checks(api, value['revision'], runs)
    if not all(any(c['name'] == name and c.get('conclusion') == 'success' and c.get('status') == 'completed' for c in external_runs) for name in required) or any(c.get('status') != 'completed' or c.get('conclusion') not in ('success', 'neutral', 'skipped') for c in external_runs):
        raise ValueError('exact-source automated checks are not green')
    if not highlights.strip() or len(highlights.split()) > 180: raise ValueError('release highlights must be concise')
    suffix = f'[Full changelog](https://github.com/{REPOSITORY}/blob/{value["tag"]}/CHANGELOG.md)'
    body = highlights.strip()
    if value['version'] != '1.0.0':
        if body.endswith(suffix): body = body[:-len(suffix)].rstrip()
        body += '\n\nWindows/Linux x86-64 packages: Desktop + CLI includes application windows and terminal commands; CLI-only contains no desktop application.'
        windows = {p['variant']: p['signing'] for p in value['packages'] if p['platform'] == 'windows_amd64'}
        states = [label + ': ' + ('publisher signature verified' if windows[variant].get('configured') and windows[variant].get('verified') else 'unsigned, no verified publisher signature')
                  for variant, label in [('cli', 'CLI-only'), ('desktop', 'Desktop + CLI')]]
        body += '\nWindows signing: ' + '; '.join(states) + '.'
        if value.get('installers'): body += '\nWindows setup EXE (recommended) and raw MSI: selectable Desktop application, CLI or both; unsigned, no verified publisher signature.'
        guide = 'windows-installation' if value.get('installers') else 'desktop'
        body += f'\n[Installation, updates and removal](https://github.com/{REPOSITORY}/blob/{value["tag"]}/docs/v{value["version"]}/{guide}.md).'
        if value['version'] == '1.0.1': body += '\nmacOS support is withdrawn starting with 1.0.1; historical 1.0.0 records remain available.'
    if not body.endswith(suffix): body += '\n\n' + suffix
    release = api.call('GET', f'repos/{REPOSITORY}/releases/tags/{value["tag"]}')
    if release and not release.get('draft'):
        if release.get('tag_name') != value['tag'] or release.get('target_commitish') != value['revision']: raise ValueError('existing published release identity differs')
        remote_assets(api, release, expected)
        # Reconcile an uncertain prior final PATCH without rewriting public bytes.
        return {'published': True, 'tag': value['tag'], 'revision': value['revision'], 'release_id': release['id'], 'reconciled': True}
    if not release:
        release = api.call('POST', f'repos/{REPOSITORY}/releases', {'tag_name': value['tag'], 'target_commitish': value['revision'], 'name': 'insonic ' + value['version'], 'body': body, 'draft': True})
    if release.get('target_commitish') != value['revision'] or release.get('tag_name') != value['tag']: raise ValueError('draft release identity differs')
    current = api.call('GET', f'repos/{REPOSITORY}/releases/{release["id"]}/assets?per_page=100')
    expected_names = {entry['name'] for entry in expected}
    if any(a['name'] not in expected_names for a in current): raise ValueError('draft contains unexpected assets')
    for entry in expected:
        existing = [a for a in current if a['name'] == entry['name']]
        if existing:
            if len(existing) != 1 or existing[0].get('digest') != 'sha256:' + entry['sha256'] or existing[0].get('size') != entry['size_bytes']: raise ValueError('draft already has conflicting bytes')
            continue
        endpoint = release['upload_url'].split('{')[0] + '?name=' + urllib.parse.quote(entry['name'])
        api.call('POST', endpoint, path=directory / entry['name'])
    remote_assets(api, release, expected)
    verify_tag(api, value['tag'], value['revision'])
    published = api.call('PATCH', f'repos/{REPOSITORY}/releases/{release["id"]}', {'draft': False, 'body': body})
    if published.get('draft') is not False: raise ValueError('release publication was not confirmed')
    confirmed = api.call('GET', f'repos/{REPOSITORY}/releases/{release["id"]}')
    if confirmed.get('draft') is not False or confirmed.get('tag_name') != value['tag'] or confirmed.get('target_commitish') != value['revision']: raise ValueError('published release identity differs')
    remote_assets(api, confirmed, expected)
    return {'published': True, 'tag': value['tag'], 'revision': value['revision'], 'release_id': confirmed['id']}


def deployment_configuration(config, environment=None):
    if not isinstance(config, dict) or set(config) - {'argv', 'environment_names'}:
        raise ValueError('documentation deployment requires only argv and optional environment_names')
    argv = config.get('argv')
    if (not isinstance(argv, list) or not argv or any(not isinstance(arg, str) or not arg or '\0' in arg for arg in argv)
            or not Path(argv[0]).is_absolute()):
        raise ValueError('documentation deployment requires configured argument array and absolute executable')
    if not Path(argv[0]).is_file() or (os.name != 'nt' and not os.access(argv[0], os.X_OK)):
        raise ValueError('documentation deployment executable is unavailable on this runner')
    if '{archive}' not in '\n'.join(argv[1:]):
        raise ValueError('documentation deployment arguments must reference the verified {archive}')
    names = config.get('environment_names', [])
    if (not isinstance(names, list) or any(not isinstance(name, str) or not re.fullmatch(r'[A-Za-z_][A-Za-z0-9_]*', name) for name in names)
            or len(names) != len(set(names))):
        raise ValueError('deployment environment references must be unique variable names')
    if any(name.upper() in AUTOMATION_CREDENTIAL_NAMES or name.upper().startswith(AUTOMATION_ENVIRONMENT_PREFIXES) for name in names):
        raise ValueError('deployment cannot forward reserved automation credentials')
    selected = os.environ if environment is None else environment
    if any(not selected.get(name) for name in names):
        raise ValueError('a configured documentation deployment environment reference is unavailable')
    return {'argv': list(argv), 'environment_names': list(names)}


def materialize_deployment(path, environment=None):
    selected = os.environ if environment is None else environment
    raw = selected.get('INSONIC_DOCUMENTATION_DEPLOYMENT_JSON', '')
    if not raw.strip():
        raise ValueError('documentation promotion requires INSONIC_DOCUMENTATION_DEPLOYMENT_JSON configuration before publication')
    try:
        config = json.loads(raw)
    except (ValueError, TypeError):
        raise ValueError('documentation deployment configuration must be valid JSON') from None
    validated = deployment_configuration(config, selected)
    write_json(path, validated)
    return {'deployment_configuration': 'validated'}


def promotion_authority(directory, api, verifier=verify_package, trusted_root=ROOT, history_runner=child, source_root=ROOT):
    trusted_history(read_json(directory / 'release-manifest.json').get('revision'), api, trusted_root, history_runner)
    value, expected = load_candidate(directory, verifier, source_root)
    for name in ('documentation.tar.gz', 'offline-help.tar.gz'):
        verify_documentation(directory / name, value['version'], value['revision'], require_release=True)
    verify_tag(api, value['tag'], value['revision'])
    release = api.call('GET', f'repos/{REPOSITORY}/releases/tags/{value["tag"]}')
    if not release or release.get('draft') is not False or release.get('target_commitish') != value['revision']: raise ValueError('documentation promotion requires the exact published release')
    remote_assets(api, release, expected)
    return value


def pages_module():
    spec = importlib.util.spec_from_file_location('insonic_pages', ROOT / 'scripts/pages.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def pages_promote(directory, api, verifier=verify_package, trusted_root=ROOT, history_runner=child, source_root=ROOT):
    value = promotion_authority(directory, api, verifier, trusted_root, history_runner, source_root)
    transport = pages_module()
    destination = transport.preflight(api, REPOSITORY)
    files = transport.prepare_tree(directory / 'documentation.tar.gz', destination, value['version'], value['revision'])
    result = transport.promote_tree(api, REPOSITORY, destination, files, value['version'], value['revision'])
    result['public_verification'] = verify_public(directory, api, source_root=source_root,
        verifier=verifier, pages_commit=result['pages_commit'])
    return result


def public_release(api, value):
    verify_tag(api, value['tag'], value['revision'])
    release = api.call('GET', f'repos/{REPOSITORY}/releases/tags/{value["tag"]}')
    if (not release or release.get('draft') is not False or release.get('prerelease') is not False
            or release.get('tag_name') != value['tag'] or release.get('target_commitish') != value['revision']
            or not release.get('published_at')):
        raise ValueError('public verification requires exact published release identity')
    return release


def verify_public(directory, api, *, source_root=ROOT, verifier=verify_package,
                  site='https://insonic.io/', pages_commit=None, reader=None):
    """Verify existing publication with GETs only; never rebuild or republish."""
    value, expected = load_candidate(directory, verifier, source_root)
    release = public_release(api, value)
    remote_assets(api, release, expected)
    for name in ('documentation.tar.gz', 'offline-help.tar.gz'):
        verify_documentation(directory / name, value['version'], value['revision'], require_release=True)
    spec = importlib.util.spec_from_file_location('insonic_public_verification', ROOT / 'scripts/public-verification.py')
    public = importlib.util.module_from_spec(spec); spec.loader.exec_module(public)
    package_names = [entry['name'] for entry in expected if entry['name'].startswith('insonic_')]
    package_urls = [f'https://github.com/{REPOSITORY}/releases/download/{value["tag"]}/{name}' for name in package_names]
    options = {'package_urls': package_urls, 'api': api, 'pages_commit': pages_commit}
    if reader is not None: options['reader'] = reader
    result = public.verify(directory / 'documentation.tar.gz', value['version'], value['revision'], site, **options)
    # Fence both publication metadata and tag after observing live responses.
    if public_release(api, value)['id'] != release['id']:
        raise ValueError('public release identity changed during verification')
    remote_assets(api, release, expected)
    result.update(publication_proof='complete-api-asset-digests-and-sizes',
                  local_candidate_proof='all-original-assets-verified', public_assets=len(expected),
                  release=f'https://github.com/{REPOSITORY}/releases/tag/{value["tag"]}')
    return result


def promote(directory, config, api, runner=child, verifier=verify_package, trusted_root=ROOT, history_runner=child, source_root=ROOT):
    value = promotion_authority(directory, api, verifier, trusted_root, history_runner, source_root)
    deployment_environment(api)
    config = deployment_configuration(config)
    # The configured program receives a verified immutable archive; no shell.
    args = [a.replace('{archive}', str((directory / 'documentation.tar.gz').resolve())).replace('{version}', value['version']).replace('{revision}', value['revision']) for a in config['argv']]
    environment_names = config.get('environment_names', [])
    if runner is child:
        system_names = ('PATH', 'SystemRoot', 'WINDIR', 'TEMP', 'TMP', 'TMPDIR', 'HOME', 'USERPROFILE', 'LANG', 'LC_ALL')
        selected = {name: value for name, value in os.environ.items() if name.upper() in {n.upper() for n in (*system_names, *environment_names)}}
        runner(args, environment=selected)
    else: runner(args)
    return {'promoted': True, 'version': value['version'], 'revision': value['revision']}


def main():
    parser = argparse.ArgumentParser(description=__doc__); sub = parser.add_subparsers(dest='command', required=True)
    p = sub.add_parser('prepare'); p.add_argument('--version', required=True); p.add_argument('--date', required=True); p.add_argument('--highlights', type=Path, required=True)
    p = sub.add_parser('validate'); p.add_argument('--revision', required=True); p.add_argument('--tag')
    p = sub.add_parser('docs'); p.add_argument('--revision', required=True); p.add_argument('--output', type=Path, required=True)
    p = sub.add_parser('candidate'); p.add_argument('--revision', required=True); p.add_argument('--assets', type=Path, required=True); p.add_argument('--output', type=Path, required=True); p.add_argument('receipts', nargs='+', type=Path)
    p = sub.add_parser('collect'); p.add_argument('--revision', required=True); p.add_argument('--input', type=Path, required=True); p.add_argument('--output', type=Path, required=True); p.add_argument('--summary', action='store_true', help='Print concise completion metadata; retain the complete verified manifest in the output directory')
    p = sub.add_parser('trusted'); p.add_argument('--revision', required=True)
    p = sub.add_parser('publish'); p.add_argument('--candidate', type=Path, required=True); p.add_argument('--highlights', type=Path, required=True); p.add_argument('--source-root', type=Path, default=ROOT)
    p = sub.add_parser('published'); p.add_argument('--revision', required=True); p.add_argument('--output', type=Path, required=True); p.add_argument('--source-root', type=Path, default=ROOT)
    p = sub.add_parser('verify-public'); p.add_argument('--candidate', type=Path, required=True); p.add_argument('--source-root', type=Path, default=ROOT); p.add_argument('--site', default='https://insonic.io/'); p.add_argument('--pages-commit'); p.add_argument('--output', type=Path)
    p = sub.add_parser('deployment'); p.add_argument('--configuration', type=Path, required=True)
    sub.add_parser('pages-preflight')
    p = sub.add_parser('pages-promote'); p.add_argument('--candidate', type=Path, required=True); p.add_argument('--source-root', type=Path, default=ROOT)
    p = sub.add_parser('promote'); p.add_argument('--candidate', type=Path, required=True); p.add_argument('--configuration', type=Path, required=True); p.add_argument('--source-root', type=Path, default=ROOT)
    options = parser.parse_args()
    if options.command == 'prepare': result = prepare(options.version, options.date, options.highlights.read_text(encoding='utf-8'))
    elif options.command == 'validate': result = {'version': versions(), 'revision': options.revision}; exact_source(options.revision, tag=options.tag)
    elif options.command == 'docs': document_archives(options.output, options.revision); result = {'documentation': 'packaged'}
    elif options.command == 'candidate': result = candidate(options.receipts, options.assets, options.output, options.revision)
    elif options.command == 'collect': result = collect(options.input, options.output, options.revision, summary=options.summary)
    elif options.command == 'trusted': result = trusted_history(options.revision, GitHub(os.environ.get('GH_TOKEN', '')))
    elif options.command == 'publish': result = publish(options.candidate, options.highlights.read_text(encoding='utf-8'), GitHub(os.environ.get('GH_TOKEN', '')), root=options.source_root, trusted_root=ROOT)
    elif options.command == 'published': result = published_candidate(options.output, options.revision, GitHub(os.environ.get('GH_TOKEN', '')), root=options.source_root, trusted_root=ROOT)
    elif options.command == 'verify-public':
        result = verify_public(options.candidate, GitHub(os.environ.get('GH_TOKEN', ''), read_only=True), source_root=options.source_root, site=options.site, pages_commit=options.pages_commit)
        if options.output: write_json(options.output, result)
    elif options.command == 'deployment': deployment_environment(GitHub(os.environ.get('GH_TOKEN', ''))); result = materialize_deployment(options.configuration)
    elif options.command == 'pages-preflight':
        destination = pages_module().preflight(GitHub(os.environ.get('GH_TOKEN', '')), REPOSITORY)
        result = {key: destination[key] for key in ('repository', 'domain', 'branch', 'head')}
        result['pages_destination'] = 'verified'
    elif options.command == 'pages-promote': result = pages_promote(options.candidate, GitHub(os.environ.get('GH_TOKEN', '')), source_root=options.source_root)
    else: result = promote(options.candidate, read_json(options.configuration), GitHub(os.environ.get('GH_TOKEN', '')), source_root=options.source_root)
    print(json.dumps(result))


if __name__ == '__main__':
    try: main()
    except (ValueError, KeyError, OSError) as error: print(str(error), file=sys.stderr); sys.exit(1)
