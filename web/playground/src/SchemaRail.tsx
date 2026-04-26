import React from 'react';
import { formql } from './realData';
import type { Table, Relationship } from './types';

interface TableRowProps {
  t: Table;
  expanded: boolean;
  onToggle: () => void;
  onInsert: (text: string) => void;
  isCurrent: boolean;
  isBase: boolean;
}

function TableRow({ t, expanded, onToggle, onInsert, isCurrent, isBase }: TableRowProps): React.ReactElement {
  const rels: Relationship[] = formql.relsFrom(t.name);
  return (
    <div className={'sr-table' + (isCurrent ? ' current' : '') + (isBase ? ' base' : '')}>
      <button className="sr-table-head" onClick={onToggle} type="button">
        <span className="sr-caret">{expanded ? '▾' : '▸'}</span>
        <span className="sr-table-name">{t.name}</span>
        <span className="sr-meta">{t.columns.length}c · {rels.length}r</span>
        {isCurrent && <span className="sr-pin">context</span>}
        {isBase && !isCurrent && <span className="sr-pin base">base</span>}
      </button>
      {expanded && (
        <div className="sr-body">
          <div className="sr-section-label">Columns</div>
          {t.columns.map((c) => (
            <button key={c.name} className="sr-row col" onClick={() => onInsert(c.name)} title={'Insert ' + c.name} type="button">
              <span className="sr-icon col">C</span>
              <span className="sr-label">{c.name}</span>
              <span className="sr-type">{c.type}</span>
            </button>
          ))}
          {rels.length > 0 && <div className="sr-section-label">Relationships</div>}
          {rels.map((r) => (
            <button
              key={r.name}
              className={'sr-row rel' + (r.indexed ? '' : ' unindexed')}
              onClick={() => onInsert(r.name + '.')}
              title={r.indexed ? 'Insert ' + r.name : 'Unindexed join — will produce a warning'}
              type="button"
            >
              <span className="sr-icon rel">→</span>
              <span className="sr-label">{r.name}</span>
              <span className="sr-type">{r.to_table}</span>
              {!r.indexed && <span className="sr-bang" title="unindexed">!</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

interface SchemaRailProps {
  baseTable: string;
  contextTable: string;
  onInsert: (text: string) => void;
  query?: string;
  onQuery: (q: string) => void;
  collapsed: boolean;
  onToggleCollapsed: () => void;
}

export default function SchemaRail({ baseTable, contextTable, onInsert, query = '', onQuery, collapsed, onToggleCollapsed }: SchemaRailProps): React.ReactElement {
  const [expanded, setExpanded] = React.useState<Set<string>>(() => new Set([contextTable, baseTable]));

  React.useEffect(() => {
    setExpanded((s) => {
      const n = new Set(s);
      n.add(contextTable);
      n.add(baseTable);
      return n;
    });
  }, [contextTable, baseTable]);

  const tables = React.useMemo(() => {
    const all = [...formql.TABLES].sort((a, b) => a.name.localeCompare(b.name));
    const ordered: Table[] = [];
    const ctx = all.find((t) => t.name === contextTable);
    if (ctx) ordered.push(ctx);
    if (baseTable !== contextTable) {
      const b = all.find((t) => t.name === baseTable);
      if (b) ordered.push(b);
    }
    for (const t of all) {
      if (t.name !== contextTable && t.name !== baseTable) ordered.push(t);
    }
    return ordered;
  }, [baseTable, contextTable]);

  const filtered = query
    ? tables.filter((t) =>
        t.name.includes(query) ||
        t.columns.some((c) => c.name.includes(query)) ||
        formql.relsFrom(t.name).some((r) => r.name.includes(query)))
    : tables;

  if (collapsed) {
    return (
      <aside className="sr collapsed">
        <button className="sr-collapse" onClick={onToggleCollapsed} title="Expand schema">›</button>
        <div className="sr-vlabel">SCHEMA</div>
      </aside>
    );
  }

  return (
    <aside className="sr">
      <div className="sr-head">
        <div className="sr-title">Schema</div>
        <div className="sr-sub">rental</div>
        <button className="sr-collapse" onClick={onToggleCollapsed} title="Collapse">‹</button>
      </div>
      <input
        className="sr-search"
        placeholder="Filter columns, tables, relationships…"
        value={query}
        onChange={(e) => onQuery(e.target.value)}
      />
      <div className="sr-list">
        {filtered.map((t) => (
          <TableRow
            key={t.name}
            t={t}
            expanded={expanded.has(t.name)}
            onToggle={() => setExpanded((s) => {
              const n = new Set(s);
              n.has(t.name) ? n.delete(t.name) : n.add(t.name);
              return n;
            })}
            onInsert={onInsert}
            isCurrent={t.name === contextTable}
            isBase={t.name === baseTable}
          />
        ))}
      </div>
    </aside>
  );
}
