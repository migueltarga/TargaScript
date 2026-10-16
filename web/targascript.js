// TargaScript for the browser.
//
// Loads targascript.wasm, which must sit next to this file, and runs the
// page's <script type="text/targascript"> tags in order. In a worker created
// with new Worker("targascript.js?src=a.tg&src=b.tg") it runs those files.
// Once loaded, targascriptRun(source, onOutput) runs a program in a fresh
// environment and returns its error message, or null. When the program
// pauses at an await, it returns a promise of one of those instead.
//
// targascriptCheck(source) returns the errors it can find without running
// it, as objects with line, column and message.
//
// In Node, require() returns a function that takes the bytes of
// targascript.wasm and resolves to targascriptRun, with the check function
// as its check property.
(() => {
  const encoder = new TextEncoder();
  const decoder = new TextDecoder();
  const FAILED = 0x80000000;

  // JavaScript values seen by the interpreter, by handle. Objects keep the
  // same handle while referenced and are counted; the first five are fixed.
  const values = [undefined, null, true, false, globalThis];
  const counts = [];
  const ids = new Map([[globalThis, 4]]);
  const free = [];

  const handle = (v) => {
    switch (v) {
      case undefined: return 0;
      case null: return 1;
      case true: return 2;
      case false: return 3;
    }
    const isObject = typeof v === "object" || typeof v === "function";
    if (isObject && ids.has(v)) {
      const id = ids.get(v);
      if (id > 4) counts[id]++;
      return id;
    }
    const id = free.length ? free.pop() : values.length;
    values[id] = v;
    counts[id] = 1;
    if (isObject) ids.set(v, id);
    return id;
  };

  const release = (h) => {
    if (h > 4 && --counts[h] === 0) {
      ids.delete(values[h]);
      values[h] = undefined;
      free.push(h);
    }
  };

  let wasm;
  const view = () => new DataView(wasm.memory.buffer);
  const bytes = (ptr, len) => new Uint8Array(wasm.memory.buffer, ptr, len);
  const text = (ptr, len) => decoder.decode(bytes(ptr, len));
  const list = (ptr, len) => {
    const memory = view();
    return Array.from({ length: len }, (_, i) => values[memory.getUint32(ptr + i * 4, true)]);
  };

  // Runs fn and returns the handle of its result, or of the error message
  // with the FAILED bit set when it throws.
  const attempt = (fn) => {
    try {
      return handle(fn());
    } catch (err) {
      return (handle(String(err instanceof Error ? err.message : err)) | FAILED) >>> 0;
    }
  };

  let pending = new Uint8Array(0);

  const host = {
    get: (h, p, n) => attempt(() => values[h][text(p, n)]),
    set: (h, p, n, v) => attempt(() => { values[h][text(p, n)] = values[v]; }),
    call: (h, p, n, a, len) => attempt(() => {
      const target = values[h];
      const name = text(p, n);
      if (typeof target[name] !== "function") throw new TypeError(`${name} is not a function`);
      return target[name](...list(a, len));
    }),
    invoke: (h, a, len) => attempt(() => values[h](...list(a, len))),
    construct: (h, a, len) => attempt(() => new values[h](...list(a, len))),
    kind: (h) => {
      const v = values[h];
      if (v === undefined) return 0;
      if (v === null) return 1;
      switch (typeof v) {
        case "boolean": return 2;
        case "number": return 3;
        case "string": return 4;
        case "function": return 7;
      }
      return Array.isArray(v) ? 5 : 6;
    },
    number: (h) => values[h],
    makeNumber: (f) => handle(f),
    stringLength: (h) => (pending = encoder.encode(String(values[h]))).length,
    stringRead: (ptr) => { bytes(ptr, pending.length).set(pending); },
    makeString: (p, n) => handle(text(p, n)),
    length: (h) => values[h].length,
    index: (h, i) => handle(values[h][i]),
    makeArray: () => handle([]),
    push: (h, v) => { values[h].push(values[v]); },
    makeObject: () => handle({}),
    makeFunction: (id) => handle(function (...args) {
      const result = wasm.callback(id, handle(args));
      const value = values[result];
      release(result);
      return value;
    }),
    retain: (h) => { if (h > 4) counts[h]++; },
    release,
  };

  // The few WASI functions the interpreter's runtime uses. There is no file
  // system, so load reports that files can't be opened.
  const EBADF = 8, ENOTCAPABLE = 76;
  const lines = { 1: "", 2: "" };
  const system = {
    fd_write: (fd, iovs, count, written) => {
      const memory = view();
      let total = 0;
      for (let i = 0; i < count; i++) {
        const ptr = memory.getUint32(iovs + i * 8, true);
        const len = memory.getUint32(iovs + i * 8 + 4, true);
        lines[fd] = (lines[fd] || "") + text(ptr, len);
        total += len;
      }
      const parts = lines[fd].split("\n");
      lines[fd] = parts.pop();
      parts.forEach((line) => (fd === 2 ? console.error : console.log)(line));
      memory.setUint32(written, total, true);
      return 0;
    },
    random_get: (ptr, len) => {
      for (let i = 0; i < len; i += 65536) {
        crypto.getRandomValues(bytes(ptr + i, Math.min(65536, len - i)));
      }
      return 0;
    },
    clock_time_get: (id, precision, out) => {
      view().setBigUint64(out, BigInt(Math.round(Date.now() * 1e6)), true);
      return 0;
    },
    args_sizes_get: (argc, size) => { view().setUint32(argc, 0, true); view().setUint32(size, 0, true); return 0; },
    args_get: () => 0,
    proc_exit: (code) => { throw new Error(`TargaScript exited with code ${code}`); },
    fd_close: () => 0,
    fd_fdstat_get: () => EBADF,
    fd_filestat_get: () => EBADF,
    fd_prestat_get: () => EBADF,
    fd_prestat_dir_name: () => EBADF,
    fd_read: () => EBADF,
    path_open: () => ENOTCAPABLE,
  };

  const imports = { targascript: host, wasi_snapshot_preview1: system };

  // Runs source, in a fresh environment or the shared one, and returns the
  // error message or null, or a promise of them if the program awaits.
  const evaluate = (source, onOutput, fresh) => {
    const s = handle(String(source));
    const o = handle(onOutput);
    const result = wasm.evaluate(s, o, fresh ? 1 : 0);
    release(s);
    release(o);
    const error = result ? values[result] : null;
    release(result);
    return error;
  };

  // Finds the errors in source without running it: an array of objects with
  // line, column and message.
  const check = (source) => {
    const s = handle(String(source));
    const result = wasm.check(s);
    release(s);
    const problems = JSON.parse(values[result]);
    release(result);
    return problems;
  };

  const start = (instance) => {
    wasm = instance.exports;
    wasm._initialize();
    const run = (source, onOutput = () => {}) => evaluate(source, onOutput, true);
    run.check = check;
    return run;
  };

  if (typeof module === "object" && module.exports) {
    module.exports = (wasmBytes) =>
      WebAssembly.instantiate(wasmBytes, imports).then(({ instance }) => start(instance));
    return;
  }

  const inWorker = typeof document === "undefined";
  const base = inWorker ? self.location.href : document.currentScript.src;
  const url = (path) => new URL(path, base).href;

  // print in page scripts goes to the console, one line at a time
  let line = "";
  const toConsole = (chunk) => {
    const parts = (line + chunk).split("\n");
    line = parts.pop();
    parts.forEach((part) => console.log(part));
  };

  const fetchText = (src) => fetch(src).then((response) => {
    if (!response.ok) throw new Error(`could not load ${src}: HTTP ${response.status}`);
    return response.text();
  });

  const scripts = () => {
    if (inWorker) {
      return new URL(self.location.href).searchParams.getAll("src")
        .map((src) => ({ name: src, load: () => fetchText(url(src)) }));
    }
    return [...document.querySelectorAll('script[type="text/targascript"]')].map((tag, i) =>
      tag.src
        ? { name: tag.getAttribute("src"), load: () => fetchText(tag.src) }
        : { name: `inline script ${i + 1}`, load: () => Promise.resolve(tag.textContent) });
  };

  const domReady = inWorker || document.readyState !== "loading"
    ? Promise.resolve()
    : new Promise((resolve) => document.addEventListener("DOMContentLoaded", resolve));

  const instantiate = () => fetch(url("targascript.wasm"))
    .then((response) => response.arrayBuffer())
    .then((buffer) => WebAssembly.instantiate(buffer, imports));

  Promise.all([instantiate(), domReady])
    .then(async ([{ instance }]) => {
      globalThis.targascriptRun = start(instance);
      globalThis.targascriptCheck = check;
      for (const script of scripts()) {
        const source = await script.load();
        const report = (error) => {
          if (error) console.error(`TargaScript error in ${script.name}: ${error}`);
        };
        const error = evaluate(source, toConsole, false);
        if (error instanceof Promise) error.then(report);
        else report(error);
      }
    })
    .catch((err) => console.error("TargaScript:", err));
})();
