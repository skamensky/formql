import React from 'react';
import type { CompileResult, VerifyResult, ExecuteResult, DiagnosticError, DiagnosticWarning } from './types';

interface StatusPillProps {
  state: string;
  label: string;
  value?: string;
  onClick: () => void;
  active: boolean;
}

function StatusPill({ state, label, value, onClick, active }: StatusPillProps): React.ReactElement {
  const cls = 'pill ' + state + (active ? ' active' : '');
  const ico = state === 'ok' ? '✓' : state === 'warn' ? '!' : state === 'err' ? '✕' : state === 'pending' ? '…' : '·';
  return (
    <button className={cls} onClick={onClick} type="button">
      <span className="pill-dot">{ico}</span>
      <span className="pill-label">{label}</span>
      {value !== undefined && <span className="pill-value">{value}</span>}
    </button>
  );
}

function SqlView({ sql }: { sql?: string }): React.ReactElement {
  if (!sql) return <div className="od-empty">No SQL — formula has errors.</div>;
  const KW = /\b(SELECT|FROM|LEFT JOIN|JOIN|ON|AS|WHERE|LIMIT|GROUP BY|ORDER BY|CAST|AND|OR)\b/g;
  const parts: { t: string; k?: string }[] = [];
  let i = 0;
  let m: RegExpExecArray | null;
  KW.lastIndex = 0;
  while ((m = KW.exec(sql)) !== null) {
    if (m.index > i) parts.push({ t: sql.slice(i, m.index) });
    parts.push({ t: m[0], k: 'kw' });
    i = m.index + m[0].length;
  }
  if (i < sql.length) parts.push({ t: sql.slice(i) });
  return (
    <pre className="od-sql">
      {parts.map((p, ix) => <span key={ix} className={p.k ? 'sql-kw' : ''}>{p.t}</span>)}
    </pre>
  );
}

type DiagItem = (DiagnosticError | DiagnosticWarning) & { kind: 'err' | 'warn' };

function DiagnosticsView({ errors, warnings }: { errors: DiagnosticError[]; warnings: DiagnosticWarning[] }): React.ReactElement {
  const items: DiagItem[] = [
    ...errors.map((e) => ({ ...e, kind: 'err' as const })),
    ...warnings.map((w) => ({ ...w, kind: 'warn' as const })),
  ];
  if (items.length === 0) {
    return (
      <div className="od-empty ok">
        <span className="ok-tick">✓</span>No diagnostics. Compile clean.
      </div>
    );
  }
  return (
    <ul className="od-diag">
      {items.map((it, i) => (
        <li key={i} className={'diag ' + it.kind}>
          <span className="diag-icon">{it.kind === 'err' ? '✕' : '!'}</span>
          <div className="diag-body">
            <div className="diag-msg">{it.message}</div>
            {'hint' in it && it.hint && <div className="diag-hint">hint: {it.hint}</div>}
            {it.code && <div className="diag-code">{it.code}</div>}
            {'positioned' in it && it.positioned && (
              <div className="diag-loc">char {it.start}–{it.end}</div>
            )}
          </div>
        </li>
      ))}
    </ul>
  );
}

