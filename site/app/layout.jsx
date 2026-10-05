// SPDX-License-Identifier: Apache-2.0
import './styles.css';
import { getBrandMessaging } from '../lib/brand.mjs';

const brand = getBrandMessaging();

export const metadata = {
  title: { default: `${brand.title} | ${brand.slogan}`, template: '%s | insonic' },
  description: brand.description,
};

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <head>
        <meta name="color-scheme" content="light dark" />
        <script src="/theme-init.js" />
        <link rel="stylesheet" href="/brand/theme.css" />
        <link rel="stylesheet" href="/brand/fonts/fonts.css" />
        <link rel="icon" href="/brand/icons/web/favicon.ico" />
        <link rel="apple-touch-icon" href="/brand/icons/web/favicon-180x180.png" />
      </head>
      <body>
        <a className="skip-link" href="#main">Skip to content</a>
        <header className="site-header">
          <a className="brand-link" href="/" aria-label="insonic home">
            <img className="theme-logo-light" src="/brand/logos/named/insonic-logo-lockup-wide-full-color-clear-for-light.svg" alt="insonic" width="240" height="64" />
            <img className="theme-logo-dark" src="/brand/logos/named/insonic-logo-lockup-wide-full-color-clear-for-dark.svg" alt="insonic" width="240" height="64" />
          </a>
          <nav aria-label="Main navigation">
            <a href="/docs/">Documentation</a>
            <a href="https://github.com/shruggietech/insonic">Repository</a>
            <button id="theme-toggle" className="theme-toggle" type="button" role="switch" aria-label="Dark mode" aria-checked="false" title="Switch to dark theme">
              <svg className="theme-icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M20.9 13.1A9 9 0 0 1 10.9 3.1 9 9 0 1 0 20.9 13.1Z" /></svg>
              <svg className="theme-icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true"><circle cx="12" cy="12" r="4" /><path d="M12 2v2 M12 20v2 M2 12h2 M20 12h2 M4.9 4.9l1.4 1.4 M17.7 17.7l1.4 1.4 M4.9 19.1l1.4-1.4 M17.7 6.3l1.4-1.4" /></svg>
            </button>
          </nav>
        </header>
        {children}
        <footer className="site-footer">
          <p>A <a href="https://shruggie.tech/">ShruggieTech</a> project.</p>
          <p>Apache 2.0. Documentation versions are published with their corresponding releases.</p>
        </footer>
        <script src="/site-runtime.js" defer />
      </body>
    </html>
  );
}
