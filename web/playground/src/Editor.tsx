import React from 'react';
import { formql } from './realData';
import type { Token, CompleteItem, DiagnosticError, DiagnosticWarning } from './types';

function syntaxRender(src: string): React.ReactNode[] {
  const tokens: Token[] = formql.tokenize(src);
  return tokens.map((tk, i) => (
    <span key={i} className={'tk-' + tk.kind}>{tk.text}</span>
  ));
}

interface Segment { text: string; classes: string[] }

function squigglesRender(src: string, errors: DiagnosticError[], warnings: DiagnosticWarning[]): React.ReactNode | null {
  const all = [
    ...errors.map((e) => ({ ...e, kind: 'err' })),
    ...warnings.map((w) => ({ ...w, kind: 'warn' })),
  ].filter((r) => r.start < r.end);
  if (!all.length) return null;

  const bps = new Set<number>([0, src.length]);
  for (const r of all) { bps.add(r.start); bps.add(r.end); }
  const points = [...bps].sort((a, b) => a - b);

  const segments: Segment[] = [];
  for (let i = 0; i < points.length - 1; i++) {
    const s = points[i], e = points[i + 1];
    if (s === e) continue;
    const text = src.slice(s, e);
    const classes: string[] = [];
    for (const r of all) {
      if (r.start <= s && r.end >= e && r.start < r.end) classes.push(r.kind);
    }
    segments.push({ text, classes });
  }
  return segments.map((seg, i) => (
    <span key={i} className={seg.classes.map((c) => 'sq-' + c).join(' ')}>{seg.text}</span>
  ));
}

interface FormulaEditorProps {
  value: string;
  onChange: (v: string) => void;
  baseTable: string;
  mode: string;
  errors?: DiagnosticError[];
  warnings?: DiagnosticWarning[];
  onSchemaContextChange?: (table: string) => void;
  onCursorChange?: (pos: number) => void;
  ariaLabel?: string;
  height?: number;
}

