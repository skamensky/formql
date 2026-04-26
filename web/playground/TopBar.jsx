/* Top toolbar: title, base table chip, mode segmented control,
   max depth, status dot, raw JSON toggle (when on). */

function TopBar({
  baseTable, onBaseTable, mode, onMode, maxDepth, onMaxDepth,
  status, statusMsg, onToggleRail, onOpenPresets, density, onDensity, fontFamily, onFont, showRaw, onShowRaw,
}) {
  const tables = window.FORMQL.TABLES.map((t) => t.name);
  const dotState = status; // "ok" | "warn" | "err" | "idle" | "pending"
  return React.createElement(
    "header",
    { className: "tb" },
    React.createElement(
      "div",
      { className: "tb-left" },
      React.createElement("button", { className: "tb-rail-toggle", onClick: onToggleRail, title: "Toggle schema" }, "≡"),
      React.createElement(
        "div",
        { className: "tb-brand" },
        React.createElement("div", { className: "tb-mark" }, "ƒ"),
        React.createElement("div", null,
          React.createElement("div", { className: "tb-title" }, "FormQL"),
          React.createElement("div", { className: "tb-sub" }, "playground"))
      ),
      React.createElement("div", { className: "tb-sep" }),
      // base table chip / select
      React.createElement(
        "label",
        { className: "tb-field" },
        React.createElement("span", { className: "tb-field-label" }, "BASE"),
        React.createElement(
          "select",
          { value: baseTable, onChange: (e) => onBaseTable(e.target.value), className: "tb-select" },
          tables.map((n) => React.createElement("option", { key: n, value: n }, n))
        )
      ),
      // mode segmented
      React.createElement(
        "div",
        { className: "tb-seg" },
        React.createElement(
          "button",
          { className: mode === "formula" ? "active" : "", onClick: () => onMode("formula"), type: "button" },
          "Formula"
        ),
        React.createElement(
          "button",
          { className: mode === "document" ? "active" : "", onClick: () => onMode("document"), type: "button" },
          "Report"
        )
      ),
      // depth
      React.createElement(
        "label",
        { className: "tb-field" },
        React.createElement("span", { className: "tb-field-label" }, "DEPTH"),
        React.createElement("input", {
          className: "tb-num",
          type: "number",
          min: 1,
          max: 100,
          value: maxDepth,
          onChange: (e) => onMaxDepth(Number(e.target.value) || 30),
        })
      )
    ),
    React.createElement(
      "div",
      { className: "tb-right" },
      // status dot
      React.createElement(
        "div",
        { className: "tb-status " + dotState },
        React.createElement("span", { className: "tb-dot" }),
        React.createElement("span", { className: "tb-status-text" }, statusMsg)
      ),
      React.createElement(
        "button",
        { className: "tb-btn", onClick: onOpenPresets, type: "button" },
        "Examples"
      )
    )
  );
}

window.TopBar = TopBar;
