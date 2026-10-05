// SPDX-License-Identifier: Apache-2.0
import { notFound } from 'next/navigation';
import { getManifest, readDocument, renderDocument, versionPath } from '../../../../lib/docs.mjs';

export const dynamicParams = false;

export function generateStaticParams() {
  return getManifest().versions.flatMap((entry) => entry.pages.map((page) => ({ version: `v${entry.version}`, slug: page.slug })));
}

export async function generateMetadata({ params }) {
  const { version, slug } = await params;
  const document = readDocument(version, slug);
  return { title: document ? `${document.page.title} (v${document.entry.version})` : 'Page not found' };
}

export default async function DocumentationPage({ params }) {
  const { version, slug } = await params;
  const document = readDocument(version, slug);
  if (!document) notFound();
  const { entry, page, source } = document;
  const { html, headings } = renderDocument(source, entry);
  const index = entry.pages.findIndex((item) => item.slug === slug);
  const previous = entry.pages[index - 1];
  const next = entry.pages[index + 1];
  return (
    <div className="docs-layout">
      <aside className="docs-sidebar">
        <nav aria-label={`Documentation v${entry.version}`}>
          <a className="version-label" href="/docs/">v{entry.version} · {entry.status}</a>
          <ul>{entry.pages.map((item) => <li key={item.slug}><a href={versionPath(entry.version, item.slug)} aria-current={item.slug === slug ? 'page' : undefined}>{item.title}</a></li>)}</ul>
        </nav>
      </aside>
      <main id="main" className="docs-main">
        <p className="eyebrow">insonic · v{entry.version}</p>
        <article className="prose" dangerouslySetInnerHTML={{ __html: html }} />
        <nav className="page-navigation" aria-label="Previous and next page">
          {previous ? <a href={versionPath(entry.version, previous.slug)}>← {previous.title}</a> : <span />}
          {next && <a href={versionPath(entry.version, next.slug)}>{next.title} →</a>}
        </nav>
      </main>
      <aside className="table-of-contents">
        <nav aria-label="On this page">
          <h2>On this page</h2>
          <ul>{headings.map((heading) => <li key={heading.id}><a href={`#${heading.id}`}>{heading.title}</a></li>)}</ul>
        </nav>
      </aside>
    </div>
  );
}
