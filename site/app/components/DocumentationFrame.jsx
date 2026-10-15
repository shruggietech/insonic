// SPDX-License-Identifier: Apache-2.0
import { getManifest, latestPath, versionPath } from '../../lib/docs.mjs';

export default function DocumentationFrame({ entry, slug, headings = [], previous, next, children, reference = false }) {
  const current = getManifest().latest === entry.version;
  return (
    <div className="docs-layout">
      <aside className="docs-sidebar">
        <details className="navigation-disclosure" data-responsive-disclosure="960">
          <summary>Documentation</summary>
          <nav aria-label={`Documentation topics v${entry.version}`}>
            <p className="version-label">v{entry.version} · {entry.status}</p>
            <ul>{entry.pages.map(item => <li key={item.slug}><a href={versionPath(entry.version, item.slug)} aria-current={!reference && item.slug === slug ? 'page' : undefined}>{item.title}</a></li>)}</ul>
          </nav>
        </details>
      </aside>
      {headings.length > 0 && <aside className="table-of-contents">
        <details className="navigation-disclosure" data-responsive-disclosure="1200">
          <summary>On this page</summary>
          <nav aria-label="On this page">
            <ul>{headings.map(heading => <li key={heading.id}><a href={`#${heading.id}`}>{heading.title}</a></li>)}</ul>
          </nav>
        </details>
      </aside>}
      <main id="main" className="docs-main" tabIndex={-1}>
        <div className="docs-version-context">
          <p>{reference ? 'Reference' : 'insonic'} · v{entry.version} · {entry.status}{current ? ' · Current documentation' : ''}</p>
          <a href="/docs/versions/">All versions</a>
          {!current && <a href={latestPath()}>Current documentation</a>}
        </div>
        {children}
        {(previous || next) && <nav className="page-navigation" aria-label="Previous and next page">
          {previous ? <a href={versionPath(entry.version, previous.slug)} rel="prev"><span>Previous</span>{previous.title}</a> : <span />}
          {next && <a href={versionPath(entry.version, next.slug)} rel="next"><span>Next</span>{next.title}</a>}
        </nav>}
      </main>
    </div>
  );
}
