// realData.ts — bridges the real WASM compiler + HTTP APIs to the FORMQL interface.

import type {
  Table, Relationship, FunctionSpec, Preset, Token, TokenKind,
  CompleteItem, CompleteResult, CompileResult, CompileMode,
  DiagnosticWarning, VerifyResult, ExecuteResult, WasmSchemaTable,
  WasmSchemaRelationship,
} from './types';

// ─── Runtime state ────────────────────────────────────────────────
let _catalogJSON: string | null = null;

// ─── Dynamic schema (populated on init) ──────────────────────────
let TABLES: Table[] = [];
let RELATIONSHIPS: Relationship[] = [];

// ─── Static data ─────────────────────────────────────────────────
const FUNCTIONS: FunctionSpec[] = [
  { name: 'STRING',   sig: 'STRING(any) → text',                   desc: 'Cast any value to text' },
  { name: 'DATE',     sig: 'DATE(text) → date',                    desc: 'Parse text to date' },
  { name: 'UPPER',    sig: 'UPPER(text) → text',                   desc: 'Uppercase a string' },
  { name: 'LOWER',    sig: 'LOWER(text) → text',                   desc: 'Lowercase a string' },
  { name: 'TRIM',     sig: 'TRIM(text) → text',                    desc: 'Trim whitespace' },
  { name: 'LEN',      sig: 'LEN(text) → int',                      desc: 'Length of string' },
  { name: 'ROUND',    sig: 'ROUND(numeric, int) → numeric',        desc: 'Round to N decimal places' },
  { name: 'ABS',      sig: 'ABS(numeric) → numeric',               desc: 'Absolute value' },
  { name: 'IF',       sig: 'IF(bool, a, b) → a|b',                 desc: 'Conditional expression' },
  { name: 'AND',      sig: 'AND(bool, bool, ...) → bool',          desc: 'Logical AND' },
  { name: 'OR',       sig: 'OR(bool, bool, ...) → bool',           desc: 'Logical OR' },
  { name: 'NOT',      sig: 'NOT(bool) → bool',                     desc: 'Logical NOT' },
  { name: 'COALESCE', sig: 'COALESCE(a, b, ...) → first non-null', desc: 'First non-null argument' },
  { name: 'NULLVALUE',sig: 'NULLVALUE(a, b, ...) → first non-null',desc: 'Alias for COALESCE' },
  { name: 'ISNULL',   sig: 'ISNULL(any) → bool',                   desc: 'True when value is NULL' },
  { name: 'ISBLANK',  sig: 'ISBLANK(any) → bool',                  desc: 'True when NULL or empty string' },
  { name: 'TODAY',    sig: 'TODAY() → date',                       desc: 'Current date' },
];

const PRESETS: Preset[] = [
  {
    id: 'customer-tag',
    title: 'Customer tag',
    description: 'Email & total — common label format for receipts.',
    baseTable: 'rental_contract',
    mode: 'formula',
    formula: 'customer_id__rel.email & " / " & STRING(actual_total)',
  },
  {
    id: 'rep-chain',
    title: 'Rep manager chain',
    description: 'Two-hop traversal across rep → manager → branch.',
    baseTable: 'rental_contract',
    mode: 'formula',
    formula: 'rep_id__rel.manager_id__rel.first_name & " @ " & rep_id__rel.branch_id__rel.name',
  },
  {
    id: 'vehicle-badge',
    title: 'Vehicle badge',
    description: 'Model name and year, hyphen-joined.',
    baseTable: 'rental_contract',
    mode: 'formula',
    formula: 'vehicle_id__rel.model_name & " / " & STRING(vehicle_id__rel.model_year)',
  },
  {
    id: 'quote-route',
    title: 'Quote route',
    description: 'Pickup → dropoff branch names on a rental offer.',
    baseTable: 'rental_offer',
    mode: 'formula',
    formula: 'pickup_branch_id__rel.name & " -> " & dropoff_branch_id__rel.name',
  },
  {
    id: 'contract-doc',
    title: 'Contract report',
    description: 'Multi-field document: total, customer email, vehicle.',
    baseTable: 'rental_contract',
    mode: 'document',
    formula: 'actual_total,\ncustomer_id__rel.email AS customer_email,\nvehicle_id__rel.model_name AS vehicle_model',
  },
  {
    id: 'resale-doc',
    title: 'Resale margin report',
    description: 'Resale price, buyer email, and manager last name.',
    baseTable: 'resale_sale',
    mode: 'document',
    formula: 'sale_price,\ncustomer_id__rel.email AS buyer_email,\nrep_id__rel.manager_id__rel.last_name AS manager_last_name',
  },
];

