(async function main() {
  const state = {
    catalog: null,
    schemaInfo: null,
    compilation: null,
    suggestions: [],
    suggestionIndex: 0,
    ready: false,
  };

  const presets = [
    {
      label: "Customer Email / Total",
      value: 'customer_id__rel.email & " / " & STRING(actual_total)',
      document: false,
      baseTable: "rental_contract",
    },
    {
      label: "Rep Manager / Branch",
      value: 'rep_id__rel.manager_id__rel.first_name & " @ " & rep_id__rel.branch_id__rel.name',
      document: false,
      baseTable: "rental_contract",
    },
    {
      label: "Vehicle Badge",
      value: 'vehicle_id__rel.model_name & " / " & STRING(vehicle_id__rel.model_year)',
      document: false,
      baseTable: "rental_contract",
    },
    {
      label: "Quote Route",
      value: 'pickup_branch_id__rel.name & " -> " & dropoff_branch_id__rel.name',
      document: false,
      baseTable: "rental_offer",
    },
    {
      label: "Contract Document",
      value: "actual_total, customer_id__rel.email AS customer_email, vehicle_id__rel.model_name AS vehicle_model",
      document: true,
      baseTable: "rental_contract",
    },
    {
      label: "Resale Margin Document",
      value: "sale_price, customer_id__rel.email AS buyer_email, rep_id__rel.manager_id__rel.last_name AS manager_last_name",
      document: true,
      baseTable: "resale_sale",
    },
  ];

  const baseTable = document.getElementById("base-table");
  const preset = document.getElementById("preset");
  const maxDepth = document.getElementById("max-depth");
  const documentMode = document.getElementById("document-mode");
  const editor = document.getElementById("editor");
  const suggestions = document.getElementById("suggestions");
  const status = document.getElementById("status");
  const sqlOutput = document.getElementById("sql-output");
  const compilationOutput = document.getElementById("compilation-output");
  const verificationOutput = document.getElementById("verification-output");
  const executionOutput = document.getElementById("execution-output");
  const currentTable = document.getElementById("current-table");
  const catalogView = document.getElementById("catalog-view");
  const summaryBase = document.getElementById("summary-base");
  const summaryJoins = document.getElementById("summary-joins");
  const summaryRows = document.getElementById("summary-rows");

  populatePresetOptions();
  bindEvents();

  try {
    await loadRuntime();
    await loadCatalogAndSchema();
    applyPreset(presets[0]);
    setStatus("ready");
  } catch (error) {
    setStatus(error.message, true);
  }

  function bindEvents() {
    preset.addEventListener("change", () => {
      applyPreset(presets[preset.selectedIndex]);
    });

    baseTable.addEventListener("change", async () => {
      await loadSchemaInfo(baseTable.value);
      renderCatalogViews();
      scheduleCompletion();
    });

    maxDepth.addEventListener("change", () => {
      scheduleCompletion();
    });

    documentMode.addEventListener("change", () => {
      scheduleCompletion();
    });

    editor.addEventListener("input", () => {
      scheduleCompletion();
    });
    editor.addEventListener("click", () => {
      scheduleCompletion();
      renderCurrentTableContext();
    });
    editor.addEventListener("keyup", (event) => {
      if (event.key === "Escape") {
        hideSuggestions();
        return;
      }
      if (event.key === "ArrowUp" || event.key === "ArrowDown" || event.key === "Enter" || event.key === "Tab") {
        return;
      }
      scheduleCompletion();
      renderCurrentTableContext();
    });
    editor.addEventListener("keydown", (event) => {
      if (suggestions.hidden) {
        if (event.key === " " && event.ctrlKey) {
          event.preventDefault();
          scheduleCompletion(true);
        }
        return;
      }

      if (event.key === "ArrowDown") {
        event.preventDefault();
        moveSuggestion(1);
      } else if (event.key === "ArrowUp") {
        event.preventDefault();
        moveSuggestion(-1);
      } else if (event.key === "Enter" || event.key === "Tab") {
        event.preventDefault();
        applySuggestion(state.suggestions[state.suggestionIndex]);
      } else if (event.key === "Escape") {
        event.preventDefault();
        hideSuggestions();
      }
    });

    document.getElementById("compile").addEventListener("click", async () => {
      await compileCurrent();
    });
    document.getElementById("verify").addEventListener("click", async () => {
      const compilation = await compileCurrent();
      if (compilation) {
        await verifyCompiled(compilation.sql.query);
      }
    });
    document.getElementById("execute").addEventListener("click", async () => {
      const compilation = await compileCurrent();
      if (compilation) {
        await executeCompiled(compilation.sql.query);
      }
    });
    document.getElementById("run-all").addEventListener("click", async () => {
      const compilation = await compileCurrent();
      if (!compilation) {
        return;
      }
      await verifyCompiled(compilation.sql.query);
      await executeCompiled(compilation.sql.query);
    });
  }

  async function loadRuntime() {
    const go = new Go();
    const response = await fetch("/wasm/formql.wasm");
    if (!response.ok) {
      throw new Error(`wasm fetch failed: ${response.status}`);
    }
    const bytes = await response.arrayBuffer();
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
    go.run(instance);
    await tick();
    if (!window.FormQL) {
      throw new Error("FormQL wasm runtime did not initialize");
    }
    state.ready = true;
  }

  async function loadCatalogAndSchema() {
    const catalogResponse = await fetch("/api/catalog/rental-agency");
    if (!catalogResponse.ok) {
      throw new Error(`catalog fetch failed: ${catalogResponse.status}`);
    }
    state.catalog = await catalogResponse.json();
    await loadSchemaInfo("rental_contract");
    updateBaseTableOptions();
    renderCatalogViews();
  }

  async function loadSchemaInfo(requestedBaseTable) {
    const url = new URL("/api/schema-info/rental-agency", window.location.origin);
    url.searchParams.set("base_table", requestedBaseTable);
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`schema info fetch failed: ${response.status}`);
    }
    const payload = await response.json();
    if (!payload.ok) {
      throw new Error(payload.error?.message || "schema info failed");
    }
    state.schemaInfo = payload.info;
    summaryBase.textContent = state.schemaInfo.base_table;
  }

  function updateBaseTableOptions() {
    const selected = state.schemaInfo?.base_table || "rental_contract";
    baseTable.innerHTML = state.schemaInfo.tables
      .map((table) => table.name)
      .sort()
      .map((name) => `<option value="${escapeHTML(name)}"${name === selected ? " selected" : ""}>${escapeHTML(name)}</option>`)
      .join("");
  }

  function populatePresetOptions() {
    preset.innerHTML = presets
      .map((item) => `<option>${escapeHTML(item.label)}</option>`)
      .join("");
  }

  function applyPreset(item) {
    if (!item) {
      return;
    }
    editor.value = item.value;
    documentMode.checked = item.document;
    if (baseTable.querySelector(`option[value="${item.baseTable}"]`)) {
      baseTable.value = item.baseTable;
    }
    loadSchemaInfo(baseTable.value)
      .then(() => {
        renderCatalogViews();
        scheduleCompletion();
      })
      .catch((error) => setStatus(error.message, true));
  }

  async function compileCurrent() {
    if (!state.ready || !state.catalog) {
      return null;
    }

    try {
      const options = compilerOptions();
      const result = documentMode.checked
        ? window.FormQL.compileDocumentCatalogJSON(state.catalog, editor.value, options)
        : window.FormQL.compileCatalogJSON(state.catalog, editor.value, options);
      if (!result.ok) {
        throw new Error(result.error?.message || "compile failed");
      }

      state.compilation = result.compilation;
      sqlOutput.textContent = result.compilation.sql.query;
      compilationOutput.textContent = prettyJSON(result.compilation);
      verificationOutput.textContent = "{}";
      executionOutput.innerHTML = "<div class=\"panel-body\">compile succeeded. execute to fetch rows.</div>";
      summaryJoins.textContent = String(result.compilation.hir?.joins?.length || 0);
      summaryRows.textContent = "0";
      setStatus("compile succeeded");
      renderCurrentTableContext();
      return result.compilation;
    } catch (error) {
      setStatus(error.message, true);
      compilationOutput.textContent = prettyJSON({ ok: false, error: { message: error.message } });
      sqlOutput.textContent = "";
      return null;
    }
  }

  async function verifyCompiled(sql) {
    verificationOutput.textContent = "{}";
    const response = await fetch("/api/verify-sql", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        sql,
        verify_mode: "syntax",
      }),
    });
    const payload = await response.json();
    verificationOutput.textContent = prettyJSON(payload);
    if (!payload.ok || !payload.verification?.ok) {
      setStatus(payload.error?.message || "verification failed", true);
      return;
    }
    setStatus("verification succeeded");
  }

  async function executeCompiled(sql) {
    executionOutput.innerHTML = "";
    const response = await fetch("/api/execute-sql", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        sql,
        max_rows: 25,
      }),
    });
    const payload = await response.json();
    if (!payload.ok) {
      executionOutput.innerHTML = `<div class="panel-body"><pre>${escapeHTML(prettyJSON(payload))}</pre></div>`;
      setStatus(payload.error?.message || "execution failed", true);
      return;
    }
    summaryRows.textContent = String(payload.rows.length);
    executionOutput.innerHTML = renderRows(payload.columns, payload.rows);
    setStatus(`execution returned ${payload.rows.length} row(s)`);
  }

  function scheduleCompletion(force = false) {
    if (!state.ready || !state.catalog) {
      return;
    }
    const offset = editor.selectionStart;
    const items = window.FormQL.completeCatalogJSON(state.catalog, editor.value, offset, compilerOptions());
    if (!items.ok) {
      hideSuggestions();
      return;
    }
    const visible = force || shouldShowSuggestions(editor.value, offset);
    if (!visible || !items.items || items.items.length === 0) {
      hideSuggestions();
      return;
    }

    state.suggestions = items.items.slice(0, 14);
    state.suggestionIndex = 0;
    renderSuggestions();
  }

  function renderSuggestions() {
    if (state.suggestions.length === 0) {
      hideSuggestions();
      return;
    }

    suggestions.hidden = false;
    suggestions.innerHTML = state.suggestions
      .map((item, index) => {
        const active = index === state.suggestionIndex ? " active" : "";
        return `<button class="suggestion${active}" data-index="${index}" type="button">
          <span>${escapeHTML(item.label)}</span>
          <small>${escapeHTML(item.detail || kindLabel(item.kind))}</small>
        </button>`;
      })
      .join("");

    suggestions.querySelectorAll(".suggestion").forEach((button) => {
      button.addEventListener("mousedown", (event) => {
        event.preventDefault();
        const index = Number(button.dataset.index);
        applySuggestion(state.suggestions[index]);
      });
    });
  }

  function moveSuggestion(delta) {
    if (state.suggestions.length === 0) {
      return;
    }
    state.suggestionIndex = (state.suggestionIndex + delta + state.suggestions.length) % state.suggestions.length;
    renderSuggestions();
  }

  function applySuggestion(item) {
    if (!item) {
      return;
    }

    const cursor = editor.selectionStart;
    const replacement = insertionRange(editor.value, cursor);
    const suffix = item.kind === 3 ? "(" : "";
    editor.value = editor.value.slice(0, replacement.start) + item.label + suffix + editor.value.slice(replacement.end);
    const nextCursor = replacement.start + item.label.length + suffix.length;
    editor.focus();
    editor.setSelectionRange(nextCursor, nextCursor);
    hideSuggestions();
    renderCurrentTableContext();
  }

  function hideSuggestions() {
    state.suggestions = [];
    suggestions.hidden = true;
    suggestions.innerHTML = "";
  }

  function compilerOptions() {
    return {
      baseTable: baseTable.value,
      fieldAlias: "result",
      maxRelationshipDepth: Number(maxDepth.value) || 30,
      verifyMode: "syntax",
    };
  }

  function renderCatalogViews() {
    renderCurrentTableContext();

    const current = baseTable.value;
    const outgoing = new Map();
    for (const relationship of state.schemaInfo.relationships) {
      if (!outgoing.has(relationship.from_table)) {
        outgoing.set(relationship.from_table, []);
      }
      outgoing.get(relationship.from_table).push(relationship);
    }

    catalogView.innerHTML = state.schemaInfo.tables
      .map((table) => {
        const rels = outgoing.get(table.name) || [];
        return `<div class="table-card${table.name === current ? " current" : ""}">
          <h4>${escapeHTML(table.name)}</h4>
          <div class="chips">
            ${table.columns.map((column) => renderChip(column.name, column.type, "column")).join("")}
          </div>
          <div class="chips" style="margin-top:0.65rem;">
            ${rels.map((relationship) => renderChip(relationship.name, relationship.to_table, "relationship")).join("")}
          </div>
        </div>`;
      })
      .join("");

    bindChipInsertions(catalogView);
  }

  function renderCurrentTableContext() {
    if (!state.schemaInfo) {
      return;
    }
    const resolved = resolveCurrentContextTable();
    const table = state.schemaInfo.tables.find((entry) => entry.name === resolved.currentTable);
    const relationships = state.schemaInfo.relationships.filter((relationship) => relationship.from_table === resolved.currentTable);

    currentTable.innerHTML = `<div class="table-card current">
      <h4>${escapeHTML(resolved.currentTable)}</h4>
      <div class="chips">
        ${table ? table.columns.map((column) => renderChip(column.name, column.type, "column")).join("") : ""}
      </div>
      <div class="chips" style="margin-top:0.65rem;">
        ${relationships.map((relationship) => renderChip(relationship.name, relationship.to_table, "relationship")).join("")}
      </div>
    </div>`;

    bindChipInsertions(currentTable);
  }

  function resolveCurrentContextTable() {
    const beforeCursor = editor.value.slice(0, editor.selectionStart);
    const fragment = beforeCursor.match(/[A-Za-z0-9_.]+$/)?.[0] || "";
    if (!fragment.includes(".")) {
      return { currentTable: baseTable.value };
    }

    const chain = fragment.split(".").slice(0, -1).filter(Boolean);
    let current = baseTable.value;
    for (const segment of chain) {
      const rel = state.schemaInfo.relationships.find(
        (relationship) => relationship.from_table === current && relationship.name === segment,
      );
      if (!rel) {
        break;
      }
      current = rel.to_table;
    }
    return { currentTable: current };
  }

  function bindChipInsertions(root) {
    root.querySelectorAll(".chip[data-insert]").forEach((element) => {
      element.addEventListener("click", () => {
        insertTextAtCursor(element.dataset.insert);
      });
    });
  }

  function insertTextAtCursor(value) {
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    editor.value = editor.value.slice(0, start) + value + editor.value.slice(end);
    const next = start + value.length;
    editor.focus();
    editor.setSelectionRange(next, next);
    scheduleCompletion();
  }

  function renderChip(label, detail, kind) {
    return `<button type="button" class="chip" data-insert="${escapeHTML(label)}">
      ${escapeHTML(label)}
      <small>${escapeHTML(kind === "relationship" ? "-> " + detail : detail)}</small>
    </button>`;
  }

  function renderRows(columns, rows) {
    if (!rows || rows.length === 0) {
      return '<div class="panel-body">query returned no rows.</div>';
    }
    return `<table>
      <thead>
        <tr>${columns.map((column) => `<th>${escapeHTML(column)}</th>`).join("")}</tr>
      </thead>
      <tbody>
        ${rows
          .map(
            (row) =>
              `<tr>${columns
                .map((column) => `<td>${escapeHTML(stringifyCell(row[column]))}</td>`)
                .join("")}</tr>`,
          )
          .join("")}
      </tbody>
    </table>`;
  }

  function shouldShowSuggestions(text, offset) {
    const fragment = text.slice(0, offset).match(/[A-Za-z0-9_.]+$/)?.[0] || "";
    return fragment.length > 0;
  }

  function insertionRange(text, cursor) {
    let start = cursor;
    while (start > 0 && /[A-Za-z0-9_]/.test(text[start - 1])) {
      start--;
    }
    let end = cursor;
    while (end < text.length && /[A-Za-z0-9_]/.test(text[end])) {
      end++;
    }
    return { start, end };
  }

  function kindLabel(kind) {
    switch (kind) {
      case 3:
        return "function";
      case 5:
        return "field";
      case 6:
        return "relationship";
      default:
        return "";
    }
  }

  function stringifyCell(value) {
    if (value === null || value === undefined) {
      return "NULL";
    }
    if (typeof value === "object") {
      return JSON.stringify(value);
    }
    return String(value);
  }

  function prettyJSON(value) {
    return JSON.stringify(value, null, 2);
  }

  function setStatus(message, isError = false) {
    status.textContent = message;
    status.classList.toggle("error", isError);
  }

  function escapeHTML(value) {
    return String(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;");
  }

  function tick() {
    return new Promise((resolve) => setTimeout(resolve, 0));
  }
})().catch((error) => {
  const status = document.getElementById("status");
  if (status) {
    status.textContent = error.message;
    status.classList.add("error");
  }
});
