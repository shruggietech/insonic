// SPDX-License-Identifier: Apache-2.0
// Self-contained so the controlled renderer can evaluate the same observer.
export function desktopDensityGeometry() {
  const host = document.querySelector<HTMLElement>('[data-bb-host="wails"]');
  if (!host) throw new Error('Desktop density host is absent.');
  const target = parseFloat(getComputedStyle(host).getPropertyValue('--bb-interaction-target-minimum'));
  if (!(target >= 44)) throw new Error('The governed activation target changed.');
  const glyphs = [...host.querySelectorAll<HTMLInputElement>('input[type="checkbox"],input[type="radio"]')]
    .filter(node => node.getClientRects().length).map(node => {
      const box = node.getBoundingClientRect(), label = node.closest('label');
      const area = label?.getBoundingClientRect();
      if (!label || label.control !== node || !area || area.width + 1 < target || area.height + 1 < target
          || Math.abs(box.width - box.height) > 1 || box.width >= target || box.width <= 0)
        throw new Error('Selection glyph or compound activation target has invalid geometry.');
      return {width: box.width, height: box.height, target_width: area.width, target_height: area.height};
    });
  const actions = [...host.querySelectorAll<HTMLButtonElement>('button.bb-button')]
    .filter(node => node.getClientRects().length).map(node => {
      const box = node.getBoundingClientRect(), style = getComputedStyle(node);
      const range = document.createRange(); range.selectNodeContents(node);
      const text = range.getBoundingClientRect();
      const intended = Math.max(target, text.width + parseFloat(style.paddingLeft) + parseFloat(style.paddingRight)
        + parseFloat(style.borderLeftWidth) + parseFloat(style.borderRightWidth));
      if (box.width + 1 < target || box.height + 1 < target || box.width > intended + 2)
        throw new Error('Action geometry: width=' + box.width + ', height=' + box.height + ', intended=' + intended);
      return {width: box.width, height: box.height};
    });
  const main = host.querySelector<HTMLElement>('.desktop-content');
  if (!main || main.getBoundingClientRect().height < target) throw new Error('Header wrapping hid the main commands.');
  for (const node of [document.documentElement, host, main])
    if (node.scrollWidth > node.clientWidth + 1) throw new Error('Desktop horizontal overflow: ' + node.tagName + ' ' + node.clientWidth + '/' + node.scrollWidth);
  for (const node of host.querySelectorAll<HTMLElement>('.table-scroll'))
    if (node.getBoundingClientRect().right > main.getBoundingClientRect().right + 1)
      throw new Error('Table overflow escaped its own region.');
  return {glyphs, actions, main_height: main.getBoundingClientRect().height, viewport_width: innerWidth, viewport_height: innerHeight};
}
