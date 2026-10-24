# SPDX-License-Identifier: Apache-2.0
"""Render exported local diagram sources in a bounded hidden browser."""
import html
from html.parser import HTMLParser
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

from process_tree import ProcessTree
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import qualification_timing as timing

ROOT = Path(__file__).resolve().parents[1]


class Diagrams(HTMLParser):
    def __init__(self):
        super().__init__()
        self.current = None
        self.sources = []

    def handle_starttag(self, tag, attributes):
        if tag == 'div' and 'mermaid' in dict(attributes).get('class', '').split():
            self.current = []

    def handle_data(self, data):
        if self.current is not None:
            self.current.append(data)

    def handle_endtag(self, tag):
        if tag == 'div' and self.current is not None:
            self.sources.append(''.join(self.current))
            self.current = None


def browser_executable():
    configured = os.environ.get('INSONIC_DOCS_BROWSER')
    candidates = [configured] if configured else [
        shutil.which(name) for name in ('google-chrome', 'google-chrome-stable', 'chromium', 'chromium-browser')]
    if os.name == 'nt' and not configured:
        drive = os.environ.get('SystemDrive', 'C:')
        bases = [os.environ.get(variable) for variable in ('ProgramFiles', 'ProgramFiles(x86)', 'LOCALAPPDATA')]
        bases.extend([drive + '/Program Files', drive + '/Program Files (x86)'])
        for base in bases:
            if base:
                candidates.extend(str(Path(base) / name) for name in (
                    'Google/Chrome/Application/chrome.exe', 'Microsoft/Edge/Application/msedge.exe'))
    for candidate in candidates:
        if candidate and Path(candidate).is_absolute() and Path(candidate).is_file():
            return candidate
    raise ValueError('Chrome, Chromium or Edge is required for documentation rendering qualification')


PROBE = r"""
const result = document.createElement('output');
result.id = 'insonic-renderer-result'; document.body.append(result);
const deadline = performance.now() + __RENDER_TIMEOUT_MS__;
function waitFor(test) {
  return new Promise((resolve, reject) => {
    const check = () => {
      if (document.querySelector('.diagram-error')) return reject(Error('diagram render error'));
      if (test()) return resolve();
      if (performance.now() > deadline) return reject(Error('diagram render timeout'));
      setTimeout(check, 20);
    }; check();
  });
}
(async () => {
  const diagrams = [...document.querySelectorAll('.mermaid')];
  await waitFor(() => diagrams.every(node => node.dataset.processed === 'true' && node.querySelector('svg')));
  const old = diagrams.map(node => node.innerHTML);
  const rect = diagrams[0].querySelector('.node rect');
  if (!rect) throw Error('expected a rendered flowchart node');
  const light = getComputedStyle(rect).fill;
  document.documentElement.dataset.theme = 'dark';
  window.dispatchEvent(new Event('insonic-themechange'));
  await waitFor(() => diagrams.every((node, index) => node.innerHTML !== old[index] && node.querySelector('svg')));
  const dark = getComputedStyle(diagrams[0].querySelector('.node rect')).fill;
  if (light === dark) throw Error('diagram colors did not follow theme tokens');
  result.textContent = JSON.stringify({status:'passed', diagrams:diagrams.length, light, dark});
})().catch(error => { result.textContent = JSON.stringify({status:'failed', error:String(error)}); });
"""


def qualify_export(export, browser):
    sources = []
    for page in sorted(export.rglob('*.html')):
        parser = Diagrams()
        parser.feed(page.read_text(encoding='utf-8'))
        for source in parser.sources:
            if source not in sources:
                sources.append(source)
    if not sources:
        raise ValueError('export has no documentation diagram sources')
    runtime = export / 'docs-runtime.js'
    styles = sorted((export / '_next').rglob('*.css')) + [export / 'brand/theme.css']
    if not runtime.is_file() or not styles or any(not path.is_file() for path in styles):
        raise ValueError('exported diagram runtime or styles are missing')
    with tempfile.TemporaryDirectory(prefix='insonic renderer ') as temporary:
        location = Path(temporary)
        fixture = location / 'diagrams.html'
        content = '<!doctype html><html data-theme="light"><head><meta charset="utf-8">'
        content += ''.join('<link rel="stylesheet" href="' + path.as_uri() + '">' for path in styles)
        content += '</head><body>' + ''.join('<figure class="diagram"><div class="mermaid">' + html.escape(source) + '</div></figure>' for source in sources)
        content += '<script src="' + runtime.as_uri() + '"></script><script>' + PROBE.replace('__RENDER_TIMEOUT_MS__', str(timing.DOCS_RENDERER_SECONDS * 1000)) + '</script></body></html>'
        fixture.write_text(content, encoding='utf-8', newline='\n')
        args = [browser, '--headless=new', '--disable-gpu', '--disable-background-networking',
                '--disable-extensions', '--no-first-run', '--no-default-browser-check',
                '--allow-file-access-from-files', '--host-resolver-rules=MAP * ~NOTFOUND',
                '--user-data-dir=' + str(location / 'profile'), '--virtual-time-budget=' + str(timing.DOCS_RENDERER_SECONDS * 1000),
                '--dump-dom', fixture.as_uri()]
        with ProcessTree(args, cwd=ROOT, env=os.environ.copy()) as tree:
            try:
                stdout, stderr = tree.process.communicate(timeout=timing.DOCS_BROWSER_SECONDS)
            except subprocess.TimeoutExpired:
                tree.kill()
                tree.process.communicate(timeout=timing.CLEANUP_SECONDS)
                raise ValueError('bounded documentation browser timed out') from None
            if tree.process.returncode:
                raise ValueError('headless documentation browser failed: ' + stderr.decode('utf-8', errors='replace')[-2000:])
        # Parse the actual output element; never accept a marker embedded in the probe source.
        class Result(HTMLParser):
            def __init__(self):
                super().__init__(); self.active = False; self.text = ''
            def handle_starttag(self, tag, attributes):
                self.active = tag == 'output' and dict(attributes).get('id') == 'insonic-renderer-result'
            def handle_data(self, data):
                if self.active: self.text += data
            def handle_endtag(self, tag):
                if tag == 'output': self.active = False
        parser = Result(); parser.feed(stdout.decode('utf-8'))
        result = json.loads(parser.text or '{}')
        if result.get('status') != 'passed' or result.get('diagrams') != len(sources):
            raise ValueError('exported diagram/theme qualification failed: ' + str(result))
        print(json.dumps({'export': export.name, **result}, sort_keys=True))


if __name__ == '__main__':
    browser = browser_executable()
    for name in ('out', 'offline'):
        qualify_export(ROOT / 'site' / name, browser)
