import React from 'react';
import { formql } from './realData';
import type { CompileMode, StatusState } from './types';

interface TopBarProps {
  baseTable: string;
  onBaseTable: (t: string) => void;
  mode: CompileMode;
  onMode: (m: CompileMode) => void;
  maxDepth: number;
  onMaxDepth: (d: number) => void;
  status: StatusState;
  statusMsg: string;
  onToggleRail: () => void;
  onOpenPresets: () => void;
}

export default function TopBar({
  baseTable, onBaseTable, mode, onMode, maxDepth, onMaxDepth,
  status, statusMsg, onToggleRail, onOpenPresets,
}: TopBarProps): React.ReactElement {
  const tables = formql.TABLES.map((t) => t.name);
  return (
    <header className="tb">
      <div className="tb-left">
        <button className="tb-rail-toggle" onClick={onToggleRail} title="Toggle schema">≡</button>
        <div className="tb-brand">
          <div className="tb-mark">ƒ</div>
          <div>
            <div className="tb-title">FormQL</div>
            <div className="tb-sub">playground</div>
          </div>
        </div>
        <div className="tb-sep" />
        <label className="tb-field">
          <span className="tb-field-label">BASE</span>
          <select value={baseTable} onChange={(e) => onBaseTable(e.target.value)} className="tb-select">
            {tables.map((n) => <option key={n} value={n}>{n}</option>)}
          </select>
        </label>
        <div className="tb-seg">
          <button className={mode === 'formula' ? 'active' : ''} onClick={() => onMode('formula')} type="button">
            Formula
          </button>
          <button className={mode === 'document' ? 'active' : ''} onClick={() => onMode('document')} type="button">
            Report
          </button>
        </div>
        <label className="tb-field">
          <span className="tb-field-label">DEPTH</span>
          <input
            className="tb-num"
            type="number"
            min={1}
            max={100}
            value={maxDepth}
            onChange={(e) => onMaxDepth(Number(e.target.value) || 30)}
          />
        </label>
      </div>
      <div className="tb-right">
        <div className={'tb-status ' + status}>
          <span className="tb-dot" />
          <span className="tb-status-text">{statusMsg}</span>
        </div>
        <button className="tb-btn" onClick={onOpenPresets} type="button">Examples</button>
      </div>
    </header>
  );
}
