import React from 'react';
import { formql } from './realData';
import TopBar from './TopBar';
import SchemaRail from './SchemaRail';
import FormulaEditor from './Editor';
import OutputDrawer from './OutputDrawer';
import PresetsCards from './PresetsCards';
import type { CompileMode, Preset } from './types';

interface PlaygroundState {
  baseTable: string;
  mode: CompileMode;
  formula: string;
}

function usePlayground(initial: PlaygroundState) {
  const [baseTable, setBaseTable] = React.useState(initial.baseTable);
  const [mode, setMode] = React.useState<CompileMode>(initial.mode);
  const [formula, setFormula] = React.useState(initial.formula);
  const [maxDepth, setMaxDepth] = React.useState(30);
  const [contextTable, setContextTable] = React.useState(initial.baseTable);
  const [verify, setVerify] = React.useState<import('./types').VerifyResult | null>(null);
  const [exec, setExec] = React.useState<import('./types').ExecuteResult | null>(null);
  const [isVerifying, setIsVerifying] = React.useState(false);
  const [isExecuting, setIsExecuting] = React.useState(false);

  const compile = React.useMemo(
    () => formql.compile(baseTable, formula, mode),
    [baseTable, formula, mode],
  );

  React.useEffect(() => { setVerify(null); setExec(null); }, [compile.sql]);

  const runAll = React.useCallback(async () => {
    if (!compile.ok) return;
    setIsVerifying(true);
    setIsExecuting(true);
    const v = await formql.verify(compile.sql);
    setVerify(v);
    setIsVerifying(false);
    const e = await formql.execute(compile);
    setExec(e);
    setIsExecuting(false);
  }, [compile]);

  return {
    baseTable, setBaseTable, mode, setMode, formula, setFormula,
    maxDepth, setMaxDepth, contextTable, setContextTable,
    compile, verify, exec, isVerifying, isExecuting, runAll,
  };
}

export function statusFromCompile(c: import('./types').CompileResult | null): { state: string; msg: string } {
  if (!c) return { state: 'idle', msg: '—' };
  if (!c.ok) return { state: 'err', msg: c.errors.length + ' error' + (c.errors.length === 1 ? '' : 's') };
  if (c.warnings.length) return { state: 'warn', msg: c.warnings.length + ' warning' + (c.warnings.length === 1 ? '' : 's') };
  return { state: 'ok', msg: 'compiled · ' + c.hir.joins.length + ' join' + (c.hir.joins.length === 1 ? '' : 's') };
}

interface IDEArtboardProps {
  showRaw: boolean;
  onMode: (m: CompileMode) => void;
}

export default function IDEArtboard({ showRaw, onMode }: IDEArtboardProps): React.ReactElement {
  const pg = usePlayground({
    baseTable: 'rental_contract',
    mode: 'formula',
    formula: 'customer_id__rel.email & " / " & STRING(actual_total)',
  });

  const [railCollapsed, setRailCollapsed] = React.useState(false);
  const [railQuery, setRailQuery] = React.useState('');
  const [presetsOpen, setPresetsOpen] = React.useState(false);

  const status = statusFromCompile(pg.compile);

  function insertAt(text: string) {
    pg.setFormula((s) => s + text);
  }

  function pickPreset(p: Preset) {
    pg.setBaseTable(p.baseTable);
    pg.setFormula(p.formula);
    if (p.mode !== 'formula') onMode(p.mode);
    setPresetsOpen(false);
  }

  function handleMode(m: CompileMode) {
    pg.setMode(m);
    if (m !== 'formula') onMode(m);
  }

  return (
    <div className="ab">
      <TopBar
        baseTable={pg.baseTable}
        onBaseTable={pg.setBaseTable}
        mode={pg.mode}
        onMode={handleMode}
        maxDepth={pg.maxDepth}
        onMaxDepth={pg.setMaxDepth}
        status={status.state as import('./types').StatusState}
        statusMsg={status.msg}
        onToggleRail={() => setRailCollapsed((v) => !v)}
        onOpenPresets={() => setPresetsOpen(true)}
      />
      <div className="presets-strip">
        <div className="ps-label">QUICK START</div>
        <div className="ps-cards">
          {formql.PRESETS.slice(0, 4).map((p) => (
            <button key={p.id} className="ps-card" onClick={() => pickPreset(p)} type="button">
              <div className="ps-card-row">
                <span className="ps-card-title">{p.title}</span>
                <span className={'ps-card-mode ' + p.mode}>{p.mode === 'document' ? 'report' : 'formula'}</span>
              </div>
              <div className="ps-card-desc">{p.description}</div>
              <div className="ps-card-foot"><span className="ps-chip">{p.baseTable}</span></div>
            </button>
          ))}
          <button className="ps-more" onClick={() => setPresetsOpen(true)} type="button">All examples →</button>
        </div>
      </div>
      <div className="ide-main">
        <SchemaRail
          baseTable={pg.baseTable}
          contextTable={pg.contextTable}
          onInsert={insertAt}
          query={railQuery}
          onQuery={setRailQuery}
          collapsed={railCollapsed}
          onToggleCollapsed={() => setRailCollapsed((v) => !v)}
        />
        <div className="ide-center">
          <div className="editor-frame">
            <div className="ef-head">
              <div className="ef-tab active"><span className="ef-icon">ƒ</span>formula</div>
              <div className="ef-actions">
                <span className={'ef-status ' + status.state}>
                  <span className="ef-dot" />
                  {status.msg}
                </span>
                <span className="ef-kbd"><kbd>⌃</kbd><kbd>Space</kbd> complete</span>
              </div>
            </div>
            <div className="ef-meter">
              <div className={'ef-meter-bar ' + status.state} />
            </div>
            <FormulaEditor
              value={pg.formula}
              onChange={pg.setFormula}
              baseTable={pg.baseTable}
              mode={pg.mode}
              errors={pg.compile.errors}
              warnings={pg.compile.warnings}
              onSchemaContextChange={pg.setContextTable}
              height={220}
              ariaLabel="FormQL formula"
            />
            <div className="ef-foot">
              <div className="ef-context">
                <span className="ef-ctx-k">cursor in</span>
                <span className="ef-ctx-v mono">{pg.contextTable}</span>
              </div>
              {pg.compile.ok && (
                <div className="ef-typeinfo">
                  <span className="ef-ti-k">result type</span>
                  <span className="ef-ti-v">
                    {pg.compile.hir.projections.length === 0 ? '—'
                      : pg.compile.hir.projections.length === 1 ? pg.compile.hir.projections[0].type
                      : pg.compile.hir.projections.length + ' cols'}
                  </span>
                </div>
              )}
            </div>
          </div>
          <OutputDrawer
            compile={pg.compile}
            exec={pg.exec ?? undefined}
            verify={pg.verify ?? undefined}
            isCompiling={false}
            isVerifying={pg.isVerifying}
            isExecuting={pg.isExecuting}
            showRaw={showRaw}
            onRunAll={pg.runAll}
          />
        </div>
      </div>
      <PresetsCards open={presetsOpen} onClose={() => setPresetsOpen(false)} onPick={pickPreset} />
    </div>
  );
}
