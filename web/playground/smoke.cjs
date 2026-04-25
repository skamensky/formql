const vm = require("vm");

const baseURL = process.env.FORMQL_WEB_URL || "http://127.0.0.1:8090";

async function main() {
  const health = await fetch(`${baseURL}/api/health`);
  const healthJSON = await health.json();
  if (!healthJSON.ok) {
    throw new Error("backend health check failed");
  }

  const wasmExecSource = await fetchText(`${baseURL}/wasm/wasm_exec.js`);
  vm.runInThisContext(wasmExecSource, { filename: "wasm_exec.js" });

  const go = new Go();
  const wasmBytes = await fetchBytes(`${baseURL}/wasm/formql.wasm`);
  const { instance } = await WebAssembly.instantiate(wasmBytes, go.importObject);
  go.run(instance);
  await new Promise((resolve) => setTimeout(resolve, 0));

  if (!globalThis.FormQL) {
    throw new Error("FormQL global was not registered");
  }

  const catalog = await fetchJSON(`${baseURL}/api/catalog/rental-agency`);
  const schemaInfoResponse = await fetchJSON(`${baseURL}/api/schema-info/rental-agency?base_table=rental_contract`);
  if (!schemaInfoResponse.ok) {
    throw new Error(`backend schema info failed: ${JSON.stringify(schemaInfoResponse)}`);
  }
  if (schemaInfoResponse.info.base_table !== "rental_contract") {
    throw new Error(`unexpected schema info base table: ${schemaInfoResponse.info.base_table}`);
  }
  if (!schemaInfoResponse.info.tables.some((table) => table.name === "rental_contract")) {
    throw new Error("schema info did not include rental_contract table");
  }

  const info = globalThis.FormQL.loadSchemaInfoJSON(catalog, {
    baseTable: "rental_contract",
  });
  if (!info.ok) {
    throw new Error(`schema info failed: ${info.error.message}`);
  }

  const compiled = globalThis.FormQL.compileCatalogJSON(
    catalog,
    'customer_rel.email & " / " & STRING(actual_total)',
    {
      baseTable: "rental_contract",
      fieldAlias: "result",
    },
  );
  if (!compiled.ok) {
    throw new Error(`wasm compile failed: ${compiled.error.message}`);
  }
  if (!compiled.compilation.sql.query.includes('"rental_contract"')) {
    throw new Error("wasm compiled SQL did not reference rental_contract");
  }

  const verified = await postJSON(`${baseURL}/api/verify-sql`, {
    sql: compiled.compilation.sql.query,
    verify_mode: "syntax",
  });
  if (!verified.ok || !verified.verification.ok) {
    throw new Error(`backend verification failed: ${JSON.stringify(verified)}`);
  }

  const backendCompiled = await postJSON(`${baseURL}/api/compile-and-verify`, {
    catalog_json: catalog,
    formula: "actual_total + 1",
    field_alias: "result",
    verify_mode: "syntax",
  });
  if (!backendCompiled.ok || !backendCompiled.verification.ok) {
    throw new Error(`backend compile+verify failed: ${JSON.stringify(backendCompiled)}`);
  }

  console.log("web backend + wasm smoke passed");
}

async function fetchText(url) {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`GET ${url} failed: ${response.status}`);
  }
  return response.text();
}

async function fetchBytes(url) {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`GET ${url} failed: ${response.status}`);
  }
  return Buffer.from(await response.arrayBuffer());
}

async function fetchJSON(url) {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`GET ${url} failed: ${response.status}`);
  }
  return response.json();
}

async function postJSON(url, body) {
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    throw new Error(`POST ${url} failed: ${response.status}`);
  }
  return response.json();
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
