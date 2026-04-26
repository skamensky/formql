import React from 'react';
import { formql, fieldsToDocument } from './realData';
import TopBar from './TopBar';
import ReportBuilder from './ReportBuilder';
import PresetsCards from './PresetsCards';
import { statusFromCompile } from './IDEArtboard';
import type { CompileMode, Field, Preset } from './types';

function buildInitialFields(): Field[] {
  const reachable = new Map<string, typeof formql.RELATIONSHIPS[0][]>();
  reachable.set('rental_contract', []);
  const queue: { table: string; path: typeof formql.RELATIONSHIPS[0][] }[] = [{ table: 'rental_contract', path: [] }];
  while (queue.length) {
    const item = queue.shift()!;
    if (item.path.length >= 3) continue;
    for (const r of formql.relsFrom(item.table)) {
      if (!reachable.has(r.to_table)) {
        const np = [...item.path, r];
        reachable.set(r.to_table, np);
        queue.push({ table: r.to_table, path: np });
      }
    }
  }

  function makeField(table: string, col: string): Field | null {
    const t = formql.tableByName(table);
    if (!t) return null;
    const c = t.columns.find((x) => x.name === col);
    if (!c) return null;
    const path = reachable.get(table) ?? [];
    const traversal = path.map((r) => r.name).concat([col]).join('.');
    const alias = path.length === 0 ? col : path[path.length - 1].name.replace(/__rel$/, '') + '_' + col;
    return {
      id: traversal, expr: traversal, alias,
      finalTable: table, type: c.type,
      pathLabel: path.map((r) => r.name.replace(/__rel$/, '')).concat([col]).join(' › '),
      indexedPath: path.every((r) => r.indexed),
    };
  }

  return [
    makeField('rental_contract', 'actual_total'),
    makeField('customer', 'email'),
    makeField('vehicle', 'model_name'),
    makeField('vehicle', 'model_year'),
  ].filter((f): f is Field => f !== null);
}

interface ReportArtboardProps {
  onMode: (m: CompileMode) => void;
}

export default function ReportArtboard({ onMode }: ReportArtboardProps): React.ReactElement {
  const [baseTable, setBaseTable] = React.useState('rental_contract');
  const [maxDepth, setMaxDepth] = React.useState(30);
  const [presetsOpen, setPresetsOpen] = React.useState(false);
  const [fields, setFields] = React.useState<Field[]>(buildInitialFields);
  const [exec, setExec] = React.useState<import('./types').ExecuteResult | null>(null);
  const [verify, setVerify] = React.useState<import('./types').VerifyResult | null>(null);
  const [isExecuting, setIsExecuting] = React.useState(false);
  const [sqlOpen, setSqlOpen] = React.useState(false);

  const formula = React.useMemo(() => fieldsToDocument(fields), [fields]);
  const compile = React.useMemo(() => formql.compile(baseTable, formula, 'document'), [baseTable, formula]);

  React.useEffect(() => { setExec(null); setVerify(null); }, [compile.sql]);

  async function runAll() {
    if (!compile.ok) return;
    setIsExecuting(true);
    const v = await formql.verify(compile.sql);
    setVerify(v);
    const e = await formql.execute(compile);
    setExec(e);
    setIsExecuting(false);
  }

  function pickPreset(p: Preset) {
    setBaseTable(p.baseTable);
    if (p.mode !== 'document') {
      setFields([{
        id: 'result', expr: p.formula, alias: 'result',
        finalTable: p.baseTable, type: 'text',
        pathLabel: p.title, indexedPath: true,
      }]);
    } else {
      const parts = p.formula.split(',').map((s) => s.trim()).filter(Boolean);
      setFields(parts.map((part, i) => {
        const m = part.match(/^(.+?)\s+AS\s+(\w+)$/i);
        const expr = m ? m[1].trim() : part;
        const alias = m ? m[2] : 'f' + i;
        return { id: expr, expr, alias, finalTable: p.baseTable, type: 'text', pathLabel: expr, indexedPath: true };
      }));
    }
    setPresetsOpen(false);
  }

  const status = statusFromCompile(compile);

  return (
    <div className="ab">
      <TopBar
        baseTable={baseTable}
        onBaseTable={setBaseTable}
        mode="document"
        onMode={(m) => { if (m === 'formula') onMode(m); }}
        maxDepth={maxDepth}
        onMaxDepth={setMaxDepth}
        status={status.state as import('./types').StatusState}
        statusMsg={status.msg}
        onToggleRail={() => {}}
        onOpenPresets={() => setPresetsOpen(true)}
      />
      <ReportBuilder
        baseTable={baseTable}
        fields={fields}
        onFields={setFields}
        compile={compile}
        exec={exec ?? undefined}
        isExecuting={isExecuting}
        onRunAll={runAll}
      />
      {compile.ok && (
        <div className="rb-sql-panel">
          <button className="rb-sql-toggle" onClick={() => setSqlOpen((v) => !v)} type="button">
            <span className="rb-sql-toggle-caret">{sqlOpen ? '▾' : '▸'}</span>
            Generated SQL
            <span style={{ marginLeft: 'auto', color: 'var(--muted-2)' }}>{compile.sql.split('\n').length} lines</span>
          </button>
          {sqlOpen && (
            <div className="rb-sql-body">
              <pre className="rb-sql-pre">{compile.sql}</pre>
            </div>
          )}
        </div>
      )}
      <div className="rb-statusbar">
        <div className="rbsb-pills">
          <span className={'pill ' + status.state}>
            <span className="pill-dot">{status.state === 'ok' ? '✓' : status.state === 'warn' ? '!' : '✕'}</span>
            <span className="pill-label">
              {status.state === 'ok'
                ? 'compiled · ' + compile.hir.joins.length + ' join' + (compile.hir.joins.length === 1 ? '' : 's')
                : status.msg}
            </span>
          </span>
          <span className="pill-sep">·</span>
          <span className={'pill ' + (verify ? 'ok' : 'idle')}>
            <span className="pill-dot">{verify ? '✓' : '·'}</span>
            <span className="pill-label">{verify ? 'verified' : 'not verified'}</span>
          </span>
          <span className="pill-sep">·</span>
          <span className={'pill ' + (exec?.ok ? 'ok' : isExecuting ? 'pending' : 'idle')}>
            <span className="pill-dot">{exec?.ok ? '✓' : isExecuting ? '…' : '·'}</span>
            <span className="pill-label">
              {exec?.ok ? (exec.rows ?? []).length + ' rows · ' + exec.elapsed_ms + 'ms'
                : isExecuting ? 'running…' : 'not run'}
            </span>
          </span>
        </div>
        <div className="rbsb-right mono">
          {compile.ok
            ? compile.sql.split('\n').length + ' lines · ' + fields.length + ' field' + (fields.length === 1 ? '' : 's')
            : '—'}
        </div>
      </div>
      <PresetsCards open={presetsOpen} onClose={() => setPresetsOpen(false)} onPick={pickPreset} />
    </div>
  );
}
