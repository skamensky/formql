import React from 'react';
import { formql } from './realData';
import FormulaEditor from './Editor';
import type { Field, CompileResult, ExecuteResult, Relationship, Column } from './types';

function isCustomExpr(expr: string): boolean {
  return /[^A-Za-z0-9_.]/.test(expr.trim());
}

function resolveNavTable(baseTable: string, navPath: Relationship[]): string {
  let table = baseTable;
  for (const rel of navPath) table = rel.to_table;
  return table;
}

// ─── Field expression edit modal ─────────────────────────────────
interface FieldEditModalProps {
  field: Field;
  baseTable: string;
  onSave: (f: Field) => void;
  onClose: () => void;
}

function FieldEditModal({ field, baseTable, onSave, onClose }: FieldEditModalProps): React.ReactElement {
  const [expr, setExpr] = React.useState(field.expr);
  const [alias, setAlias] = React.useState(field.alias);
  const [contextTable, setContextTable] = React.useState(baseTable);

  const compile = React.useMemo(
    () => formql.compile(baseTable, expr, 'formula'),
    [baseTable, expr],
  );
  const status = compile.ok ? 'ok' : expr.trim() ? 'err' : 'idle';

  function handleSave() {
    if (!expr.trim() || !compile.ok) return;
    onSave({ ...field, expr: expr.trim(), alias: alias || field.alias });
  }

  return (
    <div className="fe-backdrop" onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="fe-box">
        <div className="fe-head">
          <div className="fe-head-title">
            <span className="fe-head-label">Edit field</span>
            <span className="fe-head-alias">{field.alias}</span>
          </div>
          <button className="fe-close" onClick={onClose} type="button">✕</button>
        </div>
        <div className="fe-alias-row">
          <label className="fe-alias-label">Alias</label>
          <input className="fe-alias-input" value={alias} onChange={(e) => setAlias(e.target.value)} spellCheck={false} />
        </div>
        <div className="fe-editor-wrap">
          <div className="fe-editor-label-row">
            <span className="fe-editor-label">Expression</span>
            <span className={'fe-status ' + status}>
              <span className="fe-status-dot" />
              {status === 'ok'
                ? (compile.hir.projections[0]?.type ?? 'ok')
                : status === 'err'
                ? (compile.errors[0]?.message ?? 'error')
                : '—'}
            </span>
          </div>
          <FormulaEditor
            value={expr}
            onChange={setExpr}
            baseTable={baseTable}
            mode="formula"
            errors={compile.errors}
            warnings={compile.warnings}
            onSchemaContextChange={setContextTable}
            height={120}
            ariaLabel="Field expression"
          />
        </div>
        <div className="fe-foot">
          <span className="fe-foot-ctx">cursor in <strong>{contextTable}</strong></span>
          <div className="fe-foot-actions">
            <button className="fe-btn-cancel" onClick={onClose} type="button">Cancel</button>
            <button className="fe-btn-save" onClick={handleSave} type="button" disabled={!expr.trim() || !compile.ok}>Save</button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Drill-down relationship navigator ───────────────────────────
interface PickerNavProps {
  baseTable: string;
  fields: Field[];
  onAddField: (f: Field) => void;
}

function PickerNav({ baseTable, fields, onAddField }: PickerNavProps): React.ReactElement {
  const [navPath, setNavPath] = React.useState<Relationship[]>([]);
  const [colFilter, setColFilter] = React.useState('');
  React.useEffect(() => { setNavPath([]); setColFilter(''); }, [baseTable]);

  const currentTableName = resolveNavTable(baseTable, navPath);
  const currentTable = formql.tableByName(currentTableName);
  const rels = formql.relsFrom(currentTableName);

  function exprForCol(colName: string): string {
    return navPath.map((r) => r.name).concat([colName]).join('.');
  }

  function addCol(col: Column) {
    const expr = exprForCol(col.name);
    const lastRel = navPath[navPath.length - 1];
    const alias = navPath.length === 0
      ? col.name
      : lastRel.name.replace(/__rel$/, '') + '_' + col.name;
    onAddField({
      id: expr, expr, alias,
      finalTable: currentTableName,
      type: col.type,
      pathLabel: navPath.map((r) => r.name.replace(/__rel$/, '')).concat([col.name]).join(' › '),
      indexedPath: navPath.every((r) => r.indexed),
    });
  }

  const filteredCols = colFilter
    ? (currentTable?.columns ?? []).filter((c) => c.name.toLowerCase().includes(colFilter.toLowerCase()))
    : (currentTable?.columns ?? []);

  return (
    <div className="rb-nav">
      <div className="rb-nav-crumb">
        {navPath.length > 0 && (
          <button className="rb-nav-back" onClick={() => setNavPath(navPath.slice(0, -1))} type="button">←</button>
        )}
        <div className="rb-nav-crumb-path">
          <button className="rb-nav-crumb-seg" onClick={() => setNavPath([])} type="button">{baseTable}</button>
          {navPath.map((r, i) => (
            <React.Fragment key={i}>
              <span className="rb-nav-crumb-sep">›</span>
              <button
                className={'rb-nav-crumb-seg' + (i === navPath.length - 1 ? ' active' : '')}
                onClick={() => setNavPath(navPath.slice(0, i + 1))}
                type="button"
              >
                {r.name.replace(/__rel$/, '')}
              </button>
            </React.Fragment>
          ))}
        </div>
      </div>
      <input
        className="rb-picker-search rb-nav-filter"
        placeholder="Filter columns…"
        value={colFilter}
        onChange={(e) => setColFilter(e.target.value)}
      />
      <div className="rb-nav-list">
        {filteredCols.length > 0 && (
          <>
            <div className="rb-nav-section">Columns</div>
            {filteredCols.map((col) => {
              const expr = exprForCol(col.name);
              const added = !!fields.find((f) => f.id === expr);
              return (
                <button key={col.name} className={'rb-pick' + (added ? ' added' : '')} onClick={() => addCol(col)} type="button" disabled={added}>
                  <span className="rb-pick-icon">C</span>
                  <span className="rb-pick-name">{col.name}</span>
                  <span className="rb-pick-type">{col.type}</span>
                  <span className="rb-pick-add">{added ? '✓' : '+'}</span>
                </button>
              );
            })}
          </>
        )}
        {!colFilter && rels.length > 0 && (
          <>
            <div className="rb-nav-section">Relationships</div>
            {rels.map((rel) => (
              <button
                key={rel.name}
                className="rb-nav-rel"
                onClick={() => { setNavPath([...navPath, rel]); setColFilter(''); }}
                type="button"
              >
                <span className="rb-pick-icon rel">R</span>
                <span className="rb-nav-rel-name">{rel.name.replace(/__rel$/, '')}</span>
                <span className="rb-nav-rel-to">{rel.to_table}</span>
                <span className={'rb-nav-rel-arr' + (!rel.indexed ? ' warn' : '')}>›</span>
              </button>
            ))}
          </>
        )}
      </div>
    </div>
  );
}

// ─── Main ReportBuilder ──────────────────────────────────────────
interface ReportBuilderProps {
  baseTable: string;
  fields: Field[];
  onFields: (f: Field[]) => void;
  compile: CompileResult;
  exec?: ExecuteResult;
  isExecuting: boolean;
  onRunAll: () => void;
}

export default function ReportBuilder({ baseTable, fields, onFields, compile, exec, isExecuting, onRunAll }: ReportBuilderProps): React.ReactElement {
  const [editingField, setEditingField] = React.useState<Field | null>(null);

  function removeField(id: string) {
    const f = fields.find((x) => x.id === id);
    if (f && isCustomExpr(f.expr)) {
      if (!window.confirm('This field has a custom expression. Remove anyway?')) return;
    }
    onFields(fields.filter((x) => x.id !== id));
  }

  function clearAll() {
    if (fields.some((f) => isCustomExpr(f.expr))) {
      if (!window.confirm('Some fields have custom expressions. Clear all anyway?')) return;
    }
    onFields([]);
  }

  function moveField(id: string, dir: number) {
    const idx = fields.findIndex((f) => f.id === id);
    if (idx < 0) return;
    const next = [...fields];
    const j = idx + dir;
    if (j < 0 || j >= next.length) return;
    [next[idx], next[j]] = [next[j], next[idx]];
    onFields(next);
  }

  function renameField(id: string, alias: string) {
    onFields(fields.map((f) => f.id === id ? { ...f, alias } : f));
  }

  function saveFieldEdit(updated: Field) {
    onFields(fields.map((f) => f.id === updated.id ? updated : f));
    setEditingField(null);
  }

  function addField(f: Field) {
    if (fields.find((x) => x.id === f.id)) return;
    onFields([...fields, f]);
  }

  const previewRows = exec?.ok ? (exec.rows ?? []) : generatePreview(fields);

  return (
    <div className="rb">
      {editingField && (
        <FieldEditModal field={editingField} baseTable={baseTable} onSave={saveFieldEdit} onClose={() => setEditingField(null)} />
      )}
      <div className="rb-fields">
        <div className="rb-fields-label">REPORT FIELDS</div>
        {fields.length === 0 && (
          <div className="rb-empty-fields">Navigate relationships on the right → click columns to add fields</div>
        )}
        <div className="rb-chips">
          {fields.map((f, i) => (
            <div key={f.id} className={'rb-chip' + (f.indexedPath ? '' : ' unindexed')} title={f.expr}>
              <div className="rb-chip-bar">
                <span className="rb-chip-i">{i + 1}</span>
                <button className="rb-chip-mv" onClick={() => moveField(f.id, -1)} title="Move left" type="button">‹</button>
                <button className="rb-chip-mv" onClick={() => moveField(f.id, 1)} title="Move right" type="button">›</button>
                <button className="rb-chip-edit" onClick={() => setEditingField(f)} title="Edit expression" type="button">✎</button>
                <button className="rb-chip-x" onClick={() => removeField(f.id)} title="Remove" type="button">✕</button>
              </div>
              <input className="rb-chip-alias" value={f.alias} onChange={(e) => renameField(f.id, e.target.value)} spellCheck={false} />
              <div className="rb-chip-path">
                {isCustomExpr(f.expr) && <span className="rb-chip-custom">custom </span>}
                {f.pathLabel}
              </div>
              <div className="rb-chip-foot">
                <span className="rb-chip-type">{f.type}</span>
                {!f.indexedPath && <span className="rb-chip-flag">unindexed</span>}
              </div>
            </div>
          ))}
        </div>
        {fields.length > 0 && <button className="rb-clear" onClick={clearAll} type="button">Clear all</button>}
      </div>
      <div className="rb-body">
        <section className="rb-preview">
          <div className="rb-preview-head">
            <div className="rb-preview-title">Preview</div>
            <div className="rb-preview-meta">
              {exec?.ok
                ? `${(exec.rows ?? []).length} row${(exec.rows ?? []).length === 1 ? '' : 's'} · ${exec.elapsed_ms}ms · live`
                : `${fields.length} field${fields.length === 1 ? '' : 's'} · sample`}
            </div>
            <button className="rb-run" onClick={onRunAll} type="button" disabled={isExecuting || !compile?.ok || fields.length === 0}>
              {isExecuting ? 'running…' : '▸ Run'}
            </button>
          </div>
          {fields.length === 0
            ? (
              <div className="rb-preview-empty">
                <div className="rb-preview-empty-mark">▦</div>
                <div>Add fields to compose your report</div>
              </div>
            ) : (
              <div className="rb-rows-wrap">
                <table className="rb-rows">
                  <thead>
                    <tr>
                      <th className="rn">#</th>
                      {fields.map((f) => (
                        <th key={f.id}>
                          <div className="th-name">{f.alias}</div>
                          <div className="th-type">{f.type}</div>
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {previewRows.map((r, ri) => (
                      <tr key={ri}>
                        <td className="rn">{ri + 1}</td>
                        {fields.map((f) => (
                          <td key={f.id}>{formatVal(r[f.alias] ?? r[f.id] ?? samplePreviewVal(f, ri))}</td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
        </section>
        <section className="rb-picker">
          <div className="rb-picker-head">
            <div className="rb-picker-title">Add fields</div>
          </div>
          <PickerNav baseTable={baseTable} fields={fields} onAddField={addField} />
        </section>
      </div>
    </div>
  );
}

function formatVal(v: unknown): string {
  if (v == null) return '—';
  return String(v);
}

function generatePreview(fields: Field[]): Record<string, unknown>[] {
  return Array.from({ length: 8 }, (_, i) => {
    const row: Record<string, unknown> = {};
    for (const f of fields) row[f.alias] = samplePreviewVal(f, i);
    return row;
  });
}

function samplePreviewVal(f: Field, i: number): unknown {
  const path = f.expr;
  const emails = ['maya@ex.com', 'jpark@ex.com', 'aisha@ex.com', 'leo.t@ex.com', 'sasha@ex.com', 'ben@ex.com', 'noor@ex.com', 'riley@ex.com'];
  const models = ['Toyota Corolla', 'Hyundai Elantra', 'Ford Escape', 'Honda Civic', 'Tesla Model 3', 'VW Jetta', 'Nissan Sentra', 'Kia Forte'];
  const branches = ['SFO Downtown', 'OAK Airport', 'SJC Central', 'LAX Terminal', 'SAN Marina', 'PDX Pearl', 'SEA Capitol', 'DEN Tech'];
  const reps = ['Rao', 'Webb', 'Zhao', 'Costa', 'Bauer', 'Saleh', 'Reyes', 'Klein'];
  if (path.endsWith('actual_total')) return (220 + i * 47.3).toFixed(2);
  if (path.endsWith('sale_price')) return (12400 + i * 1830).toFixed(2);
  if (path.endsWith('quoted_total')) return (180 + i * 39.2).toFixed(2);
  if (path.endsWith('email')) return emails[i % emails.length];
  if (path.endsWith('first_name')) return ['Priya', 'Tom', 'Lin', 'Marco', 'Eva', 'Omar', 'Talia', 'Hugo'][i % 8];
  if (path.endsWith('last_name')) return reps[i % reps.length];
  if (path.endsWith('model_name')) return models[i % models.length];
  if (path.endsWith('model_year')) return 2020 + (i % 5);
  if (path.endsWith('name')) return branches[i % branches.length];
  if (f.type === 'numeric') return (100 + i * 17.3).toFixed(2);
  if (f.type === 'int4') return 1000 + i;
  return '—';
}
