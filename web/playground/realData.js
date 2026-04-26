/* realData.js — bridges the real WASM compiler + HTTP APIs to the window.FORMQL
   interface expected by the design components.

   window.FORMQL mirrors the mock shape from the design prototype so the UI
   components work without modification. */

(function () {
  // ─── Runtime state ────────────────────────────────────────────────
  let _catalogJSON = null;

  // ─── Dynamic schema (populated on init) ──────────────────────────
  let TABLES = [];
  let RELATIONSHIPS = [];

  // ─── Static data ─────────────────────────────────────────────────
  const FUNCTIONS = [
    { name: "STRING",   sig: "STRING(any) → text",                   desc: "Cast any value to text" },
    { name: "DATE",     sig: "DATE(text) → date",                    desc: "Parse text to date" },
    { name: "UPPER",    sig: "UPPER(text) → text",                   desc: "Uppercase a string" },
    { name: "LOWER",    sig: "LOWER(text) → text",                   desc: "Lowercase a string" },
    { name: "TRIM",     sig: "TRIM(text) → text",                    desc: "Trim whitespace" },
    { name: "LEN",      sig: "LEN(text) → int",                      desc: "Length of string" },
    { name: "ROUND",    sig: "ROUND(numeric, int) → numeric",         desc: "Round to N decimal places" },
    { name: "ABS",      sig: "ABS(numeric) → numeric",               desc: "Absolute value" },
    { name: "IF",       sig: "IF(bool, a, b) → a|b",                 desc: "Conditional expression" },
    { name: "AND",      sig: "AND(bool, bool, ...) → bool",          desc: "Logical AND" },
    { name: "OR",       sig: "OR(bool, bool, ...) → bool",           desc: "Logical OR" },
    { name: "NOT",      sig: "NOT(bool) → bool",                     desc: "Logical NOT" },
    { name: "COALESCE", sig: "COALESCE(a, b, ...) → first non-null", desc: "First non-null argument" },
    { name: "NULLVALUE",sig: "NULLVALUE(a, b, ...) → first non-null",desc: "Alias for COALESCE" },
    { name: "ISNULL",   sig: "ISNULL(any) → bool",                   desc: "True when value is NULL" },
    { name: "ISBLANK",  sig: "ISBLANK(any) → bool",                  desc: "True when NULL or empty string" },
    { name: "TODAY",    sig: "TODAY() → date",                       desc: "Current date" },
  ];

  const PRESETS = [
    {
      id: "customer-tag",
      title: "Customer tag",
      description: "Email & total — common label format for receipts.",
      baseTable: "rental_contract",
      mode: "formula",
      formula: 'customer_id__rel.email & " / " & STRING(actual_total)',
    },
    {
      id: "rep-chain",
      title: "Rep manager chain",
      description: "Two-hop traversal across rep → manager → branch.",
      baseTable: "rental_contract",
      mode: "formula",
      formula: 'rep_id__rel.manager_id__rel.first_name & " @ " & rep_id__rel.branch_id__rel.name',
    },
    {
      id: "vehicle-badge",
      title: "Vehicle badge",
      description: "Model name and year, hyphen-joined.",
      baseTable: "rental_contract",
      mode: "formula",
      formula: 'vehicle_id__rel.model_name & " / " & STRING(vehicle_id__rel.model_year)',
    },
    {
      id: "quote-route",
      title: "Quote route",
      description: "Pickup → dropoff branch names on a rental offer.",
      baseTable: "rental_offer",
      mode: "formula",
      formula: 'pickup_branch_id__rel.name & " -> " & dropoff_branch_id__rel.name',
    },
    {
      id: "contract-doc",
      title: "Contract report",
      description: "Multi-field document: total, customer email, vehicle.",
      baseTable: "rental_contract",
      mode: "document",
      formula: "actual_total,\ncustomer_id__rel.email AS customer_email,\nvehicle_id__rel.model_name AS vehicle_model",
    },
    {
      id: "resale-doc",
      title: "Resale margin report",
      description: "Resale price, buyer email, and manager last name.",
      baseTable: "resale_sale",
      mode: "document",
      formula: "sale_price,\ncustomer_id__rel.email AS buyer_email,\nrep_id__rel.manager_id__rel.last_name AS manager_last_name",
    },
  ];

  // ─── Lookup helpers ───────────────────────────────────────────────
  function tableByName(name) {
    return TABLES.find(function (t) { return t.name === name; });
  }

  function relsFrom(name) {
    return RELATIONSHIPS.filter(function (r) { return r.from_table === name; });
  }

  // ─── Tokenizer ────────────────────────────────────────────────────
  const KEYWORDS = ["AS"];

  function tokenize(src) {
    var tokens = [];
    var re = /(\s+)|("(?:[^"\\]|\\.)*")|(\d+(?:\.\d+)?)|([A-Za-z_][A-Za-z0-9_]*)|(&|\+|-|\*|\/|<=|>=|<|>|=|,|\(|\)|\.)|(\S)/g;
    var m;
    while ((m = re.exec(src)) !== null) {
      var whole = m[0], ws = m[1], str = m[2], num = m[3], ident = m[4], op = m[5];
      var start = m.index;
      var end = start + whole.length;
      var kind;
      if (ws) kind = "ws";
      else if (str) kind = "string";
      else if (num) kind = "number";
      else if (ident) {
        if (KEYWORDS.indexOf(whole.toUpperCase()) !== -1) kind = "keyword";
        else if (FUNCTIONS.find(function (f) { return f.name === whole; })) kind = "fn";
        else if (whole.endsWith("__rel")) kind = "rel";
        else kind = "ident";
      } else if (op) kind = "punct";
      else kind = "other";
      tokens.push({ start: start, end: end, text: whole, kind: kind });
    }
    return tokens;
  }

  // ─── Context table (cursor-based traversal) ───────────────────────
  function resolveChain(baseTable, segments) {
    var table = baseTable;
    for (var i = 0; i < segments.length; i++) {
      var rel = RELATIONSHIPS.find(function (r) { return r.from_table === table && r.name === segments[i]; });
      if (!rel) return { table: table, ok: false, brokenAt: i };
      table = rel.to_table;
    }
    return { table: table, ok: true };
  }

  function contextTable(baseTable, src, cursor) {
    var before = src.slice(0, cursor);
    var tail = (before.match(/[A-Za-z0-9_.]+\.?$/) || [""])[0];
    if (tail.indexOf(".") === -1) return baseTable;
    var segs = tail.split(".").slice(0, -1).filter(Boolean);
    var r = resolveChain(baseTable, segs);
    return r.ok ? r.table : baseTable;
  }

  // ─── Autocomplete (real WASM) ─────────────────────────────────────
  // LSP kind constants from the tooling package
  var KIND_FIELD    = 5;
  var KIND_REL      = 6;
  var KIND_FUNCTION = 3;

  function complete(baseTable, src, cursor) {
    if (!window.FormQL || !_catalogJSON) {
      return { items: [], context: baseTable, partial: "" };
    }
    try {
      var options = { baseTable: baseTable, maxRelationshipDepth: 30 };
      var result = window.FormQL.completeCatalogJSON(_catalogJSON, src, cursor, options);
      if (!result.ok) return { items: [], context: contextTable(baseTable, src, cursor), partial: "" };
      var items = (result.items || []).map(function (item) {
        var kind = item.kind === KIND_REL ? "relationship" : item.kind === KIND_FUNCTION ? "function" : "column";
        return {
          kind: kind,
          label: item.label,
          detail: item.detail || "",
          indexed: item.indexed !== false,
        };
      });
      return { items: items, context: contextTable(baseTable, src, cursor), partial: "" };
    } catch (e) {
      return { items: [], context: baseTable, partial: "" };
    }
  }

  // ─── Compile (real WASM, output mapped to mock shape) ─────────────
  function emptyCompile(baseTable, mode, message) {
    return {
      ok: false,
      sql: "",
      errors: [{ message: message, start: 0, end: 0, positioned: false }],
      warnings: [],
      hir: { base_table: baseTable, mode: mode, joins: [], projections: [] },
    };
  }

  function compile(baseTable, src, mode) {
    if (!window.FormQL || !_catalogJSON) {
      return emptyCompile(baseTable, mode, "compiler not ready");
    }
    if (!src.trim()) {
      return emptyCompile(baseTable, mode, "empty formula");
    }

    try {
      var options = { baseTable: baseTable, fieldAlias: "result", maxRelationshipDepth: 30 };
      var result = mode === "document"
        ? window.FormQL.compileDocumentCatalogJSON(_catalogJSON, src, options)
        : window.FormQL.compileCatalogJSON(_catalogJSON, src, options);

      if (!result.ok) {
        var errMsg = result.error ? result.error.message : "compile failed";
        var errHint = result.error ? (result.error.hint || "") : "";
        var errPos = result.error && result.error.position >= 0 ? result.error.position : -1;
        var errStart = errPos >= 0 ? errPos : 0;
        var errEnd = errStart;
        if (errPos >= 0) {
          errEnd = errStart + 1;
          while (errEnd < src.length && /[A-Za-z0-9_]/.test(src[errEnd])) errEnd++;
        }
        return {
          ok: false, sql: "",
          errors: [{ message: errMsg, hint: errHint, start: errStart, end: errEnd, positioned: errPos >= 0 }],
          warnings: [],
          hir: { base_table: baseTable, mode: mode, joins: [], projections: [] },
        };
      }

      var comp = result.compilation || {};
      var hir  = comp.hir || {};
      var sql  = (comp.sql && comp.sql.query) ? comp.sql.query : "";

      // Map real HIR joins → mock {from, to, via, indexed}
      var joins = (hir.joins || []).map(function (j) {
        var path = j.path || [];
        var via  = path.length > 0 ? path[path.length - 1] : (j.join_column || "") + "__rel";
        var from = j.from_table || baseTable;
        var rel  = RELATIONSHIPS.find(function (r) { return r.from_table === from && r.name === via; });
        return {
          from: from,
          to: j.to_table || "",
          via: via,
          indexed: rel ? rel.indexed : true,
        };
      });

      // Map projections
      var projections;
      if (mode === "document") {
        projections = (hir.fields || []).map(function (f) {
          return { path: f.alias, table: baseTable, column: f.alias, type: f.type || "text" };
        });
      } else {
        var exprType = hir.expr ? (hir.expr.type || "text") : "text";
        projections = sql ? [{ path: "result", table: baseTable, column: "result", type: exprType }] : [];
      }

      // Map warnings (diagnostic.Warning has {code, message})
      var warnings = (hir.warnings || []).map(function (w) {
        var msg  = (typeof w === "string") ? w : (w.message || String(w));
        var code = (typeof w === "object" && w.code) ? w.code : "warning";
        return { code: code, message: msg, start: 0, end: 0 };
      });

      return {
        ok: true,
        sql: sql,
        hir: { base_table: baseTable, mode: mode, joins: joins, projections: projections },
        errors: [],
        warnings: warnings,
      };
    } catch (e) {
      return emptyCompile(baseTable, mode, e.message || "compile failed");
    }
  }

  // ─── Verify (HTTP) ────────────────────────────────────────────────
  function verify(sql) {
    if (!sql) return Promise.resolve({ ok: false, message: "no SQL to verify" });
    return fetch("/api/verify-sql", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ sql: sql, verify_mode: "syntax" }),
    }).then(function (r) { return r.json(); }).then(function (payload) {
      var verOk = payload.ok && payload.verification && payload.verification.ok;
      return {
        ok: verOk,
        message: verOk ? "syntax ok" : (payload.error ? payload.error.message : "verification failed"),
        plan_hint: verOk ? "syntax check passed" : null,
      };
    });
  }

  // ─── Execute (HTTP) ───────────────────────────────────────────────
  function execute(compileResult) {
    if (!compileResult || !compileResult.ok) {
      return Promise.resolve({ ok: false, message: "cannot execute — compile failed" });
    }
    var t0 = Date.now();
    return fetch("/api/execute-sql", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ sql: compileResult.sql, max_rows: 25 }),
    }).then(function (r) { return r.json(); }).then(function (payload) {
      if (!payload.ok) {
        return { ok: false, message: payload.error ? payload.error.message : "execution failed" };
      }
      var elapsed = Date.now() - t0;
      var cols = payload.columns || [];
      var rows = (payload.rows || []).map(function (r) {
        var row = {};
        cols.forEach(function (c) { row[c] = r[c]; });
        return row;
      });
      return {
        ok: true,
        columns: cols.map(function (c) { return { name: c, type: "text" }; }),
        rows: rows,
        elapsed_ms: elapsed,
      };
    });
  }

  // ─── WASM loader ──────────────────────────────────────────────────
  function loadWASM() {
    var go = new Go();
    return fetch("/wasm/formql.wasm").then(function (r) {
      if (!r.ok) throw new Error("wasm fetch failed: " + r.status);
      return r.arrayBuffer();
    }).then(function (bytes) {
      return WebAssembly.instantiate(bytes, go.importObject);
    }).then(function (result) {
      go.run(result.instance);
      return new Promise(function (resolve) { setTimeout(resolve, 0); });
    }).then(function () {
      if (!window.FormQL) throw new Error("FormQL wasm runtime did not initialize");
    });
  }

  // ─── Catalog loader ───────────────────────────────────────────────
  function loadCatalog() {
    return fetch("/api/catalog/rental-agency").then(function (r) {
      if (!r.ok) throw new Error("catalog fetch failed: " + r.status);
      return r.json();
    }).then(function (data) {
      _catalogJSON = JSON.stringify(data);
    });
  }

  function loadSchemaInfo(baseTable) {
    var url = "/api/schema-info/rental-agency?base_table=" + encodeURIComponent(baseTable);
    return fetch(url).then(function (r) {
      if (!r.ok) throw new Error("schema-info fetch failed: " + r.status);
      return r.json();
    }).then(function (payload) {
      if (!payload.ok) throw new Error(payload.error ? payload.error.message : "schema-info failed");
      var info = payload.info || {};
      TABLES = (info.tables || []).map(function (t) {
        return {
          name: t.name,
          pk: "id",
          columns: (t.columns || []).map(function (c) { return { name: c.name, type: c.type }; }),
        };
      });
      RELATIONSHIPS = (info.relationships || []).map(function (r) {
        return {
          from_table: r.from_table,
          name: r.name,
          to_table: r.to_table,
          indexed: r.join_column_indexed !== false,
        };
      });
    });
  }

  // ─── Public init ──────────────────────────────────────────────────
  function init() {
    return Promise.all([loadWASM(), loadCatalog()])
      .then(function () { return loadSchemaInfo("rental_contract"); });
  }

  // ─── Export ───────────────────────────────────────────────────────
  window.FORMQL = {
    get TABLES() { return TABLES; },
    get RELATIONSHIPS() { return RELATIONSHIPS; },
    FUNCTIONS: FUNCTIONS,
    PRESETS: PRESETS,
    tableByName: tableByName,
    relsFrom: relsFrom,
    tokenize: tokenize,
    complete: complete,
    contextTable: contextTable,
    compile: compile,
    verify: verify,
    execute: execute,
    init: init,
  };
})();
