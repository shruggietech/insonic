// SPDX-License-Identifier: Apache-2.0
import { getManifest, getLatestEntry, latestPath, versionPath } from '../../../lib/docs.mjs';

export const metadata = { title: 'Documentation versions' };

export default function DocumentationVersions() {
  const manifest = getManifest();
  const current = getLatestEntry(manifest);
  const versions = [current, ...manifest.versions.filter(entry => entry.version !== current.version).reverse()];
  return (
    <main id="main" className="landing">
      <p className="eyebrow">Documentation archive</p>
      <h1>Documentation versions</h1>
      <p className="lead">Choose the documentation matching your installation. Explicit version links retain their version; current documentation follows the manifest's latest version.</p>
      <p><a className="button" href={latestPath()}>Read current documentation · v{current.version} · {current.status}</a></p>
      <ul className="version-list">
        {versions.map(entry => (
          <li key={entry.version}>
            <h2><a href={versionPath(entry.version, entry.pages[0].slug)}>v{entry.version}</a></h2>
            <p>{entry.status === 'specification' ? 'System specification baseline.' : entry.status === 'candidate' ? 'Prepared release candidate; official product downloads have not been published for this version.' : 'Released documentation.'} {entry.version === manifest.latest ? 'Current documentation.' : 'Archived documentation.'}</p>
          </li>
        ))}
      </ul>
    </main>
  );
}
