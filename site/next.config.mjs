// SPDX-License-Identifier: Apache-2.0
import { fileURLToPath } from 'node:url';
/** @type {import('next').NextConfig} */
const config = {
  output: 'export',
  trailingSlash: true,
  images: { unoptimized: true },
  poweredByHeader: false,
  reactStrictMode: true,
  turbopack: { root: fileURLToPath(new URL('..', import.meta.url)) },
};

export default config;