export default function FormulaEditor({
  value, onChange, baseTable,
  errors = [], warnings = [],
  onSchemaContextChange, onCursorChange, ariaLabel, height = 220,
}: FormulaEditorProps): React.ReactElement {
  const taRef = React.useRef<HTMLTextAreaElement>(null);
  const overlayRef = React.useRef<HTMLDivElement>(null);
  const squigRef = React.useRef<HTMLDivElement>(null);
  const [completion, setCompletion] = React.useState<{ items: CompleteItem[]; x: number; y: number } | null>(null);
  const [acIndex, setAcIndex] = React.useState(0);

  const syncScroll = React.useCallback(() => {
    const ta = taRef.current;
    if (!ta) return;
    if (overlayRef.current) { overlayRef.current.scrollTop = ta.scrollTop; overlayRef.current.scrollLeft = ta.scrollLeft; }
    if (squigRef.current) { squigRef.current.scrollTop = ta.scrollTop; squigRef.current.scrollLeft = ta.scrollLeft; }
  }, []);

  const recompute = React.useCallback((force = false) => {
    const ta = taRef.current;
    if (!ta) return;
    const cursor = ta.selectionStart;
    const before = value.slice(0, cursor);
    const tail = before.match(/[A-Za-z0-9_.]*$/)?.[0] ?? '';
    onCursorChange?.(cursor);
    onSchemaContextChange?.(formql.contextTable(baseTable, value, cursor));
    if (!force && tail.length === 0) { setCompletion(null); return; }
    const r = formql.complete(baseTable, value, cursor);
    if (!r.items.length) { setCompletion(null); return; }
    const pos = caretPos(ta, cursor);
    setCompletion({ items: r.items.slice(0, 12), x: pos.left, y: pos.top + pos.height });
    setAcIndex(0);
  }, [value, baseTable, onCursorChange, onSchemaContextChange]);

  React.useEffect(() => { recompute(); }, [value, baseTable]);

  function applyItem(item: CompleteItem) {
    const ta = taRef.current;
    if (!ta || !item) return;
    const cursor = ta.selectionStart;
    let start = cursor;
    while (start > 0 && /[A-Za-z0-9_]/.test(value[start - 1])) start--;
    let end = cursor;
    while (end < value.length && /[A-Za-z0-9_]/.test(value[end])) end++;
    const suffix = item.kind === 'function' ? '(' : item.kind === 'relationship' ? '.' : '';
    const next = value.slice(0, start) + item.label + suffix + value.slice(end);
    const cursor2 = start + item.label.length + suffix.length;
    onChange(next);
    requestAnimationFrame(() => {
      const ta2 = taRef.current;
      if (!ta2) return;
      ta2.focus();
      ta2.setSelectionRange(cursor2, cursor2);
    });
    setCompletion(null);
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (completion) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setAcIndex((i) => (i + 1) % completion.items.length); return; }
      if (e.key === 'ArrowUp') { e.preventDefault(); setAcIndex((i) => (i - 1 + completion.items.length) % completion.items.length); return; }
      if (e.key === 'Enter' || e.key === 'Tab') { e.preventDefault(); applyItem(completion.items[acIndex]); return; }
      if (e.key === 'Escape') { e.preventDefault(); setCompletion(null); return; }
    }
    if (e.key === ' ' && (e.ctrlKey || e.metaKey)) { e.preventDefault(); recompute(true); }
  }

  return (
    <div className="fq-editor" style={{ height }}>
      <div className="fq-editor-overlay tk" ref={overlayRef}>
        <div className="fq-editor-text">{syntaxRender(value || '')}</div>
        <div className="fq-editor-text" style={{ color: 'transparent' }}>{' '}</div>
      </div>
      <div className="fq-editor-overlay sq" ref={squigRef} aria-hidden="true">
        <div className="fq-editor-text">{squigglesRender(value || '', errors, warnings)}</div>
      </div>
      <textarea
        ref={taRef}
        className="fq-editor-input"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onScroll={syncScroll}
        onKeyDown={onKeyDown}
        onClick={() => recompute()}
        onKeyUp={(e) => { if (['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) recompute(); }}
        onBlur={() => setTimeout(() => setCompletion(null), 120)}
        spellCheck={false}
        autoCapitalize="off"
        autoCorrect="off"
        aria-label={ariaLabel ?? 'Formula editor'}
      />
      {completion && (
        <div className="fq-ac" style={{ left: completion.x, top: completion.y }}>
          {completion.items.map((it, i) => (
            <button
              key={it.label + i}
              className={'fq-ac-item' + (i === acIndex ? ' active' : '')}
              onMouseDown={(ev) => { ev.preventDefault(); applyItem(it); }}
              onMouseEnter={() => setAcIndex(i)}
              type="button"
            >
              <span className={'fq-ac-icon ico-' + it.kind}>
                {it.kind === 'column' ? 'C' : it.kind === 'relationship' ? '→' : 'ƒ'}
              </span>
              <span className="fq-ac-label">{it.label}</span>
              <span className="fq-ac-detail">{it.detail}</span>
              {it.kind === 'relationship' && !it.indexed && <span className="fq-ac-warn" title="Unindexed join">!</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function caretPos(ta: HTMLTextAreaElement, pos: number): { left: number; top: number; height: number } {
  const styles = getComputedStyle(ta);
  const div = document.createElement('div');
  const props = ['boxSizing', 'width', 'height', 'padding', 'border', 'fontFamily', 'fontSize', 'fontWeight', 'lineHeight', 'letterSpacing', 'whiteSpace', 'wordWrap', 'tabSize'];
  for (const p of props) (div.style as unknown as Record<string, string>)[p] = (styles as unknown as Record<string, string>)[p];
  div.style.position = 'absolute';
  div.style.top = '0';
  div.style.left = '0';
  div.style.visibility = 'hidden';
  div.style.whiteSpace = 'pre-wrap';
  div.style.wordWrap = 'break-word';
  div.style.overflow = 'hidden';
  div.textContent = ta.value.substring(0, pos);
  const span = document.createElement('span');
  span.textContent = ta.value.substring(pos) || '.';
  div.appendChild(span);
  ta.parentNode!.appendChild(div);
  const dr = div.getBoundingClientRect();
  const sr = span.getBoundingClientRect();
  const x = sr.left - dr.left - ta.scrollLeft;
  const y = sr.top - dr.top - ta.scrollTop;
  div.remove();
  const lh = parseFloat(styles.lineHeight) || 18;
  return { left: x + 4, top: y + lh, height: lh };
}