// ─── Lookup helpers ───────────────────────────────────────────────
function tableByName(name: string): Table | undefined {
  return TABLES.find((t) => t.name === name);
}

function relsFrom(name: string): Relationship[] {
  return RELATIONSHIPS.filter((r) => r.from_table === name);
}

// ─── Tokenizer ────────────────────────────────────────────────────
const KEYWORDS = ['AS'];

function tokenize(src: string): Token[] {
  const tokens: Token[] = [];
  const re = /(\s+)|("(?:[^"\\]|\\.)*")|(\d+(?:\.\d+)?)|([A-Za-z_][A-Za-z0-9_]*)|((?:&|\+|-|\*|\/|<=|>=|<|>|=|,|\(|\)|\.))|(\\S)/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(src)) !== null) {
    const [whole, ws, str, num, ident, op] = m;
    const start = m.index;
    const end = start + whole.length;
    let kind: TokenKind;
    if (ws) kind = 'ws';
    else if (str) kind = 'string';
    else if (num) kind = 'number';
    else if (ident) {
      if (KEYWORDS.indexOf(whole.toUpperCase()) !== -1) kind = 'keyword';
      else if (FUNCTIONS.find((f) => f.name === whole)) kind = 'fn';
      else if (whole.endsWith('__rel')) kind = 'rel';
      else kind = 'col';
    } else if (op) kind = 'punct';
    else kind = 'other';
    tokens.push({ start, end, text: whole, kind });
  }
  return tokens;
}

// ─── Context table (cursor-based traversal) ───────────────────────
function resolveChain(baseTable: string, segments: string[]): { table: string; ok: boolean } {
  let table = baseTable;
  for (const seg of segments) {
    const rel = RELATIONSHIPS.find((r) => r.from_table === table && r.name === seg);
    if (!rel) return { table, ok: false };
    table = rel.to_table;
  }
  return { table, ok: true };
}

function contextTable(baseTable: string, src: string, cursor: number): string {
  const before = src.slice(0, cursor);
  const tail = (before.match(/[A-Za-z0-9_.]+\.?$/) || [''])[0];
  if (!tail.includes('.')) return baseTable;
  const segs = tail.split('.').slice(0, -1).filter(Boolean);
  const r = resolveChain(baseTable, segs);
  return r.ok ? r.table : baseTable;
}

// ─── Autocomplete (real WASM) ─────────────────────────────────────
const KIND_REL = 6;
const KIND_FUNCTION = 3;

function complete(baseTable: string, src: string, cursor: number): CompleteResult {
  if (!window.FormQL || !_catalogJSON) {
    return { items: [], context: baseTable, partial: '' };
  }
  try {
    const options = { baseTable, maxRelationshipDepth: 30 };
    const result = window.FormQL.completeCatalogJSON(_catalogJSON, src, cursor, options);
    if (!result.ok) return { items: [], context: contextTable(baseTable, src, cursor), partial: '' };
    const items: CompleteItem[] = (result.items || []).map((item) => {
      const kind: CompleteItem['kind'] =
        item.kind === KIND_REL ? 'relationship' :
        item.kind === KIND_FUNCTION ? 'function' : 'column';
      return {
        kind,
        label: item.label,
        detail: item.detail || '',
        indexed: item.indexed !== false,
      };
    });
    return { items, context: contextTable(baseTable, src, cursor), partial: '' };
  } catch {
    return { items: [], context: baseTable, partial: '' };
  }
}

