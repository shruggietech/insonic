// SPDX-License-Identifier: Apache-2.0
import { notFound } from 'next/navigation';
import DocumentationPage from '../docs/[version]/[slug]/page';
import ReferencePage from '../docs/[version]/references/[reference]/page';
import { getManifest, getLatestEntry, getRepositoryReferences, latestTopicSlugs, latestReferenceSlugs, latestPath, versionPath, referencePath } from '../../lib/docs.mjs';

function missingTopic(entry, topic) {
  return (
    <main id="main" className="landing">
      <p className="eyebrow">Current documentation · v{entry.version} · {entry.status}</p>
      <h1>This topic is unavailable in the current documentation</h1>
      <p>The archived topic <strong>{topic}</strong> is not included in v{entry.version}. No historical content has been substituted for it.</p>
      <p><a href={latestPath()}>Open the current documentation</a></p>
      <p><a href="/docs/versions/">Choose an archived documentation version</a></p>
    </main>
  );
}

export function latestDocumentMetadata(slug = 'index') {
  const manifest = getManifest();
  const entry = getLatestEntry(manifest);
  const page = entry.pages.find(item => item.slug === slug);
  return {
    title: page ? page.title + ' (current v' + entry.version + ')' : 'Topic unavailable in current documentation',
    alternates: { canonical: versionPath(entry.version, page ? slug : entry.pages[0].slug) },
  };
}

export async function LatestDocument({ slug = 'index' }) {
  const manifest = getManifest();
  const entry = getLatestEntry(manifest);
  if (!latestTopicSlugs(manifest).includes(slug)) notFound();
  if (!entry.pages.some(item => item.slug === slug)) return missingTopic(entry, slug);
  // Render the complete article so fragments and reading work without JavaScript,
  // on static hosts and when opening the exported index.html from the filesystem.
  return DocumentationPage({ params: Promise.resolve({ version: 'v' + entry.version, slug }) });
}

export function latestReferenceMetadata(reference) {
  const entry = getLatestEntry(getManifest());
  const item = getRepositoryReferences().find(candidate => candidate.entry.version === entry.version && candidate.slug === reference);
  return {
    title: item ? item.repositoryPath + ' (current v' + entry.version + ')' : 'Reference unavailable in current documentation',
    alternates: { canonical: item ? referencePath(entry.version, reference) : versionPath(entry.version, entry.pages[0].slug) },
  };
}

export async function LatestReference({ reference }) {
  const entry = getLatestEntry(getManifest());
  const references = getRepositoryReferences();
  if (!latestReferenceSlugs(references).includes(reference)) notFound();
  if (!references.some(item => item.entry.version === entry.version && item.slug === reference)) return missingTopic(entry, 'references/' + reference);
  return ReferencePage({ params: Promise.resolve({ version: 'v' + entry.version, reference }) });
}
