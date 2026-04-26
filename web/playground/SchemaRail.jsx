/* SchemaRail.jsx — collapsible left rail with a tree view of tables.
   The "current context" table (based on cursor) is pinned + highlighted.
   Clicking a column or relationship inserts it at the cursor. */

const FQ_S = window.FORMQL;

function TableRow({ t, expanded, onToggle, onInsert, isCurrent, isBase }) {
  const rels = FQ_S.relsFrom(t.name);
  return React.createElement(
    "div",
    { className: "sr-table" + (isCurrent ? " current" : "") + (isBase ? " base" : "") },
    React.createElement(
      "button",
      { className: "sr-table-head", onClick: onToggle, type: "button" },
      React.createElement("span", { className: "sr-caret" }, expanded ? "▾" : "▸"),
      React.createElement("span", { className: "sr-table-name" }, t.name),
      React.createElement("span", { className: "sr-meta" },
        t.columns.length + "c · " + rels.length + "r"),
      isCurrent && React.createElement("span", { className: "sr-pin" }, "context"),
      isBase && !isCurrent && React.createElement("span", { className: "sr-pin base" }, "base")
    ),
    expanded && React.createElement(
      "div",
      { className: "sr-body" },
      React.createElement(
        "div",
        { className: "sr-section-label" }, "Columns"
      ),
      ...t.columns.map((c) =>
        React.createElement(
          "button",
          {
            key: c.name,
            className: "sr-row col",
            onClick: () => onInsert(c.name),
            title: "Insert " + c.name,
            type: "button",
          },
          React.createElement("span", { className: "sr-icon col" }, "C"),
          React.createElement("span", { className: "sr-label" }, c.name),
          React.createElement("span", { className: "sr-type" }, c.type)
        )
      ),
      rels.length > 0 && React.createElement("div", { className: "sr-section-label" }, "Relationships"),
      ...rels.map((r) =>
        React.createElement(
          "button",
          {
            key: r.name,
            className: "sr-row rel" + (r.indexed ? "" : " unindexed"),
            onClick: () => onInsert(r.name + "."),
            title: r.indexed ? "Insert " + r.name : "Unindexed join — will produce a warning",
            type: "button",
          },
          React.createElement("span", { className: "sr-icon rel" }, "→"),
          React.createElement("span", { className: "sr-label" }, r.name),
          React.createElement("span", { className: "sr-type" }, r.to_table),
          !r.indexed && React.createElement("span", { className: "sr-bang", title: "unindexed" }, "!")
        )
      )
    )
  );
}

function SchemaRail({ baseTable, contextTable, onInsert, query = "", onQuery, collapsed, onToggleCollapsed }) {
  const [expanded, setExpanded] = React.useState(() => new Set([contextTable, baseTable]));
  React.useEffect(() => {
    setExpanded((s) => {
      const n = new Set(s);
      n.add(contextTable);
      n.add(baseTable);
      return n;
    });
  }, [contextTable, baseTable]);

  // sort tables: context first, base second, rest alpha
  const tables = React.useMemo(() => {
    const all = [...FQ_S.TABLES];
    all.sort((a, b) => a.name.localeCompare(b.name));
    const ordered = [];
    const ctx = all.find((t) => t.name === contextTable);
    if (ctx) ordered.push(ctx);
    if (baseTable !== contextTable) {
      const b = all.find((t) => t.name === baseTable);
      if (b) ordered.push(b);
    }
    for (const t of all) {
      if (t.name === contextTable || t.name === baseTable) continue;
      ordered.push(t);
    }
    return ordered;
  }, [baseTable, contextTable]);

  const filtered = query
    ? tables.filter((t) =>
        t.name.includes(query) ||
        t.columns.some((c) => c.name.includes(query)) ||
        FQ_S.relsFrom(t.name).some((r) => r.name.includes(query)))
    : tables;

  if (collapsed) {
    return React.createElement(
      "aside",
      { className: "sr collapsed" },
      React.createElement(
        "button",
        { className: "sr-collapse", onClick: onToggleCollapsed, title: "Expand schema" },
        "›"
      ),
      React.createElement("div", { className: "sr-vlabel" }, "SCHEMA")
    );
  }

  return React.createElement(
    "aside",
    { className: "sr" },
    React.createElement(
      "div",
      { className: "sr-head" },
      React.createElement("div", { className: "sr-title" }, "Schema"),
      React.createElement("div", { className: "sr-sub" }, "rental"),
      React.createElement(
        "button",
        { className: "sr-collapse", onClick: onToggleCollapsed, title: "Collapse" },
        "‹"
      )
    ),
    React.createElement("input", {
      className: "sr-search",
      placeholder: "Filter columns, tables, relationships…",
      value: query,
      onChange: (e) => onQuery(e.target.value),
    }),
    React.createElement(
      "div",
      { className: "sr-list" },
      filtered.map((t) =>
        React.createElement(TableRow, {
          key: t.name,
          t,
          expanded: expanded.has(t.name),
          onToggle: () => setExpanded((s) => {
            const n = new Set(s);
            n.has(t.name) ? n.delete(t.name) : n.add(t.name);
            return n;
          }),
          onInsert,
          isCurrent: t.name === contextTable,
          isBase: t.name === baseTable,
        })
      )
    )
  );
}

window.SchemaRail = SchemaRail;
