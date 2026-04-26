/* App.jsx — single-artboard playground.
   Formula mode → IDE artboard with schema rail + output drawer.
   Document mode → Report builder artboard.
   Mode is lifted to the root so the TopBar seg switches between them. */

const { useState, useEffect, useMemo, useCallback } = React;
const FQX = window.FORMQL;

// ─── Shared playground state ──────────────────────────────────────
function usePlayground(initial) {
  const [baseTable, setBaseTable] = useState(initial.baseTable);
  const [mode, setMode]           = useState(initial.mode);
  const [formula, setFormula]     = useState(initial.formula);
  const [maxDepth, setMaxDepth]   = useState(30);
  const [contextTable, setContextTable] = useState(initial.baseTable);
  const [verify, setVerify]   = useState(null);
  const [exec, setExec]       = useState(null);
  const [isVerifying, setIsVerifying] = useState(false);
  const [isExecuting, setIsExecuting] = useState(false);

  const compile = useMemo(
    function () { return FQX.compile(baseTable, formula, mode); },
    [baseTable, formula, mode]
  );

  useEffect(function () { setVerify(null); setExec(null); }, [compile.sql]);

  const runVerify = useCallback(async function () {
    if (!compile.ok) return;
    setIsVerifying(true);
    const r = await FQX.verify(compile.sql);
    setVerify(r);
    setIsVerifying(false);
    return r;
  }, [compile]);

  const runExecute = useCallback(async function () {
    if (!compile.ok) return;
    setIsExecuting(true);
    const r = await FQX.execute(compile);
    setExec(r);
    setIsExecuting(false);
    return r;
  }, [compile]);

  const runAll = useCallback(async function () {
    if (!compile.ok) return;
    setIsVerifying(true);
    setIsExecuting(true);
    const v = await FQX.verify(compile.sql);
    setVerify(v);
    setIsVerifying(false);
    const e = await FQX.execute(compile);
    setExec(e);
    setIsExecuting(false);
  }, [compile]);

  return {
    baseTable, setBaseTable, mode, setMode, formula, setFormula,
    maxDepth, setMaxDepth, contextTable, setContextTable,
    compile, verify, exec, isVerifying, isExecuting,
    runVerify, runExecute, runAll,
  };
}

function statusFromCompile(c) {
  if (!c) return { state: "idle", msg: "—" };
  if (!c.ok) return { state: "err", msg: c.errors.length + " error" + (c.errors.length === 1 ? "" : "s") };
  if (c.warnings.length) return { state: "warn", msg: c.warnings.length + " warning" + (c.warnings.length === 1 ? "" : "s") };
  return { state: "ok", msg: "compiled · " + c.hir.joins.length + " join" + (c.hir.joins.length === 1 ? "" : "s") };
}

