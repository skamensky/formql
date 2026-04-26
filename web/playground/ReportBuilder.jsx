/* ReportBuilder.jsx — document mode authoring with:
   - Drill-down relationship navigator (replaces flat table list)
   - Field expression edit popup (with FormulaEditor) */

const FQ_RB = window.FORMQL;

function isCustomExpr(expr) {
  return /[^A-Za-z0-9_.]/.test(expr.trim());
}

function resolveNavTable(baseTable, navPath) {
  let table = baseTable;
  for (const rel of navPath) table = rel.to_table;
  return table;
}

// ─── Field expression edit modal ─────────────────────────────────
function FieldEditModal({ field, baseTable, onSave, onClose }) {
  const [expr, setExpr] = React.useState(field.expr);
  const [alias, setAlias] = React.useState(field.alias);
  const [contextTable, setContextTable] = React.useState(baseTable);

  const compile = React.useMemo(
    () => FQ_RB.compile(baseTable, expr, "formula"),
    [baseTable, expr]
  );
  const status = compile.ok ? "ok" : expr.trim() ? "err" : "idle";

  function handleSave() {
    if (!expr.trim() || !compile.ok) return;
    onSave({ ...field, expr: expr.trim(), alias: alias || field.alias });
  }

  return React.createElement(
    "div",
    {
      className: "fe-backdrop",
      onMouseDown: (e) => { if (e.target === e.currentTarget) onClose(); },
    },
    React.createElement(
      "div",
      { className: "fe-box" },
      React.createElement(
        "div",
        { className: "fe-head" },
        React.createElement("div", { className: "fe-head-title" },
          React.createElement("span", { className: "fe-head-label" }, "Edit field"),
          React.createElement("span", { className: "fe-head-alias" }, field.alias)
        ),
        React.createElement("button", { className: "fe-close", onClick: onClose, type: "button" }, "✕")
      ),
      React.createElement(
        "div",
        { className: "fe-alias-row" },
        React.createElement("label", { className: "fe-alias-label" }, "Alias"),
        React.createElement("input", {
          className: "fe-alias-input",
          value: alias,
          onChange: (e) => setAlias(e.target.value),
          spellCheck: false,
        })
      ),
      React.createElement(
        "div",
        { className: "fe-editor-wrap" },
        React.createElement(
          "div",
          { className: "fe-editor-label-row" },
          React.createElement("span", { className: "fe-editor-label" }, "Expression"),
          React.createElement("span", { className: "fe-status " + status },
            React.createElement("span", { className: "fe-status-dot" }),
            status === "ok"
              ? (compile.hir.projections[0]?.type || "ok")
              : status === "err"
              ? (compile.errors[0]?.message || "error")
              : "—"
          )
        ),
        React.createElement(window.FormulaEditor, {
          value: expr,
          onChange: setExpr,
          baseTable,
          mode: "formula",
          errors: compile.errors,
          warnings: compile.warnings,
          onSchemaContextChange: setContextTable,
          height: 120,
          ariaLabel: "Field expression",
        })
      ),
      React.createElement(
        "div",
        { className: "fe-foot" },
        React.createElement("span", { className: "fe-foot-ctx" },
          "cursor in ", React.createElement("strong", null, contextTable)
        ),
        React.createElement(
          "div",
          { className: "fe-foot-actions" },
          React.createElement("button", { className: "fe-btn-cancel", onClick: onClose, type: "button" }, "Cancel"),
          React.createElement("button", {
            className: "fe-btn-save",
            onClick: handleSave,
            type: "button",
            disabled: !expr.trim() || !compile.ok,
          }, "Save")
        )
      )
    )
  );
}

