// SPDX-License-Identifier: Apache-2.0
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export function getBrandMessaging() {
  const brand = JSON.parse(readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../../brand/kit/brand.json'), 'utf8'));
  for (const role of ['slogan', 'short_description', 'long_description']) {
    const message = brand.messaging?.[role];
    if (message?.status !== 'approved' || typeof message.text !== 'string' || !message.text.trim()) throw new Error(`The upstream brand has no approved ${role}. Resolve messaging with its owner before publication.`);
  }
  return { title: brand.title, slogan: brand.messaging.slogan.text, description: brand.messaging.short_description.text, longDescription: brand.messaging.long_description.text };
}
