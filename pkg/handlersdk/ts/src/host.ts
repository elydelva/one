// host.ts — type-safe bindings to the One CLI host functions exposed by the
// wazero runtime. The actual functions are imported by Javy at compile time
// from the `one` host module. All buffer plumbing happens here so handler
// authors only see typed JSON.

// These extern function declarations are bound at link time by Javy.
declare function read_inputs(ptr: number, max: number): number;
declare function write_output(ptr: number, len: number): void;
declare function creds_get(keyPtr: number, keyLen: number, outPtr: number, outMax: number): number;
declare function http_request(reqPtr: number, reqLen: number, outPtr: number, outMax: number): number;
declare function log_write(levelPtr: number, levelLen: number, msgPtr: number, msgLen: number): void;
declare function fail(codePtr: number, codeLen: number, msgPtr: number, msgLen: number, hintPtr: number, hintLen: number): void;
declare function time_now(): bigint;
declare function time_sleep(ms: number): void;

const IO_BUF_SIZE = 1 << 20;
const ioBuf = new Uint8Array(IO_BUF_SIZE);
const enc = new TextEncoder();
const dec = new TextDecoder();

function writeStr(s: string): { ptr: number; len: number } {
  const bytes = enc.encode(s);
  ioBuf.set(bytes);
  return { ptr: 0, len: bytes.byteLength };
}

export interface Envelope<I> {
  action: string;
  inputs: I;
  context: { trace_id?: string; dry_run?: boolean };
}

export function readInputs<I = unknown>(): Envelope<I> {
  const n = read_inputs(0, IO_BUF_SIZE);
  if (n < 0) throw new Error('read_inputs failed');
  return JSON.parse(dec.decode(ioBuf.subarray(0, n))) as Envelope<I>;
}

export function writeOutput(value: unknown): void {
  const bytes = enc.encode(JSON.stringify(value));
  ioBuf.set(bytes);
  write_output(0, bytes.byteLength);
}

export const host = {
  creds: {
    get(key: string): string {
      const k = writeStr(key);
      // Reuse another offset for output to avoid clobbering the key.
      const outOff = k.len;
      const n = creds_get(k.ptr, k.len, outOff, IO_BUF_SIZE - outOff);
      if (n < 0) throw new Error(`creds.get: not allowed: ${key}`);
      return dec.decode(ioBuf.subarray(outOff, outOff + n));
    },
  },
  http: {
    request(req: {
      method: string;
      url: string;
      headers?: Record<string, string>;
      body?: string;
    }): { status: number; headers: Record<string, string>; body: string } {
      const payload = enc.encode(JSON.stringify(req));
      ioBuf.set(payload);
      const outOff = payload.byteLength;
      const n = http_request(0, payload.byteLength, outOff, IO_BUF_SIZE - outOff);
      if (n < 0) throw new Error(`http.request: not allowed: ${req.url}`);
      const resp = JSON.parse(dec.decode(ioBuf.subarray(outOff, outOff + n)));
      const body = resp.body_b64 ? atob(resp.body_b64) : '';
      return { status: resp.status, headers: resp.headers ?? {}, body };
    },
  },
  log: {
    debug(msg: string) { logAt('debug', msg); },
    info(msg: string) { logAt('info', msg); },
    warn(msg: string) { logAt('warn', msg); },
  },
  time: {
    now(): number { return Number(time_now()); },
    sleep(ms: number): void { time_sleep(ms); },
  },
  fail: {
    withCode(code: string, message: string, hint?: string): never {
      const codeB = enc.encode(code);
      const msgB = enc.encode(message);
      const hintB = enc.encode(hint ?? '');
      const off1 = codeB.byteLength;
      const off2 = off1 + msgB.byteLength;
      ioBuf.set(codeB, 0);
      ioBuf.set(msgB, off1);
      ioBuf.set(hintB, off2);
      fail(0, codeB.byteLength, off1, msgB.byteLength, off2, hintB.byteLength);
      throw new Error(`handler.fail: ${code}`);
    },
  },
};

function logAt(level: 'debug' | 'info' | 'warn', msg: string) {
  const levelB = enc.encode(level);
  const msgB = enc.encode(msg);
  ioBuf.set(levelB, 0);
  ioBuf.set(msgB, levelB.byteLength);
  log_write(0, levelB.byteLength, levelB.byteLength, msgB.byteLength);
}
