// SPDX-License-Identifier: Apache-2.0
import { getManifest, versionPath } from '../../lib/docs.mjs';

export const metadata = { title: 'Documentation versions' };

export default function DocumentationIndex() {
  const manifest = getManifest();
  return (
    <main id="main" className="landing">
      <p className="eyebrow">Documentation</p>
      <h1>Versioned by release</h1>
      <p className="lead">Choose the documentation that matches your installed version. The current specification establishes the design before application releases begin.</p>
      <ul className="version-list">
        {manifest.versions.map((entry) => (
          <li key={entry.version}>
            <h2><a href={versionPath(entry.version, entry.pages[0].slug)}>v{entry.version}</a></h2>
            <p>{entry.status === 'specification' ? 'System specification; application not yet released.' : 'Release documentation.'} {entry.version === manifest.latest ? 'Latest documentation.' : 'Archived version.'}</p>
          </li>
        ))}
      </ul>
    </main>
  );
}
