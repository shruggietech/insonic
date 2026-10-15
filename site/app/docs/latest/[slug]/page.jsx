// SPDX-License-Identifier: Apache-2.0
import { getManifest, latestTopicSlugs } from '../../../../lib/docs.mjs';
import { LatestDocument, latestDocumentMetadata } from '../../../components/LatestDocumentation.jsx';

export const dynamicParams = false;
export function generateStaticParams() { return latestTopicSlugs(getManifest()).map(slug => ({ slug })); }
export async function generateMetadata({ params }) { return latestDocumentMetadata((await params).slug); }
export default async function LatestTopic({ params }) { return LatestDocument({ slug: (await params).slug }); }
