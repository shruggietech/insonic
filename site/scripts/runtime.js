// SPDX-License-Identifier: Apache-2.0
import mermaid from 'mermaid';

const diagrams = [...document.querySelectorAll('.mermaid')];
const sources = diagrams.map((element) => element.textContent);

async function renderDiagrams() {
  await document.fonts.ready;
  const style = getComputedStyle(document.documentElement);
  const token = (name) => style.getPropertyValue(name).trim();
  mermaid.initialize({
    startOnLoad: false,
    securityLevel: 'strict',
    theme: 'base',
    fontFamily: 'Geist, system-ui, sans-serif',
    themeVariables: {
      darkMode: document.documentElement.dataset.theme === 'dark',
      fontFamily: token('--bb-typography-body-family'),
      fontSize: getComputedStyle(document.body).fontSize,
      background: token('--bb-surface-card'),
      primaryColor: token('--bb-surface-card'),
      primaryTextColor: token('--bb-text-primary'),
      primaryBorderColor: token('--in-link'),
      secondaryColor: token('--in-panel-raised'),
      tertiaryColor: token('--bb-surface-background'),
      lineColor: token('--bb-text-muted'),
      edgeLabelBackground: token('--bb-surface-card'),
      clusterBkg: token('--in-panel-raised'),
      clusterBorder: token('--in-border'),
    },
    flowchart: { htmlLabels: false, useMaxWidth: true },
  });
  for (const [index, element] of diagrams.entries()) {
    element.textContent = sources[index];
    element.removeAttribute('data-processed');
    try {
      const { svg, bindFunctions } = await mermaid.render(`insonic-diagram-${index}`, sources[index]);
      element.innerHTML = svg;
      element.dataset.processed = 'true';
      bindFunctions?.(element);
    } catch {
      element.textContent = sources[index];
      if (!element.parentElement.querySelector('.diagram-error')) {
        const message = document.createElement('p');
        message.className = 'diagram-error';
        message.textContent = 'The diagram could not be rendered. Its source is available below.';
        element.after(message);
      }
    }
  }
}

let renderQueue = Promise.resolve();
function scheduleRender() { if (diagrams.length) renderQueue = renderQueue.then(renderDiagrams).catch(() => {}); }
window.addEventListener('insonic-themechange', scheduleRender);
scheduleRender();
