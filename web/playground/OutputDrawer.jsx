/* OutputDrawer.jsx — bottom drawer with status pills and expandable detail panes.
   Pills: Compile · Verify · Rows. Each can be opened/collapsed.
   Tabs inside: SQL | Diagnostics | Compile (HIR) | Raw JSON (toggleable). */

function StatusPill({ kind, state, label, value, onClick, active }) {
  const cls = "pill " + state + (active ? " active" : "");
  const ico = state === "ok" ? "✓" : state === "warn" ? "!" : state === "err" ? "✕" : state === "pending" ? "…" : "·";
  return React.createElement(
    "button",
    { className: cls, onClick, type: "button" },
    React.createElement("span", { className: "pill-dot" }, ico),
    React.createElement("span", { className: "pill-label" }, label),
    value !== undefined && React.createElement("span", { className: "pill-value" }, value)
  );
}

function SqlView({ sql }) {
  if (!sql) return React.createElement("div", { className: "od-empty" }, "No SQL — formula has errors.");
  // Highlight SQL keywords
  const KW = /\b(SELECT|FROM|LEFT JOIN|JOIN|ON|AS|WHERE|LIMIT|GROUP BY|ORDER BY|CAST|AND|OR)\b/g;
  const parts = [];
  let i = 0;
  let m;
  KW.lastIndex = 0;
  while ((m = KW.exec(sql)) !== null) {
    if (m.index > i) parts.push({ t: sql.slice(i, m.index) });
    parts.push({ t: m[0], k: "kw" });
    i = m.index + m[0].length;
  }
  if (i < sql.length) parts.push({ t: sql.slice(i) });

  return React.createElement(
    "pre",
    { className: "od-sql" },
    parts.map((p, ix) =>
      React.createElement("span", { key: ix, className: p.k ? "sql-kw" : "" }, p.t)
    )
  );
}

function DiagnosticsView({ errors, warnings }) {
  const items = [
    ...errors.map((e) => ({ ...e, kind: "err" })),
    ...warnings.map((w) => ({ ...w, kind: "warn" })),
  ];
  if (items.length === 0) {
    return React.createElement("div", { className: "od-empty ok" },
      React.createElement("span", { className: "ok-tick" }, "✓"),
      "No diagnostics. Compile clean.");
  }
  return React.createElement(
    "ul",
    { className: "od-diag" },
    items.map((it, i) =>
      React.createElement("li", { key: i, className: "diag " + it.kind },
        React.createElement("span", { className: "diag-icon" }, it.kind === "err" ? "✕" : "!"),
        React.createElement("div", { className: "diag-body" },
          React.createElement("div", { className: "diag-msg" }, it.message),
          it.hint && React.createElement("div", { className: "diag-hint" }, "hint: " + it.hint),
          it.code && React.createElement("div", { className: "diag-code" }, it.code),
          it.positioned &&
            React.createElement("div", { className: "diag-loc" }, "char " + it.start + "–" + it.end)
        )
      )
    )
  );
}

function HirView({ hir }) {
  if (!hir) return React.createElement("div", { className: "od-empty" }, "Compile to see HIR.");
  return React.createElement(
    "div",
    { className: "od-hir" },
    React.createElement("div", { className: "hir-row" },
      React.createElement("div", { className: "hir-k" }, "BASE TABLE"),
      React.createElement("div", { className: "hir-v mono" }, hir.base_table)),
    React.createElement("div", { className: "hir-row" },
      React.createElement("div", { className: "hir-k" }, "MODE"),
      React.createElement("div", { className: "hir-v" }, hir.mode || "formula")),
    React.createElement("div", { className: "hir-row" },
      React.createElement("div", { className: "hir-k" }, "JOINS · " + (hir.joins?.length || 0)),
      React.createElement("div", { className: "hir-v" },
        (hir.joins || []).map((j, i) =>
          React.createElement("div", { key: i, className: "hir-join " + (j.indexed ? "" : "unindexed") },
            React.createElement("span", { className: "mono" }, j.from),
            React.createElement("span", { className: "hir-arrow" }, "—" + j.via + "→"),
            React.createElement("span", { className: "mono" }, j.to),
            !j.indexed && React.createElement("span", { className: "hir-tag warn" }, "unindexed"))))),
    React.createElement("div", { className: "hir-row" },
      React.createElement("div", { className: "hir-k" }, "PROJECTIONS · " + (hir.projections?.length || 0)),
      React.createElement("div", { className: "hir-v" },
        (hir.projections || []).map((p, i) =>
          React.createElement("div", { key: i, className: "hir-proj" },
            React.createElement("span", { className: "mono" }, p.path),
            React.createElement("span", { className: "hir-tag" }, p.type)))))
  );
}

