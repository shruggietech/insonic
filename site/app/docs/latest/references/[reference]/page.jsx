// SPDX-License-Identifier: Apache-2.0
import { getRepositoryReferences, latestReferenceSlugs } from '../../../../../lib/docs.mjs';
import { LatestReference, latestReferenceMetadata } from '../../../../components/LatestDocumentation.jsx';

export const dynamicParams = false;
export function generateStaticParams() { return latestReferenceSlugs(getRepositoryReferences()).map(reference => ({ reference })); }
export async function generateMetadata({ params }) { return latestReferenceMetadata((await params).reference); }
export default async function LatestReferencePage({ params }) { return LatestReference({ reference: (await params).reference }); }
