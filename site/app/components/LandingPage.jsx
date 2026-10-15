// SPDX-License-Identifier: Apache-2.0
import { latestPath } from '../../lib/docs-routes.mjs';
import ArchivePreview from './ArchivePreview.jsx';
import { releaseDownloads } from '../../lib/release-status.mjs';

export default function LandingPage({ latest, brand }) {
  const released = latest.status === 'released';
  return (
    <main id="main" className="landing product-landing" data-public-landing="true" tabIndex={-1}>
      <section className="product-hero" aria-labelledby="product-heading">
        <div>
          <p className="eyebrow">insonic · v{latest.version}</p>
          <h1 id="product-heading">{brand.slogan}</h1>
          <p className="lead">{brand.description}</p>
          <p className="hero-detail">Organize audio and video, preserve source metadata, and connect speakers to what they said. Configure local or hosted processing through a CLI-first core and desktop client.</p>
          <div className="product-actions">
            <a className="button" href="#downloads">Choose a download <span aria-hidden="true">↓</span></a>
            <a className="button button-secondary" href={latestPath()}>Read the documentation <span aria-hidden="true">→</span></a>
          </div>
          <p className="product-stage">v{latest.version} {released ? 'released. Official packages are available below.' : latest.status === 'candidate' ? 'prepared release candidate. Official product downloads have not been published for this version.' : 'system specification. Official product downloads have not been published for this version.'}</p>
          <p className="product-platforms">Windows · Linux <span>CLI + desktop</span></p>
        </div>
        <ArchivePreview />
      </section>
      <section id="capabilities" className="product-section" aria-labelledby="capability-heading">
        <p className="eyebrow">From original media to connected knowledge</p>
        <h2 id="capability-heading">One library. A precise source for every result.</h2>
        <div className="capability-grid">
          <article><span className="feature-number">01 / Library</span><h3>Preserve the audio and its source.</h3><p>Capture source metadata early. Retain canonical audio, independent tracks, source timing and one current Cue JSON transcript.</p><a href={latestPath('ingestion')}>Media and metadata</a></article>
          <article><span className="feature-number">02 / Processing</span><h3>Choose your pipeline.</h3><p>Route transcription, diarization and reasoning through local models or configured providers. Cueson gives subtitles one consistent contract.</p><a href={latestPath('pipelines')}>Pipelines and adapters</a></article>
          <article><span className="feature-number">03 / Speakers</span><h3>Follow a voice.</h3><p>Connect speakers, aliases and tagged audio across the library. Elect training from a frozen corpus and retrieve model versions by speaker.</p><a href={latestPath('voice-models')}>Speaker audio and models</a></article>
          <article><span className="feature-number">04 / Explore</span><h3>Find what was said and when.</h3><p>Search words, people and time. Follow results back to source media, navigate timelines and save graph views with optional AI query assistance.</p><a href={latestPath('graph')}>Queries and exploration</a></article>
        </div>
      </section>
      <section className="product-section product-foundation" aria-labelledby="foundation-heading">
        <div><p className="eyebrow">A system you can configure</p><h2 id="foundation-heading">Open contracts.<br />Your infrastructure.</h2><p>Documented JSON schemas connect the CLI, adapters and application. The desktop client wraps the shared core, and versioned help travels with it.</p><a href={latestPath('contracts')}>Explore the JSON contracts</a></div>
        <dl className="backend-list"><div><dt>Artifact storage</dt><dd>Filesystem / S3-compatible</dd></div><div><dt>Operational catalog</dt><dd>SQLite / PostgreSQL</dd></div><div><dt>Graph queries</dt><dd>LadybugDB / ArcadeDB</dd></div><div><dt>Processing</dt><dd>Local / connected providers</dd></div></dl>
      </section>
      <section className="product-section product-cta" aria-labelledby="start-heading">
        <p className="eyebrow">Start with the system</p><h2 id="start-heading">See how the pieces connect.</h2><p>Read the architecture, import rules and processing contracts.</p><a className="text-action" href={latestPath('architecture')}>Read the architecture</a><a className="text-action" href="/docs/versions/">All documentation versions</a>
      </section>
      <section id="downloads" className="product-section download-section" aria-labelledby="download-heading">
        <p className="eyebrow">CLI + desktop</p><h2 id="download-heading">Download insonic.</h2>
        <p>{released ? `Download v${latest.version} for Windows or Linux.` : `v${latest.version} ${latest.status === 'candidate' ? 'is a prepared release candidate' : 'is the system specification baseline'}; official product downloads have not been published for this version.`}</p>
        <p>Desktop + CLI includes application windows and terminal commands. CLI only provides terminal commands.</p>
        <div className="download-grid">
          {releaseDownloads(latest.version).map(platform => (
            <article key={platform.label} aria-labelledby={`download-${platform.platform}`}>
              <h3 id={`download-${platform.platform}`}>{platform.label}</h3>
              <p>{platform.label} x86-64 (Intel or AMD). {platform.label === 'Windows'
                ? 'Both clients require Microsoft Visual C++ x64. Desktop also requires WebView2.'
                : 'Both packages require Perl. Desktop also requires GTK 3, WebKitGTK 4.1 and GStreamer decoders.'}</p>
              {platform.installer && <section className="download-group" aria-labelledby="windows-installer-heading">
                <h4 id="windows-installer-heading">Installer (MSI)</h4>
                <p>Choose Desktop, CLI or both. Both are selected by default. Use Installed Apps to change or remove components.</p>
                <div className="download-actions">
                  {released && <a className="button button-secondary" href={platform.installer}>Download Windows MSI</a>}
                  <a className="text-action" href={latestPath('windows-installation')}>Windows installation, updates and removal</a>
                </div>
              </section>}
              <section className="download-group" aria-labelledby={`portable-${platform.platform}`}>
                <h4 id={`portable-${platform.platform}`}>Portable {platform.label === 'Windows' ? 'ZIP' : 'tar.gz'} packages</h4>
                <p>Extract an archive to use a portable copy. Choose the application with its CLI or terminal commands only.</p>
                <div className="download-actions">
                  {released ? <>
                    <a className={platform.installer ? 'text-action' : 'button button-secondary'} href={platform.desktop}>Download {platform.label} Desktop + CLI {platform.label === 'Windows' ? 'ZIP' : 'tar.gz'}</a>
                    <a className="text-action" href={platform.cli}>Download {platform.label} CLI-only {platform.label === 'Windows' ? 'ZIP' : 'tar.gz'}</a>
                  </> : <p className="download-pending">{platform.label} download pending</p>}
                  <a className="text-action" href={latestPath('desktop') + (platform.label === 'Windows' ? '#install-a-windows-zip' : '#linux-prerequisites-and-installation')}>
                    {platform.label === 'Windows' ? 'Windows ZIP installation and maintenance' : 'Linux prerequisites and installation'}
                  </a>
                </div>
              </section>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
