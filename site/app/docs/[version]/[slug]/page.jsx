// SPDX-License-Identifier: Apache-2.0
import { notFound } from 'next/navigation';
import { getManifest, readDocument, renderDocument } from '../../../../lib/docs.mjs';
import DocumentationFrame from '../../../components/DocumentationFrame.jsx';

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
    <DocumentationFrame entry={entry} slug={slug} headings={headings} previous={previous} next={next}>
      <article className="prose" dangerouslySetInnerHTML={{ __html: html }} />
    </DocumentationFrame>
  );
}
