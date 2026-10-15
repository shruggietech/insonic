// SPDX-License-Identifier: Apache-2.0
// Pure routing decisions are shared by static routes, export checks and tests.
function routeSlug(slug, label) {
  if (typeof slug !== 'string' || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug)) throw new Error('Invalid documentation ' + label + ': ' + slug);
  return slug;
}

export function getLatestEntry(manifest) {
  const matches = manifest.versions.filter(entry => entry.version === manifest.latest);
  if (matches.length !== 1) throw new Error('Documentation manifest must elect exactly one latest version.');
  return matches[0];
}

export function latestPath(slug = 'index') {
  routeSlug(slug, 'topic');
  return slug === 'index' ? '/docs/latest/' : '/docs/latest/' + slug + '/';
}

export function latestReferencePath(slug) {
  return '/docs/latest/references/' + routeSlug(slug, 'reference') + '/';
}

export function latestTopicSlugs(manifest) {
  getLatestEntry(manifest);
  return [...new Set(manifest.versions.flatMap(entry => entry.pages.map(page => routeSlug(page.slug, 'topic'))))].sort();
}

export function latestReferenceSlugs(references) {
  return [...new Set(references.map(item => routeSlug(item.slug, 'reference')))].sort();
}