// ─── Compile (real WASM, output mapped to shared shape) ──────────
function emptyCompile(baseTable: string, mode: CompileMode, message: string): CompileResult {
  return {
    ok: false, sql: '',
    errors: [{ message, start: 0, end: 0, positioned: false }],
    warnings: [],
    hir: { base_table: baseTable, mode, joins: [], projections: [] },
  };
}

function compile(baseTable: string, src: string, mode: CompileMode): CompileResult {
  if (!window.FormQL || !_catalogJSON) return emptyCompile(baseTable, mode, 'compiler not ready');
  if (!src.trim()) return emptyCompile(baseTable, mode, 'empty formula');

  try {
    const options = { baseTable, fieldAlias: 'result', maxRelationshipDepth: 30 };
    const result = mode === 'document'
      ? window.FormQL.compileDocumentCatalogJSON(_catalogJSON, src, options)
      : window.FormQL.compileCatalogJSON(_catalogJSON, src, options);

    if (!result.ok) {
      const errMsg = result.error?.message ?? 'compile failed';
      const errHint = result.error?.hint ?? '';
      const errPos = (result.error?.position ?? -1) >= 0 ? result.error!.position : -1;
      const errStart = errPos >= 0 ? errPos : 0;
      let errEnd = errStart;
      if (errPos >= 0) {
        errEnd = errStart + 1;
        while (errEnd < src.length && /[A-Za-z0-9_]/.test(src[errEnd])) errEnd++;
      }
      return {
        ok: false, sql: '',
        errors: [{ message: errMsg, hint: errHint, start: errStart, end: errEnd, positioned: errPos >= 0 }],
        warnings: [],
        hir: { base_table: baseTable, mode, joins: [], projections: [] },
      };
    }

    const comp = result.compilation ?? {};
    const hir = comp.hir ?? {};
    const sql = comp.sql?.query ?? '';

    const joins = (hir.joins ?? []).map((j) => {
      const path = j.path ?? [];
      const via = path.length > 0 ? path[path.length - 1] : (j.join_column ?? '') + '__rel';
      const from = j.from_table ?? baseTable;
      const rel = RELATIONSHIPS.find((r) => r.from_table === from && r.name === via);
      return { from, to: j.to_table ?? '', via, indexed: rel ? rel.indexed : true };
    });

    const projections = mode === 'document'
      ? (hir.fields ?? []).map((f) => ({ path: f.alias, table: baseTable, column: f.alias, type: f.type ?? 'text' }))
      : sql ? [{ path: 'result', table: baseTable, column: 'result', type: hir.expr?.type ?? 'text' }] : [];

    const warnings: DiagnosticWarning[] = (hir.warnings ?? []).map((w) => ({
      message: typeof w === 'string' ? w : w.message,
      code: typeof w === 'object' ? w.code : undefined,
      start: 0,
      end: 0,
    }));

    return { ok: true, sql, hir: { base_table: baseTable, mode, joins, projections }, errors: [], warnings };
  } catch (e) {
    return emptyCompile(baseTable, mode, e instanceof Error ? e.message : 'compile failed');
  }
}

// ─── Verify (HTTP) ────────────────────────────────────────────────
function verify(sql: string): Promise<VerifyResult> {
  if (!sql) return Promise.resolve({ ok: false, message: 'no SQL to verify' });
  return fetch('/api/verify-sql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sql, verify_mode: 'syntax' }),
  }).then((r) => r.json()).then((payload: { ok: boolean; verification?: { ok: boolean }; error?: { message: string } }) => {
    const verOk = payload.ok && payload.verification?.ok;
    return {
      ok: !!verOk,
      message: verOk ? 'syntax ok' : (payload.error?.message ?? 'verification failed'),
      plan_hint: verOk ? 'syntax check passed' : null,
    };
  });
}