// ─── Drill-down relationship navigator ───────────────────────────
function PickerNav({ baseTable, fields, onAddField }) {
  const [navPath, setNavPath] = React.useState([]);
  const [colFilter, setColFilter] = React.useState("");
  React.useEffect(() => { setNavPath([]); setColFilter(""); }, [baseTable]);

  const currentTableName = resolveNavTable(baseTable, navPath);
  const currentTable = FQ_RB.tableByName(currentTableName);
  const rels = FQ_RB.relsFrom(currentTableName);

  function exprForCol(colName) {
    return navPath.map((r) => r.name).concat([colName]).join(".");
  }

  function addCol(col) {
    const expr = exprForCol(col.name);
    const lastRel = navPath[navPath.length - 1];
    const alias = navPath.length === 0
      ? col.name
      : lastRel.name.replace(/__rel$/, "") + "_" + col.name;
    onAddField({
      id: expr, expr, alias,
      finalTable: currentTableName,
      type: col.type,
      pathLabel: navPath.map((r) => r.name.replace(/__rel$/, "")).concat([col.name]).join(" › "),
      indexedPath: navPath.every((r) => r.indexed),
    });
  }

  const filteredCols = colFilter
    ? (currentTable?.columns || []).filter((c) => c.name.toLowerCase().includes(colFilter.toLowerCase()))
    : (currentTable?.columns || []);

  return React.createElement(
    "div",
    { className: "rb-nav" },
    React.createElement(
      "div",
      { className: "rb-nav-crumb" },
      navPath.length > 0 && React.createElement("button", {
        className: "rb-nav-back", onClick: () => setNavPath(navPath.slice(0, -1)), type: "button",
      }, "←"),
      React.createElement(
        "div",
        { className: "rb-nav-crumb-path" },
        React.createElement("button", {
          className: "rb-nav-crumb-seg", onClick: () => setNavPath([]), type: "button",
        }, baseTable),
        navPath.map((r, i) => React.createElement(
          React.Fragment, { key: i },
          React.createElement("span", { className: "rb-nav-crumb-sep" }, "›"),
          React.createElement("button", {
            className: "rb-nav-crumb-seg" + (i === navPath.length - 1 ? " active" : ""),
            onClick: () => setNavPath(navPath.slice(0, i + 1)),
            type: "button",
          }, r.name.replace(/__rel$/, ""))
        ))
      )
    ),
    React.createElement("input", {
      className: "rb-picker-search rb-nav-filter",
      placeholder: "Filter columns…",
      value: colFilter,
      onChange: (e) => setColFilter(e.target.value),
    }),
    React.createElement(
      "div",
      { className: "rb-nav-list" },
      filteredCols.length > 0 && React.createElement(
        React.Fragment, null,
        React.createElement("div", { className: "rb-nav-section" }, "Columns"),
        filteredCols.map((col) => {
          const expr = exprForCol(col.name);
          const added = !!fields.find((f) => f.id === expr);
          return React.createElement(
            "button",
            { key: col.name, className: "rb-pick" + (added ? " added" : ""), onClick: () => addCol(col), type: "button", disabled: added },
            React.createElement("span", { className: "rb-pick-icon" }, "C"),
            React.createElement("span", { className: "rb-pick-name" }, col.name),
            React.createElement("span", { className: "rb-pick-type" }, col.type),
            React.createElement("span", { className: "rb-pick-add" }, added ? "✓" : "+")
          );
        })
      ),
      !colFilter && rels.length > 0 && React.createElement(
        React.Fragment, null,
        React.createElement("div", { className: "rb-nav-section" }, "Relationships"),
        rels.map((rel) => React.createElement(
          "button",
          {
            key: rel.name,
            className: "rb-nav-rel",
            onClick: () => { setNavPath([...navPath, rel]); setColFilter(""); },
            type: "button",
          },
          React.createElement("span", { className: "rb-pick-icon rel" }, "R"),
          React.createElement("span", { className: "rb-nav-rel-name" }, rel.name.replace(/__rel$/, "")),
          React.createElement("span", { className: "rb-nav-rel-to" }, rel.to_table),
          React.createElement("span", { className: "rb-nav-rel-arr" + (!rel.indexed ? " warn" : "") }, "›")
        ))
      )
    )
  );
}

