// Shared types for the FormQL playground.

export type CompileMode = 'formula' | 'document';
export type StatusState = 'ok' | 'warn' | 'err' | 'idle' | 'pending';
export type TokenKind = 'ws' | 'string' | 'number' | 'keyword' | 'fn' | 'rel' | 'ident' | 'col' | 'punct' | 'other';
export type CompleteItemKind = 'column' | 'relationship' | 'function';

export interface Column {
  name: string;
  type: string;
}

export interface Table {
  name: string;
  pk: string;
  columns: Column[];
}

export interface Relationship {
  from_table: string;
  name: string;
  to_table: string;
  indexed: boolean;
}

export interface FunctionSpec {
  name: string;
  sig: string;
  desc: string;
}

export interface Preset {
  id: string;
  title: string;
  description: string;
  baseTable: string;
  mode: CompileMode;
  formula: string;
}

export interface Token {
  start: number;
  end: number;
  text: string;
  kind: TokenKind;
}

export interface CompleteItem {
  kind: CompleteItemKind;
  label: string;
  detail: string;
  indexed: boolean;
}

export interface CompleteResult {
  items: CompleteItem[];
  context: string;
  partial: string;
}

export interface DiagnosticError {
  message: string;
  hint?: string;
  code?: string;
  start: number;
  end: number;
  positioned: boolean;
}

export interface DiagnosticWarning {
  message: string;
  code?: string;
  start: number;
  end: number;
}

export interface Join {
  from: string;
  to: string;
  via: string;
  indexed: boolean;
}

export interface Projection {
  path: string;
  table: string;
  column: string;
  type: string;
}

export interface HIR {
  base_table: string;
  mode: CompileMode;
  joins: Join[];
  projections: Projection[];
}

export interface CompileResult {
  ok: boolean;
  sql: string;
  errors: DiagnosticError[];
  warnings: DiagnosticWarning[];
  hir: HIR;
}

export interface VerifyResult {
  ok: boolean;
  message: string;
  plan_hint?: string | null;
}

export interface ExecuteColumn {
  name: string;
  type: string;
}

export interface ExecuteResult {
  ok: boolean;
  columns?: ExecuteColumn[];
  rows?: Record<string, unknown>[];
  elapsed_ms?: number;
  message?: string;
}

export interface Field {
  id: string;
  expr: string;
  alias: string;
  finalTable: string;
  type: string;
  pathLabel: string;
  indexedPath: boolean;
}

// Raw shapes returned by the WASM API (JSON-parsed).
export interface WasmError {
  message: string;
  stage?: string;
  code?: string;
  hint?: string;
  position: number;
}

export interface WasmHirJoin {
  path?: string[];
  join_column?: string;
  from_table?: string;
  to_table?: string;
}

export interface WasmHirField {
  alias: string;
  type?: string;
}

export interface WasmHirExpr {
  type?: string;
}

export interface WasmHir {
  joins?: WasmHirJoin[];
  fields?: WasmHirField[];
  expr?: WasmHirExpr;
  warnings?: Array<string | { code?: string; message: string }>;
}

export interface WasmCompilation {
  sql?: { query?: string };
  hir?: WasmHir;
}

export interface WasmCompileResult {
  ok: boolean;
  compilation?: WasmCompilation;
  error?: WasmError;
}

export interface WasmCompleteItem {
  kind: number;
  label: string;
  detail?: string;
  indexed?: boolean;
}

export interface WasmCompleteResult {
  ok: boolean;
  items?: WasmCompleteItem[];
}

export interface WasmSchemaTable {
  name: string;
  columns?: Array<{ name: string; type: string }>;
}

export interface WasmSchemaRelationship {
  from_table: string;
  name: string;
  to_table: string;
  join_column_indexed?: boolean;
}

export interface WasmSchemaInfo {
  base_table?: string;
  tables?: WasmSchemaTable[];
  relationships?: WasmSchemaRelationship[];
}

// Globals set by wasm_exec.js and the FormQL WASM binary.
declare global {
  interface Window {
    Go: new () => {
      importObject: WebAssembly.Imports;
      run(instance: WebAssembly.Instance): Promise<void>;
    };
    FormQL: {
      compileCatalogJSON(catalog: unknown, formula: string, options: object): WasmCompileResult;
      compileDocumentCatalogJSON(catalog: unknown, src: string, options: object): WasmCompileResult;
      completeCatalogJSON(catalog: unknown, src: string, cursor: number, options: object): WasmCompleteResult;
      loadSchemaInfoJSON(catalog: unknown, options: object): { ok: boolean; info?: WasmSchemaInfo; error?: WasmError };
    };
  }
}
