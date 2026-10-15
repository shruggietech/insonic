// SPDX-License-Identifier: Apache-2.0
import { getManifest } from '../lib/docs.mjs';
import { getBrandMessaging } from '../lib/brand.mjs';
import LandingPage from './components/LandingPage.jsx';

export default function Home() {
  const manifest = getManifest();
  const latest = manifest.versions.find(entry => entry.version === manifest.latest);
  return <LandingPage latest={latest} brand={getBrandMessaging()} />;
}