// ─── Main ReportBuilder ──────────────────────────────────────────
function ReportBuilder({ baseTable, fields, onFields, compile, exec, isExecuting, onRunAll }) {
  const [editingField, setEditingField] = React.useState(null);

  function removeField(id) {
    const f = fields.find((x) => x.id === id);
    if (f && isCustomExpr(f.expr)) {
      if (!window.confirm("This field has a custom expression. Remove anyway?")) return;
    }
    onFields(fields.filter((x) => x.id !== id));
  }

  function clearAll() {
    if (fields.some((f) => isCustomExpr(f.expr))) {
      if (!window.confirm("Some fields have custom expressions. Clear all anyway?")) return;
    }
    onFields([]);
  }

  function moveField(id, dir) {
    const idx = fields.findIndex((f) => f.id === id);
    if (idx < 0) return;
    const next = [...fields];
    const j = idx + dir;
    if (j < 0 || j >= next.length) return;
    [next[idx], next[j]] = [next[j], next[idx]];
    onFields(next);
  }

  function renameField(id, alias) {
    onFields(fields.map((f) => f.id === id ? { ...f, alias } : f));
  }

  function saveFieldEdit(updated) {
    onFields(fields.map((f) => f.id === updated.id ? updated : f));
    setEditingField(null);
  }

  function addField(f) {
    if (fields.find((x) => x.id === f.id)) return;
    onFields([...fields, f]);
  }

  const previewRows = exec?.ok ? exec.rows : generatePreview(fields);

  return React.createElement(
    "div",
    { className: "rb" },
    editingField && React.createElement(FieldEditModal, {
      field: editingField, baseTable, onSave: saveFieldEdit, onClose: () => setEditingField(null),
    }),
    React.createElement(
      "div",
      { className: "rb-fields" },
      React.createElement("div", { className: "rb-fields-label" }, "REPORT FIELDS"),
      fields.length === 0 && React.createElement("div", { className: "rb-empty-fields" },
        "Navigate relationships on the right → click columns to add fields"),
      React.createElement(
        "div",
        { className: "rb-chips" },
        fields.map((f, i) =>
          React.createElement(
            "div",
            { key: f.id, className: "rb-chip" + (f.indexedPath ? "" : " unindexed"), title: f.expr },
            React.createElement(
              "div",
              { className: "rb-chip-bar" },
              React.createElement("span", { className: "rb-chip-i" }, i + 1),
              React.createElement("button", { className: "rb-chip-mv", onClick: () => moveField(f.id, -1), title: "Move left", type: "button" }, "‹"),
              React.createElement("button", { className: "rb-chip-mv", onClick: () => moveField(f.id, 1), title: "Move right", type: "button" }, "›"),
              React.createElement("button", { className: "rb-chip-edit", onClick: () => setEditingField(f), title: "Edit expression", type: "button" }, "✎"),
              React.createElement("button", { className: "rb-chip-x", onClick: () => removeField(f.id), title: "Remove", type: "button" }, "✕")
            ),
            React.createElement("input", {
              className: "rb-chip-alias",
              value: f.alias,
              onChange: (e) => renameField(f.id, e.target.value),
              spellCheck: false,
            }),
            React.createElement("div", { className: "rb-chip-path" },
              isCustomExpr(f.expr) && React.createElement("span", { className: "rb-chip-custom" }, "custom "),
              f.pathLabel),
            React.createElement(
              "div",
              { className: "rb-chip-foot" },
              React.createElement("span", { className: "rb-chip-type" }, f.type),
              !f.indexedPath && React.createElement("span", { className: "rb-chip-flag" }, "unindexed")
            )
          )
        )
      ),
      fields.length > 0 && React.createElement("button", { className: "rb-clear", onClick: clearAll, type: "button" }, "Clear all")
    ),
    React.createElement(
      "div",
      { className: "rb-body" },
      React.createElement(
        "section",
        { className: "rb-preview" },
        React.createElement(
          "div",
          { className: "rb-preview-head" },
          React.createElement("div", { className: "rb-preview-title" }, "Preview"),
          React.createElement("div", { className: "rb-preview-meta" },
            exec?.ok
              ? `${exec.rows.length} row${exec.rows.length === 1 ? "" : "s"} · ${exec.elapsed_ms}ms · live`
              : `${fields.length} field${fields.length === 1 ? "" : "s"} · sample`),
          React.createElement("button", {
            className: "rb-run", onClick: onRunAll, type: "button",
            disabled: isExecuting || !compile?.ok || fields.length === 0,
          }, isExecuting ? "running…" : "▸ Run")
        ),
        fields.length === 0
          ? React.createElement("div", { className: "rb-preview-empty" },
              React.createElement("div", { className: "rb-preview-empty-mark" }, "▦"),
              React.createElement("div", null, "Add fields to compose your report"))
          : React.createElement(
              "div",
              { className: "rb-rows-wrap" },
              React.createElement("table", { className: "rb-rows" },
                React.createElement("thead", null,
                  React.createElement("tr", null,
                    React.createElement("th", { className: "rn" }, "#"),
                    ...fields.map((f) =>
                      React.createElement("th", { key: f.id },
                        React.createElement("div", { className: "th-name" }, f.alias),
                        React.createElement("div", { className: "th-type" }, f.type))))),
                React.createElement("tbody", null,
                  previewRows.map((r, ri) =>
                    React.createElement("tr", { key: ri },
                      React.createElement("td", { className: "rn" }, ri + 1),
                      ...fields.map((f) =>
                        React.createElement("td", { key: f.id },
                          formatVal(r[f.alias] ?? r[f.id] ?? samplePreviewVal(f, ri))))))))
            )
      ),
      React.createElement(
        "section",
        { className: "rb-picker" },
        React.createElement("div", { className: "rb-picker-head" },
          React.createElement("div", { className: "rb-picker-title" }, "Add fields")),
        React.createElement(PickerNav, { baseTable, fields, onAddField: addField })
      )
    )
  );
}

