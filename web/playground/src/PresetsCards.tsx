import React from 'react';
import { formql } from './realData';
import type { Preset } from './types';

interface PresetsCardsProps {
  open: boolean;
  onClose: () => void;
  onPick: (p: Preset) => void;
}

export default function PresetsCards({ open, onClose, onPick }: PresetsCardsProps): React.ReactElement | null {
  if (!open) return null;
  const presets = formql.PRESETS;
  return (
    <div className="pc-backdrop" onClick={onClose}>
      <div className="pc" onClick={(e) => e.stopPropagation()}>
        <div className="pc-head">
          <h3>Examples</h3>
          <p>Pick one to load — sets base table, mode, and formula.</p>
          <button className="pc-close" onClick={onClose} type="button">✕</button>
        </div>
        <div className="pc-grid">
          {presets.map((p) => (
            <button key={p.id} className="pc-card" onClick={() => onPick(p)} type="button">
              <div className="pc-card-head">
                <div className="pc-card-title">{p.title}</div>
                <div className={'pc-card-mode ' + p.mode}>{p.mode === 'document' ? 'Report' : 'Formula'}</div>
              </div>
              <div className="pc-card-desc">{p.description}</div>
              <div className="pc-card-foot">
                <span className="pc-chip">
                  <span className="pc-chip-k">base</span>
                  <span className="pc-chip-v">{p.baseTable}</span>
                </span>
              </div>
              <pre className="pc-card-formula">{p.formula}</pre>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
