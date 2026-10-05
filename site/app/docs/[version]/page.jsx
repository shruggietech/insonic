// SPDX-License-Identifier: Apache-2.0
import { notFound } from 'next/navigation';
import DocumentationPage from './[slug]/page';
import { getManifest } from '../../../lib/docs.mjs';

export const dynamicParams = false;
export function generateStaticParams() {
  return getManifest().versions.map((entry) => ({ version: `v${entry.version}` }));
}
export async function generateMetadata({ params }) {
  const { version } = await params;
  return { title: `Documentation ${version}` };
}
export default async function VersionIndex({ params }) {
  const { version } = await params;
  const entry = getManifest().versions.find((candidate) => `v${candidate.version}` === version);
  if (!entry) notFound();
  return DocumentationPage({ params: Promise.resolve({ version, slug: entry.pages[0].slug }) });
}
