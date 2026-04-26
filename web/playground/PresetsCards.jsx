/* PresetsCards.jsx — card grid for example presets. */

function PresetsCards({ open, onClose, onPick }) {
  if (!open) return null;
  const presets = window.FORMQL.PRESETS;
  return React.createElement(
    "div",
    { className: "pc-backdrop", onClick: onClose },
    React.createElement(
      "div",
      { className: "pc", onClick: (e) => e.stopPropagation() },
      React.createElement(
        "div",
        { className: "pc-head" },
        React.createElement("h3", null, "Examples"),
        React.createElement("p", null, "Pick one to load — sets base table, mode, and formula."),
        React.createElement("button", { className: "pc-close", onClick: onClose, type: "button" }, "✕")
      ),
      React.createElement(
        "div",
        { className: "pc-grid" },
        presets.map((p) =>
          React.createElement(
            "button",
            { key: p.id, className: "pc-card", onClick: () => onPick(p), type: "button" },
            React.createElement(
              "div",
              { className: "pc-card-head" },
              React.createElement("div", { className: "pc-card-title" }, p.title),
              React.createElement(
                "div",
                { className: "pc-card-mode " + p.mode },
                p.mode === "document" ? "Report" : "Formula"
              )
            ),
            React.createElement("div", { className: "pc-card-desc" }, p.description),
            React.createElement(
              "div",
              { className: "pc-card-foot" },
              React.createElement("span", { className: "pc-chip" },
                React.createElement("span", { className: "pc-chip-k" }, "base"),
                React.createElement("span", { className: "pc-chip-v" }, p.baseTable))
            ),
            React.createElement("pre", { className: "pc-card-formula" }, p.formula)
          )
        )
      )
    )
  );
}

window.PresetsCards = PresetsCards;
