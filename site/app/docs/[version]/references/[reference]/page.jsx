// SPDX-License-Identifier: Apache-2.0
import { notFound } from 'next/navigation';
import { getRepositoryReferences, renderDocument, versionPath } from '../../../../../lib/docs.mjs';

export const dynamicParams = false;
export function generateStaticParams() {
  return getRepositoryReferences().map(({ entry, slug }) => ({ version: `v${entry.version}`, reference: slug }));
}
export async function generateMetadata({ params }) {
  const { version, reference } = await params;
  const item = getRepositoryReferences().find((candidate) => `v${candidate.entry.version}` === version && candidate.slug === reference);
  return { title: item ? `${item.repositoryPath} (v${item.entry.version})` : 'Reference not found' };
}
export default async function ReferencePage({ params }) {
  const { version, reference } = await params;
  const item = getRepositoryReferences().find((candidate) => `v${candidate.entry.version}` === version && candidate.slug === reference);
  if (!item) notFound();
  const { entry, repositoryPath, source, slug } = item;
  const markdown = repositoryPath.endsWith('.md') ? source : `# ${repositoryPath}\n\n\`\`\`${repositoryPath.split('.').at(-1)}\n${source}\n\`\`\`\n`;
  const { html } = renderDocument(markdown, entry, repositoryPath);
  const gitReference = entry.status === 'specification' ? 'main' : `v${entry.version}`;
  return (
    <main id="main" className="landing">
      <p className="eyebrow">Repository reference · v{entry.version}</p>
      <p><a href={versionPath(entry.version, entry.pages[0].slug)}>← Return to documentation</a></p>
      <article className="prose" dangerouslySetInnerHTML={{ __html: html }} />
      <p><a href={`/reference-sources/v${entry.version}/${slug}.txt`}>Read the bundled source</a> · <a href={`https://github.com/shruggietech/insonic/blob/${gitReference}/${repositoryPath}`}>View upstream repository source</a></p>
    </main>
  );
}