function ResultsTable({ exec }) {
  if (!exec) return React.createElement("div", { className: "od-empty" }, "Run to fetch rows.");
  if (!exec.ok) return React.createElement("div", { className: "od-empty err" }, exec.message);
  const cols = exec.columns;
  if (!exec.rows.length)
    return React.createElement("div", { className: "od-empty" }, "Query returned 0 rows.");
  return React.createElement(
    "div",
    { className: "od-rows-wrap" },
    React.createElement("table", { className: "od-rows" },
      React.createElement("thead", null,
        React.createElement("tr", null,
          React.createElement("th", { className: "rn" }, "#"),
          ...cols.map((c) =>
            React.createElement("th", { key: c.name },
              React.createElement("div", { className: "th-name" }, c.name),
              React.createElement("div", { className: "th-type" }, c.type))))),
      React.createElement("tbody", null,
        exec.rows.map((r, i) =>
          React.createElement("tr", { key: i },
            React.createElement("td", { className: "rn" }, i + 1),
            ...cols.map((c) =>
              React.createElement("td", { key: c.name, className: "td " + c.type },
                String(r[c.name] ?? "—"))))))),
    React.createElement("div", { className: "od-rows-foot" },
      exec.rows.length + " row" + (exec.rows.length === 1 ? "" : "s") +
      " · " + exec.elapsed_ms + "ms")
  );
}

function RawJsonView({ data }) {
  return React.createElement("pre", { className: "od-raw" }, JSON.stringify(data, null, 2));
}

function OutputDrawer({
  compile, exec, verify, isCompiling, isVerifying, isExecuting,
  showRaw, height = 320, onRunAll,
}) {
  const [tab, setTab] = React.useState("sql");

  // pill states
  const compileState = !compile ? "idle" :
    !compile.ok ? "err" : compile.warnings.length ? "warn" : "ok";
  const verifyState = !verify ? "idle" :
    isVerifying ? "pending" :
    verify.ok ? "ok" : "err";
  const execState = !exec ? "idle" :
    isExecuting ? "pending" :
    exec.ok ? "ok" : "err";

  const compileLabel = isCompiling ? "compiling…" :
    !compile ? "compile" :
    !compile.ok ? compile.errors.length + " error" + (compile.errors.length === 1 ? "" : "s") :
    compile.warnings.length ? compile.warnings.length + " warning" + (compile.warnings.length === 1 ? "" : "s") :
    "compiled";

  const verifyLabel = isVerifying ? "verifying…" :
    !verify ? "verify" :
    verify.ok ? "verified" : "verify failed";

  const execLabel = isExecuting ? "running…" :
    !exec ? "execute" :
    exec.ok ? exec.rows.length + " row" + (exec.rows.length === 1 ? "" : "s") :
    "exec failed";

  const tabs = [
    ["sql", "SQL"],
    ["diag", "Diagnostics", (compile?.errors.length || 0) + (compile?.warnings.length || 0)],
    ["rows", "Rows", exec?.ok ? exec.rows.length : null],
    ["hir", "Compile"],
  ];
  if (showRaw) tabs.push(["raw", "Raw JSON"]);

  return React.createElement(
    "section",
    { className: "od", style: { height } },
    React.createElement(
      "header",
      { className: "od-head" },
      React.createElement(
        "div",
        { className: "od-pills" },
        React.createElement(StatusPill, {
          state: compileState, label: compileLabel,
          active: tab === "diag" || tab === "hir",
          onClick: () => setTab(compileState === "err" ? "diag" : "sql"),
        }),
        React.createElement("span", { className: "pill-sep" }, "·"),
        React.createElement(StatusPill, {
          state: verifyState, label: verifyLabel, active: false,
          onClick: () => setTab("sql"),
        }),
        React.createElement("span", { className: "pill-sep" }, "·"),
        React.createElement(StatusPill, {
          state: execState, label: execLabel, active: tab === "rows",
          onClick: () => setTab("rows"),
        })
      ),
      React.createElement(
        "div",
        { className: "od-tabs" },
        tabs.map(([k, label, badge]) =>
          React.createElement(
            "button",
            {
              key: k,
              className: "od-tab" + (tab === k ? " active" : ""),
              onClick: () => setTab(k),
              type: "button",
            },
            label,
            badge != null && React.createElement("span", { className: "od-tab-badge" }, badge)
          )
        ),
        React.createElement(
          "button",
          { className: "od-run", onClick: onRunAll, type: "button", disabled: !compile?.ok },
          "▸ Run All"
        )
      )
    ),
    React.createElement(
      "div",
      { className: "od-body" },
      tab === "sql" && React.createElement(SqlView, { sql: compile?.sql }),
      tab === "diag" && React.createElement(DiagnosticsView, {
        errors: compile?.errors || [],
        warnings: compile?.warnings || [],
      }),
      tab === "hir" && React.createElement(HirView, { hir: compile?.hir }),
      tab === "rows" && React.createElement(ResultsTable, { exec }),
      tab === "raw" && React.createElement(RawJsonView, {
        data: { compile, verify, execute: exec },
      })
    )
  );
}

window.OutputDrawer = OutputDrawer;
