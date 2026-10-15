// SPDX-License-Identifier: Apache-2.0
import { latestPath } from '../lib/docs.mjs';

export default function NotFound() {
  return <main id="main" className="landing"><h1>Page not found</h1><p>This documentation page is unavailable.</p><p><a href={latestPath()}>Open current documentation</a></p><p><a href="/docs/versions/">Browse documentation versions</a></p></main>;
}
