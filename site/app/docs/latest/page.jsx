// SPDX-License-Identifier: Apache-2.0
import { LatestDocument, latestDocumentMetadata } from '../../components/LatestDocumentation.jsx';

export function generateMetadata() { return latestDocumentMetadata(); }

export default function LatestIndex() { return LatestDocument({}); }