function formatVal(v) {
  if (v == null) return "—";
  return String(v);
}

function generatePreview(fields) {
  return Array.from({ length: 8 }, (_, i) => {
    const row = {};
    for (const f of fields) row[f.alias] = samplePreviewVal(f, i);
    return row;
  });
}

function samplePreviewVal(f, i) {
  const path = f.expr;
  const emails = ["maya@ex.com","jpark@ex.com","aisha@ex.com","leo.t@ex.com","sasha@ex.com","ben@ex.com","noor@ex.com","riley@ex.com"];
  const models = ["Toyota Corolla","Hyundai Elantra","Ford Escape","Honda Civic","Tesla Model 3","VW Jetta","Nissan Sentra","Kia Forte"];
  const branches = ["SFO Downtown","OAK Airport","SJC Central","LAX Terminal","SAN Marina","PDX Pearl","SEA Capitol","DEN Tech"];
  const reps = ["Rao","Webb","Zhao","Costa","Bauer","Saleh","Reyes","Klein"];
  if (path.endsWith("actual_total")) return (220 + i * 47.3).toFixed(2);
  if (path.endsWith("sale_price")) return (12400 + i * 1830).toFixed(2);
  if (path.endsWith("quoted_total")) return (180 + i * 39.2).toFixed(2);
  if (path.endsWith("email")) return emails[i % emails.length];
  if (path.endsWith("first_name")) return ["Priya","Tom","Lin","Marco","Eva","Omar","Talia","Hugo"][i % 8];
  if (path.endsWith("last_name")) return reps[i % reps.length];
  if (path.endsWith("model_name")) return models[i % models.length];
  if (path.endsWith("model_year")) return 2020 + (i % 5);
  if (path.endsWith("name")) return branches[i % branches.length];
  if (f.type === "numeric") return (100 + i * 17.3).toFixed(2);
  if (f.type === "int4") return 1000 + i;
  return "—";
}

window.ReportBuilder = ReportBuilder;
window.fieldsToDocument = function (fields) {
  return fields.map((f) => `${f.expr} AS ${f.alias}`).join(",\n");
};
