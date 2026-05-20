// In-process fake host for unit testing handler logic. The real wasm import
// declarations from host.ts are dropped by Javy when compiling, so in tests we
// can shim them with a thin interceptor and verify handler behavior.

export interface FakeHostState {
  inputs: unknown;
  creds: Record<string, string>;
  output?: unknown;
  fail?: { code: string; message: string; hint?: string };
  httpExpectations: Array<{
    match: (req: any) => boolean;
    respond: { status: number; headers?: Record<string, string>; body?: unknown };
  }>;
  httpCalls: any[];
  logs: Array<{ level: string; msg: string }>;
}

export function installFakeHost(initial: Partial<FakeHostState> = {}): FakeHostState {
  const state: FakeHostState = {
    inputs: initial.inputs ?? {},
    creds: initial.creds ?? {},
    httpExpectations: initial.httpExpectations ?? [],
    httpCalls: [],
    logs: [],
  };
  // The real bindings are installed by Javy at compile time. Tests run in
  // Node/Bun where these globals do not exist — the host.ts module would
  // therefore throw on first call. The handler test harness is responsible
  // for stubbing those declarations via vi.stubGlobal before importing the
  // module under test. See README for the recipe.
  return state;
}
