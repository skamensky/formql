const fs = require("fs");
const path = require("path");

async function main() {
  const distDir = path.join(__dirname, "dist");
  require(path.join(distDir, "wasm_exec.js"));

  const go = new Go();
  const wasmBytes = fs.readFileSync(path.join(distDir, "formql.wasm"));
  const { instance } = await WebAssembly.instantiate(wasmBytes, go.importObject);
  go.run(instance);

  await new Promise((resolve) => setTimeout(resolve, 0));

  if (!globalThis.FormQL) {
    throw new Error("FormQL global was not registered");
  }

  const catalogJSON = fs.readFileSync(
    path.join(__dirname, "..", "..", "examples", "catalogs", "rental-agency.formql.schema.json"),
    "utf8",
  );

  const infoResult = globalThis.FormQL.loadSchemaInfoJSON(catalogJSON, {
    baseTable: "rental_contract",
  });
  if (!infoResult.ok) {
    throw new Error(`schema info failed: ${infoResult.error.message}`);
  }
  if (infoResult.info.base_table !== "rental_contract") {
    throw new Error(`unexpected base table: ${infoResult.info.base_table}`);
  }

  const compileResult = globalThis.FormQL.compileCatalogJSON(
    catalogJSON,
    'customer_rel.email & " / " & STRING(actual_total)',
    {
      baseTable: "rental_contract",
      fieldAlias: "result",
    },
  );
  if (!compileResult.ok) {
    throw new Error(`compile failed: ${compileResult.error.message}`);
  }
  if (!compileResult.compilation.sql.query.includes('"rental_contract"')) {
    throw new Error("compiled SQL did not reference rental_contract");
  }

  const verifyResult = globalThis.FormQL.compileAndVerifyCatalogJSON(
    catalogJSON,
    "actual_total + 1",
    {
      baseTable: "rental_contract",
      fieldAlias: "result",
      verifyMode: "syntax",
    },
  );
  if (!verifyResult.ok) {
    throw new Error(`compile+verify failed: ${verifyResult.error.message}`);
  }
  if (verifyResult.verification.ok) {
    throw new Error("expected wasm verification to report unsupported status");
  }
  if (verifyResult.verification.diagnostics[0].code !== "verification_unavailable") {
    throw new Error(`unexpected verification diagnostic: ${verifyResult.verification.diagnostics[0].code}`);
  }

  const sqlResult = globalThis.FormQL.verifySQL("SELECT 1", { verifyMode: "syntax" });
  if (!sqlResult.ok) {
    throw new Error(`plain SQL verify call failed: ${sqlResult.error.message}`);
  }
  if (sqlResult.verification.ok) {
    throw new Error("expected plain SQL verification to be unavailable in wasm");
  }

  console.log("wasm smoke passed");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
