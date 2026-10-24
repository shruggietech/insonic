// SPDX-License-Identifier: Apache-2.0
import { createRoot } from 'react-dom/client';
import { App } from './App';
import { qualifyDesktop, qualifyHelpOnly, qualificationMounted, runQualificationOnce, qualificationProgress } from './qualification';
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
    await qualificationProgress(bridge, "journey-entered");
    await qualificationProgress(bridge, "journey-fixtures-request");
    const fixtures = await bridge.NativeQualificationData?.();
    await qualificationProgress(bridge, "journey-fixtures-returned");
    if (fixtures?.result?.help_only) {
      await qualificationProgress(bridge, "help-open-request");
      const response = await bridge.OpenHelp!();
      await qualificationProgress(bridge, "help-open-returned");
      if(response.error)throw new Error(response.error.message);
      response.result = {...response.result,...await qualifyHelpOnly(String(response.result?.version),bridge)};
      await qualificationProgress(bridge, "help-journey-complete");
      await qualificationProgress(bridge, "complete-smoke-request");
      await bridge.CompleteSmoke(response);
      return;
    }
    await qualificationProgress(bridge, 'ready');
    await qualificationMounted();
    await qualificationProgress(bridge, 'mounted');
    const response = await bridge.Show();
    if (response.error) throw new Error(response.error.message);
    const flags = await qualifyDesktop(bridge);
    response.result = { ...response.result, ...flags };
    await qualificationProgress(bridge, 'complete');
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
    qualificationMode = (await qualificationProgress(bridge, 'module')) ?? false;
    if (!bridge.NativeQualificationData) return;
    const response = await bridge.NativeQualificationData();
    if (response.error) throw new Error(response.error.message);
    if (!response.result?.audio_path && !response.result?.help_only) return;
    await qualificationProgress(bridge, 'fixtures');
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
