/* Editor.jsx — IDE-like formula editor with live syntax highlight,
   character-level error/warning underlines, and overlay autocomplete.
   Renders a layered <pre> + <textarea> for highlight+input. */

const FQ = window.FORMQL;

function syntaxRender(src) {
  const tokens = FQ.tokenize(src);
  return tokens.map((tk, i) => {
    let cls = "tk-" + tk.kind;
    if (tk.kind === "ident") {
      // we don't yet know if it's column or relationship; leave as plain
      cls = "tk-col";
    }
    return React.createElement("span", { key: i, className: cls }, tk.text);
  });
}

function squigglesRender(src, errors, warnings) {
  // Build segments: array of { text, classes[] } where classes
  // include "err" / "warn" if any range covers that span.
  const all = [
    ...errors.map((e) => ({ ...e, kind: "err" })),
    ...warnings.map((w) => ({ ...w, kind: "warn" })),
  ].filter((r) => r.start < r.end);
  if (!all.length) return null;
  const segments = [];
  let cursor = 0;
  // make break points
  const bps = new Set([0, src.length]);
  for (const r of all) { bps.add(r.start); bps.add(r.end); }
  const points = [...bps].sort((a, b) => a - b);
  for (let i = 0; i < points.length - 1; i++) {
    const s = points[i], e = points[i + 1];
    if (s === e) continue;
    const text = src.slice(s, e);
    const classes = [];
    for (const r of all) {
      if (r.start <= s && r.end >= e && r.start < r.end) {
        classes.push(r.kind);
      }
    }
    segments.push({ text, classes });
  }
  return segments.map((seg, i) =>
    React.createElement(
      "span",
      { key: i, className: seg.classes.map((c) => "sq-" + c).join(" ") },
      seg.text
    )
  );
}

