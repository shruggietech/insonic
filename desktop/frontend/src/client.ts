// SPDX-License-Identifier: Apache-2.0
export type Obj = Record<string, any>;
export interface Response {
  workspace_id?: string;
  runtime_session_id?: string;
  result?: any;
  error?: { code: string; message: string; diagnostics?: Obj[] };
}
export interface NativeBridge {
  Show(): Promise<Response>;
  Operate(request: Obj): Promise<Response>;
  SelectWorkspace(
    path: string,
    name: string,
    create: boolean,
  ): Promise<Response>;
  Choose(kind: string): Promise<string>;
  Playback(
    mediaID: string,
    mediaRevision: number,
    recordingRevision: number,
    documentDigest: string,
  ): Promise<Response>;
  ClosePlayback(url: string): Promise<Response>;
  VerifyPlayback(url: string): Promise<Response>;
  Credential(args: string[], input: string): Promise<Response>;
  CompleteSmoke(response: Response): Promise<void>;
  NativeQualificationData?(): Promise<Response>;
}
declare global {
  interface Window {
    go: { desktop: { Bridge: NativeBridge } };
    runtime?: { EventsOn(name: string, callback: () => void): void };
    insonicQualification?: () => Promise<void>;
  }
}
export class RuntimeError extends Error {
  constructor(
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}
export class StaleResponse extends Error {
  constructor() {
    super('The selection changed. Refresh the current view.');
  }
}
export function result(response: Response) {
  if (response.error)
    throw new RuntimeError(response.error.code, response.error.message);
  return response.result;
}
// The catalog is authoritative; this tracks only ephemeral request ownership.
export class Client {
  workspace = '';
  generation = 0;
  constructor(readonly bridge: NativeBridge) {}
  select(id: string) {
    this.workspace = id;
    this.generation++;
  }
  invalidate() {
    this.generation++;
  }
  async call(operation: string, itemID = '', data?: Obj): Promise<any> {
    const workspace = this.workspace,
      generation = this.generation;
    const response = await this.bridge.Operate({
      operation,
      workspace_id: workspace,
      ...(itemID ? { item_id: itemID } : {}),
      ...(data === undefined ? {} : { data }),
    });
    if (
      generation !== this.generation ||
      workspace !== this.workspace ||
      (response.workspace_id && response.workspace_id !== workspace)
    )
      throw new StaleResponse();
    return result(response);
  }
}
export function rationalSeconds(value: string): number | null {
  if (!/^-?\d+(\/\d+)?$/.test(value)) return null;
  const [a, b = '1'] = value.split('/');
  const denominator = BigInt(b);
  if (denominator === 0n) return null;
  const milliseconds = (BigInt(a) * 1000n) / denominator;
  if (milliseconds < 0n || milliseconds > BigInt(Number.MAX_SAFE_INTEGER))
    return null;
  return Number(milliseconds) / 1000;
}
export function appendCapture(page: Obj, previous?: Obj): Obj {
  if (
    page.encoding !== 'base64' ||
    (previous &&
      (previous.sha256 !== page.sha256 || previous.next_offset !== page.offset))
  )
    throw new RuntimeError('conflict', 'The capture changed. Read it again.');
  const decoded = Uint8Array.from(atob(page.data), (c) => c.charCodeAt(0));
  const old = previous?.bytes ?? new Uint8Array();
  if (old.length + decoded.length > 64 << 20)
    throw new RuntimeError(
      'output_limit',
      'The capture exceeds the bounded desktop view. Inspect its report through the CLI.',
    );
  const bytes = new Uint8Array(old.length + decoded.length);
  bytes.set(old);
  bytes.set(decoded, old.length);
  // Decode after joining byte chunks so a multibyte character split at the
  // page boundary retains its exact representation on the following page.
  return {
    ...page,
    bytes,
    text: new TextDecoder('utf-8', { fatal: true }).decode(bytes, {
      stream: !page.complete,
    }),
  };
}
