// SPDX-License-Identifier: Apache-2.0
import { getManifest, versionPath } from '../lib/docs.mjs';
import { getBrandMessaging } from '../lib/brand.mjs';
import ArchivePreview from './components/ArchivePreview.jsx';

export default function Home() {
  const manifest = getManifest();
  const brand = getBrandMessaging();
  const latest = manifest.versions.find((entry) => entry.version === manifest.latest);
  return (
    <main id="main" className="landing product-landing" data-public-landing="true">
      <section className="product-hero" aria-labelledby="product-heading">
        <div>
          <p className="eyebrow">insonic · v{latest.version}</p>
          <h1 id="product-heading">{brand.slogan}</h1>
          <p className="lead">{brand.description}</p>
          <p className="hero-detail">Organize audio and video, preserve source metadata, and connect speakers to what they said. Configure local or hosted processing through a CLI-first core and desktop client.</p>
          <div className="product-actions">
            <a className="button" href="#downloads">Download insonic <span aria-hidden="true">↓</span></a>
            <a className="button button-secondary" href={versionPath(latest.version, latest.pages[0].slug)}>Read the docs <span aria-hidden="true">→</span></a>
          </div>
          <p className="product-stage">v{latest.version} system specification. Application downloads are not available yet.</p>
          <p className="product-platforms">Windows · macOS · Linux <span>CLI + desktop</span></p>
        </div>
        <ArchivePreview />
      </section>
      <section id="capabilities" className="product-section" aria-labelledby="capability-heading">
        <p className="eyebrow">From original media to connected knowledge</p>
        <h2 id="capability-heading">One library. A precise source for every result.</h2>
        <div className="capability-grid">
          <article><span className="feature-number">01 / Library</span><h3>Keep the original.</h3><p>Capture source metadata early. Track audio, video, supplied subtitles and optimized derivatives without losing their time or identity.</p><a href={versionPath(latest.version, 'ingestion')}>Media and metadata</a></article>
          <article><span className="feature-number">02 / Processing</span><h3>Choose your pipeline.</h3><p>Route transcription, diarization and reasoning through local models or configured providers. Cueson gives subtitles one consistent contract.</p><a href={versionPath(latest.version, 'pipelines')}>Pipelines and adapters</a></article>
          <article><span className="feature-number">03 / Speakers</span><h3>Follow a voice.</h3><p>Connect speakers, aliases and tagged audio across the library. Elect training from a frozen corpus and retrieve model versions by speaker.</p><a href={versionPath(latest.version, 'voice-models')}>Speaker audio and models</a></article>
          <article><span className="feature-number">04 / Explore</span><h3>Find what was said and when.</h3><p>Search words, people and time. Follow results back to source media, navigate timelines and save graph views with optional AI query assistance.</p><a href={versionPath(latest.version, 'graph')}>Queries and exploration</a></article>
        </div>
      </section>
      <section className="product-section product-foundation" aria-labelledby="foundation-heading">
        <div><p className="eyebrow">A system you can configure</p><h2 id="foundation-heading">Open contracts.<br />Your infrastructure.</h2><p>Documented JSON schemas connect the CLI, adapters and application. The desktop client wraps the shared core, and versioned help travels with it.</p><a href={versionPath(latest.version, 'contracts')}>Explore the JSON contracts</a></div>
        <dl className="backend-list"><div><dt>Artifact storage</dt><dd>Filesystem / S3-compatible</dd></div><div><dt>Operational catalog</dt><dd>SQLite / PostgreSQL</dd></div><div><dt>Graph queries</dt><dd>LadybugDB / ArcadeDB</dd></div><div><dt>Processing</dt><dd>Local / connected providers</dd></div></dl>
      </section>
      <section className="product-section product-cta" aria-labelledby="start-heading">
        <p className="eyebrow">Start with the system</p><h2 id="start-heading">See how the pieces connect.</h2><p>Read the architecture, import rules and processing contracts.</p><a className="button" href={versionPath(latest.version, 'architecture')}>Read the architecture</a><a className="text-action" href="/docs/">All documentation versions</a>
      </section>
      <section id="downloads" className="product-section download-section" aria-labelledby="download-heading">
        <p className="eyebrow">CLI + desktop</p><h2 id="download-heading">Download insonic.</h2>
        <p>Release packages for macOS, Windows and Linux appear here with each release. The current v{latest.version} baseline is the system specification; installable packages are not available yet.</p>
        <div className="download-grid">
          {['macOS', 'Windows', 'Linux'].map(platform => <article key={platform}><h3>{platform}</h3><p>Desktop + CLI</p><button type="button" disabled>{platform} download pending</button></article>)}
        </div>
        <a className="text-action" href={versionPath(latest.version, 'desktop')}>Installation and packaging requirements <span aria-hidden="true">→</span></a>
      </section>
    </main>
  );
}