// ─── Variation A: IDE editor ──────────────────────────────────────
function IDEArtboard({ tweaks, onMode }) {
  const pg = usePlayground({
    baseTable: "rental_contract",
    mode: "formula",
    formula: 'customer_id__rel.email & " / " & STRING(actual_total)',
  });

  const [railCollapsed, setRailCollapsed] = useState(false);
  const [railQuery, setRailQuery]         = useState("");
  const [presetsOpen, setPresetsOpen]     = useState(false);

  const status = statusFromCompile(pg.compile);

  function insertAt(text) {
    pg.setFormula(function (s) { return s + text; });
  }

  function pickPreset(p) {
    pg.setBaseTable(p.baseTable);
    pg.setFormula(p.formula);
    if (p.mode !== "formula" && onMode) {
      // switch to report builder for document presets
      onMode(p.mode);
    }
    setPresetsOpen(false);
  }

  function handleMode(m) {
    pg.setMode(m);
    if (onMode) onMode(m);
  }

  return React.createElement(
    "div",
    { className: "ab" },
    React.createElement(window.TopBar, {
      baseTable:    pg.baseTable,
      onBaseTable:  pg.setBaseTable,
      mode:         pg.mode,
      onMode:       handleMode,
      maxDepth:     pg.maxDepth,
      onMaxDepth:   pg.setMaxDepth,
      status:       status.state,
      statusMsg:    status.msg,
      onToggleRail: function () { setRailCollapsed(function (v) { return !v; }); },
      onOpenPresets: function () { setPresetsOpen(true); },
    }),

    React.createElement(
      "div",
      { className: "presets-strip" },
      React.createElement("div", { className: "ps-label" }, "QUICK START"),
      React.createElement(
        "div",
        { className: "ps-cards" },
        FQX.PRESETS.slice(0, 4).map(function (p) {
          return React.createElement(
            "button",
            { key: p.id, className: "ps-card", onClick: function () { pickPreset(p); }, type: "button" },
            React.createElement("div", { className: "ps-card-row" },
              React.createElement("span", { className: "ps-card-title" }, p.title),
              React.createElement("span", { className: "ps-card-mode " + p.mode },
                p.mode === "document" ? "report" : "formula")),
            React.createElement("div", { className: "ps-card-desc" }, p.description),
            React.createElement("div", { className: "ps-card-foot" },
              React.createElement("span", { className: "ps-chip" }, p.baseTable))
          );
        }),
        React.createElement(
          "button",
          { className: "ps-more", onClick: function () { setPresetsOpen(true); }, type: "button" },
          "All examples →"
        )
      )
    ),

    React.createElement(
      "div",
      { className: "ide-main" },
      React.createElement(window.SchemaRail, {
        baseTable:         pg.baseTable,
        contextTable:      pg.contextTable,
        onInsert:          insertAt,
        query:             railQuery,
        onQuery:           setRailQuery,
        collapsed:         railCollapsed,
        onToggleCollapsed: function () { setRailCollapsed(function (v) { return !v; }); },
      }),
      React.createElement(
        "div",
        { className: "ide-center" },
        React.createElement(
          "div",
          { className: "editor-frame" },
          React.createElement(
            "div",
            { className: "ef-head" },
            React.createElement("div", { className: "ef-tab active" },
              React.createElement("span", { className: "ef-icon" }, "ƒ"),
              "formula"),
            React.createElement(
              "div",
              { className: "ef-actions" },
              React.createElement("span", { className: "ef-status " + status.state },
                React.createElement("span", { className: "ef-dot" }),
                status.msg),
              React.createElement("span", { className: "ef-kbd" },
                React.createElement("kbd", null, "⌃"),
                React.createElement("kbd", null, "Space"),
                " complete")
            )
          ),
          React.createElement(
            "div",
            { className: "ef-meter" },
            React.createElement("div", { className: "ef-meter-bar " + status.state })
          ),
          React.createElement(window.FormulaEditor, {
            value:    pg.formula,
            onChange: pg.setFormula,
            baseTable: pg.baseTable,
            mode:     pg.mode,
            errors:   pg.compile.errors,
            warnings: pg.compile.warnings,
            onSchemaContextChange: pg.setContextTable,
            height:   220,
            ariaLabel: "FormQL formula",
          }),
          React.createElement(
            "div",
            { className: "ef-foot" },
            React.createElement("div", { className: "ef-context" },
              React.createElement("span", { className: "ef-ctx-k" }, "cursor in"),
              React.createElement("span", { className: "ef-ctx-v mono" }, pg.contextTable)),
            pg.compile.ok && React.createElement("div", { className: "ef-typeinfo" },
              React.createElement("span", { className: "ef-ti-k" }, "result type"),
              React.createElement("span", { className: "ef-ti-v" },
                pg.compile.hir.projections.length === 0 ? "—"
                  : pg.compile.hir.projections.length === 1 ? pg.compile.hir.projections[0].type
                  : pg.compile.hir.projections.length + " cols"))
          )
        ),
        React.createElement(window.OutputDrawer, {
          compile:     pg.compile,
          exec:        pg.exec,
          verify:      pg.verify,
          isCompiling: false,
          isVerifying: pg.isVerifying,
          isExecuting: pg.isExecuting,
          showRaw:     tweaks.showRaw,
          onRunAll:    pg.runAll,
        })
      )
    ),

    React.createElement(window.PresetsCards, {
      open:    presetsOpen,
      onClose: function () { setPresetsOpen(false); },
      onPick:  pickPreset,
    })
  );
}