// ─── Execute (HTTP) ───────────────────────────────────────────────
function execute(compileResult: CompileResult): Promise<ExecuteResult> {
  if (!compileResult?.ok) return Promise.resolve({ ok: false, message: 'cannot execute — compile failed' });
  const t0 = Date.now();
  return fetch('/api/execute-sql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sql: compileResult.sql, max_rows: 25 }),
  }).then((r) => r.json()).then((payload: { ok: boolean; columns?: string[]; rows?: Record<string, unknown>[]; error?: { message: string } }) => {
    if (!payload.ok) return { ok: false, message: payload.error?.message ?? 'execution failed' };
    const cols = payload.columns ?? [];
    const rows = (payload.rows ?? []).map((r) => {
      const row: Record<string, unknown> = {};
      cols.forEach((c) => { row[c] = r[c]; });
      return row;
    });
    return {
      ok: true,
      columns: cols.map((c) => ({ name: c, type: 'text' })),
      rows,
      elapsed_ms: Date.now() - t0,
    };
  });
}

// ─── WASM loader ──────────────────────────────────────────────────
function loadWASM(): Promise<void> {
  const go = new window.Go();
  return fetch('/wasm/formql.wasm').then((r) => {
    if (!r.ok) throw new Error('wasm fetch failed: ' + r.status);
    return r.arrayBuffer();
  }).then((bytes) => WebAssembly.instantiate(bytes, go.importObject))
    .then((result) => {
      go.run(result.instance);
      return new Promise<void>((resolve) => setTimeout(resolve, 0));
    }).then(() => {
      if (!window.FormQL) throw new Error('FormQL wasm runtime did not initialize');
    });
}

// ─── Catalog loader ───────────────────────────────────────────────
function loadCatalog(): Promise<void> {
  return fetch('/api/catalog/rental-agency').then((r) => {
    if (!r.ok) throw new Error('catalog fetch failed: ' + r.status);
    return r.json();
  }).then((data: unknown) => {
    _catalogJSON = JSON.stringify(data);
  });
}

function loadSchemaInfo(baseTable: string): Promise<void> {
  const url = '/api/schema-info/rental-agency?base_table=' + encodeURIComponent(baseTable);
  return fetch(url).then((r) => {
    if (!r.ok) throw new Error('schema-info fetch failed: ' + r.status);
    return r.json();
  }).then((payload: { ok: boolean; info?: { tables?: WasmSchemaTable[]; relationships?: WasmSchemaRelationship[] }; error?: { message: string } }) => {
    if (!payload.ok) throw new Error(payload.error?.message ?? 'schema-info failed');
    const info = payload.info ?? {};
    TABLES = (info.tables ?? []).map((t) => ({
      name: t.name,
      pk: 'id',
      columns: (t.columns ?? []).map((c) => ({ name: c.name, type: c.type })),
    }));
    RELATIONSHIPS = (info.relationships ?? []).map((r) => ({
      from_table: r.from_table,
      name: r.name,
      to_table: r.to_table,
      indexed: r.join_column_indexed !== false,
    }));
  });
}

// ─── Public init ──────────────────────────────────────────────────
function init(): Promise<void> {
  return Promise.all([loadWASM(), loadCatalog()])
    .then(() => loadSchemaInfo('rental_contract'));
}

// ─── Exported FORMQL object ───────────────────────────────────────
export const formql = {
  get TABLES() { return TABLES; },
  get RELATIONSHIPS() { return RELATIONSHIPS; },
  FUNCTIONS,
  PRESETS,
  tableByName,
  relsFrom,
  tokenize,
  complete,
  contextTable,
  compile,
  verify,
  execute,
  init,
};

export function fieldsToDocument(fields: Array<{ expr: string; alias: string }>): string {
  return fields.map((f) => `${f.expr} AS ${f.alias}`).join(',\n');
}
