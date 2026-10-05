// SPDX-License-Identifier: Apache-2.0
export default function DocumentationLayout({ children }) {
  return <>{children}<script src="/docs-runtime.js" defer /></>;
}