function HirView({ hir }: { hir?: CompileResult['hir'] }): React.ReactElement {
  if (!hir) return <div className="od-empty">Compile to see HIR.</div>;
  return (
    <div className="od-hir">
      <div className="hir-row"><div className="hir-k">BASE TABLE</div><div className="hir-v mono">{hir.base_table}</div></div>
      <div className="hir-row"><div className="hir-k">MODE</div><div className="hir-v">{hir.mode ?? 'formula'}</div></div>
      <div className="hir-row">
        <div className="hir-k">JOINS · {hir.joins?.length ?? 0}</div>
        <div className="hir-v">
          {(hir.joins ?? []).map((j, i) => (
            <div key={i} className={'hir-join ' + (j.indexed ? '' : 'unindexed')}>
              <span className="mono">{j.from}</span>
              <span className="hir-arrow">—{j.via}→</span>
              <span className="mono">{j.to}</span>
              {!j.indexed && <span className="hir-tag warn">unindexed</span>}
            </div>
          ))}
        </div>
      </div>
      <div className="hir-row">
        <div className="hir-k">PROJECTIONS · {hir.projections?.length ?? 0}</div>
        <div className="hir-v">
          {(hir.projections ?? []).map((p, i) => (
            <div key={i} className="hir-proj">
              <span className="mono">{p.path}</span>
              <span className="hir-tag">{p.type}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function ResultsTable({ exec }: { exec?: ExecuteResult }): React.ReactElement {
  if (!exec) return <div className="od-empty">Run to fetch rows.</div>;
  if (!exec.ok) return <div className="od-empty err">{exec.message}</div>;
  const cols = exec.columns ?? [];
  const rows = exec.rows ?? [];
  if (!rows.length) return <div className="od-empty">Query returned 0 rows.</div>;
  return (
    <div className="od-rows-wrap">
      <table className="od-rows">
        <thead>
          <tr>
            <th className="rn">#</th>
            {cols.map((c) => (
              <th key={c.name}>
                <div className="th-name">{c.name}</div>
                <div className="th-type">{c.type}</div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i}>
              <td className="rn">{i + 1}</td>
              {cols.map((c) => (
                <td key={c.name} className={'td ' + c.type}>{String(r[c.name] ?? '—')}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      <div className="od-rows-foot">
        {rows.length} row{rows.length === 1 ? '' : 's'} · {exec.elapsed_ms}ms
      </div>
    </div>
  );
}

interface OutputDrawerProps {
  compile?: CompileResult;
  exec?: ExecuteResult;
  verify?: VerifyResult;
  isCompiling?: boolean;
  isVerifying?: boolean;
  isExecuting?: boolean;
  showRaw?: boolean;
  onRunAll: () => void;
}

type TabKey = 'sql' | 'diag' | 'rows' | 'hir' | 'raw';

export default function OutputDrawer({
  compile, exec, verify, isCompiling, isVerifying, isExecuting, showRaw, onRunAll,
}: OutputDrawerProps): React.ReactElement {
  const [tab, setTab] = React.useState<TabKey>('sql');

  const compileState = !compile ? 'idle' : !compile.ok ? 'err' : compile.warnings.length ? 'warn' : 'ok';
  const verifyState = !verify ? 'idle' : isVerifying ? 'pending' : verify.ok ? 'ok' : 'err';
  const execState = !exec ? 'idle' : isExecuting ? 'pending' : exec.ok ? 'ok' : 'err';

  const compileLabel = isCompiling ? 'compiling…' : !compile ? 'compile'
    : !compile.ok ? compile.errors.length + ' error' + (compile.errors.length === 1 ? '' : 's')
    : compile.warnings.length ? compile.warnings.length + ' warning' + (compile.warnings.length === 1 ? '' : 's')
    : 'compiled';

  const verifyLabel = isVerifying ? 'verifying…' : !verify ? 'verify' : verify.ok ? 'verified' : 'verify failed';
  const execLabel = isExecuting ? 'running…' : !exec ? 'execute'
    : exec.ok ? (exec.rows?.length ?? 0) + ' row' + ((exec.rows?.length ?? 0) === 1 ? '' : 's')
    : 'exec failed';

  const tabs: [TabKey, string, number | null][] = [
    ['sql', 'SQL', null],
    ['diag', 'Diagnostics', (compile?.errors.length ?? 0) + (compile?.warnings.length ?? 0)],
    ['rows', 'Rows', exec?.ok ? (exec.rows?.length ?? 0) : null],
    ['hir', 'Compile', null],
  ];
  if (showRaw) tabs.push(['raw', 'Raw JSON', null]);

  return (
    <section className="od">
      <header className="od-head">
        <div className="od-pills">
          <StatusPill state={compileState} label={compileLabel} active={tab === 'diag' || tab === 'hir'} onClick={() => setTab(compileState === 'err' ? 'diag' : 'sql')} />
          <span className="pill-sep">·</span>
          <StatusPill state={verifyState} label={verifyLabel} active={false} onClick={() => setTab('sql')} />
          <span className="pill-sep">·</span>
          <StatusPill state={execState} label={execLabel} active={tab === 'rows'} onClick={() => setTab('rows')} />
        </div>
        <div className="od-tabs">
          {tabs.map(([k, label, badge]) => (
            <button key={k} className={'od-tab' + (tab === k ? ' active' : '')} onClick={() => setTab(k)} type="button">
              {label}
              {badge != null && <span className="od-tab-badge">{badge}</span>}
            </button>
          ))}
          <button className="od-run" onClick={onRunAll} type="button" disabled={!compile?.ok}>▸ Run All</button>
        </div>
      </header>
      <div className="od-body">
        {tab === 'sql' && <SqlView sql={compile?.sql} />}
        {tab === 'diag' && <DiagnosticsView errors={compile?.errors ?? []} warnings={compile?.warnings ?? []} />}
        {tab === 'hir' && <HirView hir={compile?.hir} />}
        {tab === 'rows' && <ResultsTable exec={exec} />}
        {tab === 'raw' && <pre className="od-raw">{JSON.stringify({ compile, verify, execute: exec }, null, 2)}</pre>}
      </div>
    </section>
  );
}
