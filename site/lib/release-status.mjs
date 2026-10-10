// SPDX-License-Identifier: Apache-2.0
export function documentationTarget(environment = process.env) {
  const target = environment.INSONIC_DOCUMENTATION_TARGET || 'snapshot';
  if (!['snapshot', 'release'].includes(target)) throw new Error('Invalid documentation publication target.');
  return target;
}

export function publicationManifest(manifest, target) {
  const value = structuredClone(manifest);
  if (target === 'release') {
    const entry = value.versions.find(item => item.version === value.latest);
    if (!entry || entry.status === 'specification') throw new Error('Release documentation requires a prepared product version.');
    entry.status = 'released';
  }
  return value;
}

export function publicationDocument(source, entry, slug, target) {
  if (target !== 'release' || entry.status !== 'released' || slug !== 'index') return source;
  const escaped = entry.version.replaceAll('.', '\\.');
  return source.replace(new RegExp('^v' + escaped + ' documents the prepared release candidate\\.[^\\n]*', 'm'),
    `v${entry.version} documents this release. Package availability and qualification are recorded in its release manifest.`);
}

export function releaseDownloads(version) {
  const base = `https://github.com/shruggietech/insonic/releases/download/v${version}/insonic_${version}_`;
  return [{ label: 'macOS', platform: 'darwin_arm64', extension: '.zip' },
    { label: 'Windows', platform: 'windows_amd64', extension: '.zip' },
    { label: 'Linux', platform: 'linux_amd64', extension: '.tar.gz' }].map(item => ({ ...item,
      desktop: base + item.platform + '_desktop' + item.extension,
      cli: base + item.platform + '_cli' + item.extension }));
}