// ─── Variation B: Report builder ──────────────────────────────────
function ReportArtboard({ tweaks, onMode }) {
  const [baseTable, setBaseTable] = useState("rental_contract");
  const [maxDepth, setMaxDepth]   = useState(30);
  const [presetsOpen, setPresetsOpen] = useState(false);

  // Build initial fields
  const [fields, setFields] = useState(function () {
    var reach = (function () {
      var seen = new Map();
      seen.set("rental_contract", []);
      var q = [{ table: "rental_contract", path: [] }];
      while (q.length) {
        var item = q.shift();
        if (item.path.length >= 3) continue;
        var rels = FQX.relsFrom(item.table);
        for (var i = 0; i < rels.length; i++) {
          var r = rels[i];
          if (!seen.has(r.to_table)) {
            var np = item.path.concat([r]);
            seen.set(r.to_table, np);
            q.push({ table: r.to_table, path: np });
          }
        }
      }
      return seen;
    })();

    function f(table, col) {
      var t = FQX.tableByName(table);
      if (!t) return null;
      var c = t.columns.find(function (x) { return x.name === col; });
      if (!c) return null;
      var path = reach.get(table) || [];
      var traversal = path.map(function (r) { return r.name; }).concat([col]).join(".");
      var alias = path.length === 0 ? col
        : path[path.length - 1].name.replace(/__rel$/, "") + "_" + col;
      return {
        id: traversal, expr: traversal, alias: alias,
        finalTable: table, type: c.type,
        pathLabel: path.map(function (r) { return r.name.replace(/__rel$/, ""); }).concat([col]).join(" › "),
        indexedPath: path.every(function (r) { return r.indexed; }),
      };
    }

    return [
      f("rental_contract", "actual_total"),
      f("customer",        "email"),
      f("vehicle",         "model_name"),
      f("vehicle",         "model_year"),
    ].filter(Boolean);
  });

  var formula  = useMemo(function () { return window.fieldsToDocument(fields); }, [fields]);
  var compile  = useMemo(function () { return FQX.compile(baseTable, formula, "document"); }, [baseTable, formula]);

  const [exec, setExec]           = useState(null);
  const [verify, setVerify]       = useState(null);
  const [isExecuting, setIsExecuting] = useState(false);
  const [sqlOpen, setSqlOpen]     = useState(false);

  useEffect(function () { setExec(null); setVerify(null); }, [compile.sql]);

  async function runAll() {
    if (!compile.ok) return;
    setIsExecuting(true);
    const v = await FQX.verify(compile.sql);
    setVerify(v);
    const e = await FQX.execute(compile);
    setExec(e);
    setIsExecuting(false);
  }

  function pickPreset(p) {
    setBaseTable(p.baseTable);
    if (p.mode !== "document") {
      setFields([{
        id: "result", expr: p.formula, alias: "result",
        finalTable: p.baseTable, type: "text",
        pathLabel: p.title, indexedPath: true,
      }]);
    } else {
      var parts = p.formula.split(",").map(function (s) { return s.trim(); }).filter(Boolean);
      var next = parts.map(function (part, i) {
        var m = part.match(/^(.+?)\s+AS\s+(\w+)$/i);
        var expr  = m ? m[1].trim() : part;
        var alias = m ? m[2] : "f" + i;
        return { id: expr, expr: expr, alias: alias, finalTable: p.baseTable, type: "text", pathLabel: expr, indexedPath: true };
      });
      setFields(next);
    }
    setPresetsOpen(false);
  }

  var status = statusFromCompile(compile);

  return React.createElement(
    "div",
    { className: "ab" },
    React.createElement(window.TopBar, {
      baseTable:    baseTable,
      onBaseTable:  setBaseTable,
      mode:         "document",
      onMode:       function (m) { if (m === "formula" && onMode) onMode(m); },
      maxDepth:     maxDepth,
      onMaxDepth:   setMaxDepth,
      status:       status.state,
      statusMsg:    status.msg,
      onToggleRail: function () {},
      onOpenPresets: function () { setPresetsOpen(true); },
    }),
    React.createElement(window.ReportBuilder, {
      baseTable: baseTable, fields: fields, onFields: setFields,
      compile: compile, exec: exec, isExecuting: isExecuting, onRunAll: runAll,
    }),
    compile.ok && React.createElement(
      "div",
      { className: "rb-sql-panel" },
      React.createElement(
        "button",
        { className: "rb-sql-toggle", onClick: function () { setSqlOpen(function (v) { return !v; }); }, type: "button" },
        React.createElement("span", { className: "rb-sql-toggle-caret" }, sqlOpen ? "▾" : "▸"),
        "Generated SQL",
        React.createElement("span", { style: { marginLeft: "auto", color: "var(--muted-2)" } },
          compile.sql.split("\n").length + " lines")
      ),
      sqlOpen && React.createElement(
        "div",
        { className: "rb-sql-body" },
        React.createElement("pre", { className: "rb-sql-pre" }, compile.sql)
      )
    ),
    React.createElement(
      "div",
      { className: "rb-statusbar" },
      React.createElement("div", { className: "rbsb-pills" },
        React.createElement("span", { className: "pill " + status.state },
          React.createElement("span", { className: "pill-dot" },
            status.state === "ok" ? "✓" : status.state === "warn" ? "!" : "✕"),
          React.createElement("span", { className: "pill-label" },
            status.state === "ok"
              ? "compiled · " + compile.hir.joins.length + " join" + (compile.hir.joins.length === 1 ? "" : "s")
              : status.msg)),
        React.createElement("span", { className: "pill-sep" }, "·"),
        React.createElement("span", { className: "pill " + (verify ? "ok" : "idle") },
          React.createElement("span", { className: "pill-dot" }, verify ? "✓" : "·"),
          React.createElement("span", { className: "pill-label" }, verify ? "verified" : "not verified")),
        React.createElement("span", { className: "pill-sep" }, "·"),
        React.createElement("span", { className: "pill " + (exec && exec.ok ? "ok" : isExecuting ? "pending" : "idle") },
          React.createElement("span", { className: "pill-dot" }, exec && exec.ok ? "✓" : isExecuting ? "…" : "·"),
          React.createElement("span", { className: "pill-label" },
            exec && exec.ok ? exec.rows.length + " rows · " + exec.elapsed_ms + "ms"
              : isExecuting ? "running…" : "not run"))
      ),
      React.createElement("div", { className: "rbsb-right mono" },
        compile.ok
          ? compile.sql.split("\n").length + " lines · " + fields.length + " field" + (fields.length === 1 ? "" : "s")
          : "—")
    ),
    React.createElement(window.PresetsCards, {
      open: presetsOpen, onClose: function () { setPresetsOpen(false); }, onPick: pickPreset,
    })
  );
}

// ─── Root ─────────────────────────────────────────────────────────
function PlaygroundApp() {
  const [ready, setReady]         = useState(false);
  const [initError, setInitError] = useState(null);
  const [mode, setMode]           = useState("formula");

  const tweaks = { fontFamily: "JetBrains Mono", density: "cozy", showRaw: false };

  useEffect(function () {
    FQX.init()
      .then(function () { setReady(true); })
      .catch(function (e) { setInitError(e.message || "failed to initialize"); });
  }, []);

  if (initError) {
    return React.createElement("div", { className: "load-screen err" }, "Error: " + initError);
  }
  if (!ready) {
    return React.createElement("div", { className: "load-screen" }, "loading wasm…");
  }

  if (mode === "document") {
    return React.createElement(ReportArtboard, { tweaks: tweaks, onMode: setMode });
  }
  return React.createElement(IDEArtboard, { tweaks: tweaks, onMode: setMode });
}

const rootEl = document.getElementById("root");
ReactDOM.createRoot(rootEl).render(React.createElement(PlaygroundApp));
