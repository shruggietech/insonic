// SPDX-License-Identifier: Apache-2.0
import { createRoot } from 'react-dom/client';
import { App } from './App';
import { qualifyDesktop, runQualificationOnce } from './qualification';
import type { NativeBridge } from './client';
import '../../../brand/kit/tokens/interface.css';
import '../../../brand/kit/tokens/typography.css';
import '../../../brand/kit/tokens/spacing.css';
import '../../../brand/kit/web/components.css';
const nativeBridge = window.go.desktop.Bridge;
let qualifying = false;
const failedOperations: string[] = [];
const bridge: NativeBridge = {
  ...nativeBridge,
  Operate: async (request) => {
    const response = await nativeBridge.Operate(request);
    if (qualifying && response.error) {
      failedOperations.push(`${request.operation} (${response.error.code})`);
      if (failedOperations.length > 8) failedOperations.shift();
    }
    return response;
  },
};
createRoot(document.getElementById('root')!).render(<App bridge={bridge} />);
const startQualification = runQualificationOnce(async () => {
  qualifying = true;
  failedOperations.length = 0;
  try {
    await bridge.QualificationStep?.('ready');
    const until = Date.now() + 5000;
    while (
      !document.querySelector('[aria-label="Library controls"]') &&
      Date.now() < until
    )
      await new Promise((resolve) => setTimeout(resolve, 20));
    if (
      !document.querySelector('[data-bb-host="wails"]') ||
      !document.querySelector('form') ||
      !document.querySelector('[aria-label="Library controls"]')
    )
      throw new Error('Desktop screens did not mount.');
    await bridge.QualificationStep?.('mounted');
    const response = await bridge.Show();
    if (response.error) throw new Error(response.error.message);
    const flags = await qualifyDesktop(bridge);
    response.result = { ...response.result, ...flags };
    await bridge.QualificationStep?.('complete');
    await bridge.CompleteSmoke(response);
  } catch (error) {
    await bridge.CompleteSmoke({
      error: {
        code: 'operation_failed',
        message:
          `Desktop screen qualification failed: ${error instanceof Error ? error.message.slice(0, 360) : 'unknown failure'}${failedOperations.length ? `; native failures: ${failedOperations.join(', ')}` : ''}`.slice(
            0,
            512,
          ),
      },
    });
  } finally {
    qualifying = false;
  }
});
window.runtime?.EventsOn('qualification', startQualification);
// Linux can finish the document before a deferred bundle registers its event
// handler. Explicit native fixture readiness starts the same deduplicated run.
void (async () => {
  let qualificationMode = false;
  try {
    qualificationMode = (await bridge.QualificationStep?.('module')) ?? false;
    if (!bridge.NativeQualificationData) return;
    const response = await bridge.NativeQualificationData();
    if (response.error) throw new Error(response.error.message);
    if (!response.result?.audio_path) return;
    await bridge.QualificationStep?.('fixtures');
    await startQualification();
  } catch (error) {
    if (qualificationMode)
      await bridge.CompleteSmoke({
        error: {
          code: 'operation_failed',
          message: `Desktop qualification bootstrap failed: ${error instanceof Error ? error.message.slice(0, 400) : 'Unknown failure.'}`,
        },
      });
  }
})();