function FormulaEditor({
  value, onChange, baseTable, mode,
  errors = [], warnings = [],
  onSchemaContextChange, onCursorChange, ariaLabel,
  height = 220,
}) {
  const taRef = React.useRef(null);
  const overlayRef = React.useRef(null);
  const squigRef = React.useRef(null);
  const wrapRef = React.useRef(null);
  const [completion, setCompletion] = React.useState(null); // {items, partialStart, x, y}
  const [acIndex, setAcIndex] = React.useState(0);

  // sync scroll: textarea → overlay
  const syncScroll = React.useCallback(() => {
    const ta = taRef.current;
    if (!ta) return;
    if (overlayRef.current) {
      overlayRef.current.scrollTop = ta.scrollTop;
      overlayRef.current.scrollLeft = ta.scrollLeft;
    }
    if (squigRef.current) {
      squigRef.current.scrollTop = ta.scrollTop;
      squigRef.current.scrollLeft = ta.scrollLeft;
    }
  }, []);

  // recompute autocomplete
  const recompute = React.useCallback((force = false) => {
    const ta = taRef.current;
    if (!ta) return;
    const cursor = ta.selectionStart;
    const before = value.slice(0, cursor);
    const tail = before.match(/[A-Za-z0-9_.]*$/)?.[0] || "";

    onCursorChange?.(cursor);
    onSchemaContextChange?.(FQ.contextTable(baseTable, value, cursor));

    if (!force && tail.length === 0) {
      setCompletion(null); return;
    }
    const r = FQ.complete(baseTable, value, cursor);
    if (!r.items.length) { setCompletion(null); return; }

    // measure cursor position via mirror
    const pos = caretPos(ta, cursor);
    setCompletion({
      items: r.items.slice(0, 12),
      x: pos.left, y: pos.top + pos.height,
    });
    setAcIndex(0);
  }, [value, baseTable, onCursorChange, onSchemaContextChange]);

  React.useEffect(() => { recompute(); }, [value, baseTable]);

  function applyItem(item) {
    const ta = taRef.current;
    if (!ta || !item) return;
    const cursor = ta.selectionStart;
    let start = cursor;
    while (start > 0 && /[A-Za-z0-9_]/.test(value[start - 1])) start--;
    let end = cursor;
    while (end < value.length && /[A-Za-z0-9_]/.test(value[end])) end++;
    let suffix = "";
    if (item.kind === "function") suffix = "(";
    else if (item.kind === "relationship") suffix = ".";
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

  function onKeyDown(e) {
    if (completion) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setAcIndex((i) => (i + 1) % completion.items.length);
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setAcIndex((i) => (i - 1 + completion.items.length) % completion.items.length);
        return;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        applyItem(completion.items[acIndex]);
        return;
      }
      if (e.key === "Escape") { e.preventDefault(); setCompletion(null); return; }
    }
    if (e.key === " " && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      recompute(true);
    }
  }

  return React.createElement(
    "div",
    { className: "fq-editor", ref: wrapRef, style: { height } },
    React.createElement("div", { className: "fq-editor-overlay tk", ref: overlayRef },
      React.createElement("div", { className: "fq-editor-text" }, syntaxRender(value || "")),
      React.createElement("div", { className: "fq-editor-text", style: { color: "transparent" } },
        " ")
    ),
    React.createElement("div", { className: "fq-editor-overlay sq", ref: squigRef, "aria-hidden": "true" },
      React.createElement("div", { className: "fq-editor-text" },
        squigglesRender(value || "", errors, warnings))
    ),
    React.createElement("textarea", {
      ref: taRef,
      className: "fq-editor-input",
      value,
      onChange: (e) => onChange(e.target.value),
      onScroll: syncScroll,
      onKeyDown,
      onClick: () => recompute(),
      onKeyUp: (e) => {
        if (["ArrowLeft", "ArrowRight", "Home", "End"].includes(e.key)) recompute();
      },
      onBlur: () => setTimeout(() => setCompletion(null), 120),
      spellCheck: false,
      autoCapitalize: "off",
      autoCorrect: "off",
      "aria-label": ariaLabel || "Formula editor",
    }),
    completion && React.createElement(
      "div",
      {
        className: "fq-ac",
        style: { left: completion.x, top: completion.y },
      },
      completion.items.map((it, i) =>
        React.createElement(
          "button",
          {
            key: it.label + i,
            className: "fq-ac-item" + (i === acIndex ? " active" : ""),
            onMouseDown: (ev) => { ev.preventDefault(); applyItem(it); },
            onMouseEnter: () => setAcIndex(i),
            type: "button",
          },
          React.createElement("span", { className: "fq-ac-icon ico-" + it.kind },
            it.kind === "column" ? "C" :
            it.kind === "relationship" ? "→" :
            it.kind === "function" ? "ƒ" : "?"),
          React.createElement("span", { className: "fq-ac-label" }, it.label),
          React.createElement("span", { className: "fq-ac-detail" }, it.detail || ""),
          it.kind === "relationship" && !it.indexed &&
            React.createElement("span", { className: "fq-ac-warn", title: "Unindexed join" }, "!")
        )
      )
    )
  );
}

// caretPos uses a hidden mirror div to compute pixel coords for the
// caret in a textarea — works because textarea uses identical font.
function caretPos(ta, pos) {
  const styles = getComputedStyle(ta);
  const div = document.createElement("div");
  const props = [
    "boxSizing", "width", "height", "padding", "border", "fontFamily",
    "fontSize", "fontWeight", "lineHeight", "letterSpacing",
    "whiteSpace", "wordWrap", "tabSize",
  ];
  for (const p of props) div.style[p] = styles[p];
  div.style.position = "absolute";
  div.style.top = "0";
  div.style.left = "0";
  div.style.visibility = "hidden";
  div.style.whiteSpace = "pre-wrap";
  div.style.wordWrap = "break-word";
  div.style.overflow = "hidden";
  const before = ta.value.substring(0, pos);
  div.textContent = before;
  const span = document.createElement("span");
  span.textContent = ta.value.substring(pos) || ".";
  div.appendChild(span);
  ta.parentNode.appendChild(div);
  const r = ta.getBoundingClientRect();
  const sr = span.getBoundingClientRect();
  const dr = div.getBoundingClientRect();
  const x = sr.left - dr.left - ta.scrollLeft;
  const y = sr.top - dr.top - ta.scrollTop;
  div.remove();
  const lh = parseFloat(styles.lineHeight) || 18;
  return { left: x + 4, top: y + lh, height: lh };
}

window.FormulaEditor = FormulaEditor;
