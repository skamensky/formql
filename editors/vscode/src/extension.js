const path = require("path");
const vscode = require("vscode");
const { LanguageClient } = require("vscode-languageclient/node");

let client;
let outputChannel;
let warningShown = false;

async function activate(context) {
  outputChannel = vscode.window.createOutputChannel("FormQL");
  context.subscriptions.push(outputChannel);

  context.subscriptions.push(
    vscode.commands.registerCommand("formql.restartLanguageServer", async () => {
      await restartClient(context);
    }),
  );

  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration(async (event) => {
      if (event.affectsConfiguration("formql")) {
        await restartClient(context);
      }
    }),
  );

  await startClient(context);
}

async function deactivate() {
  if (!client) {
    return undefined;
  }
  const running = client;
  client = undefined;
  return running.stop();
}

async function restartClient(context) {
  await stopClient();
  await startClient(context);
}

async function stopClient() {
  if (!client) {
    return;
  }
  const running = client;
  client = undefined;
  await running.stop();
}

async function startClient(context) {
  const workspaceFolder = primaryWorkspaceFolder();
  const config = vscode.workspace.getConfiguration("formql");

  const serverPath = resolveConfiguredPath(config.get("serverPath", "formqlc"), workspaceFolder);
  const serverArgs = resolveConfiguredArgs(config.get("serverArgs", []), workspaceFolder);
  const schemaPath = resolveConfiguredPath(config.get("schemaPath", ""), workspaceFolder);
  const databaseUrl = resolveConfiguredValue(config.get("databaseUrl", ""), workspaceFolder);

  if (!schemaPath && !databaseUrl) {
    if (!warningShown) {
      warningShown = true;
      vscode.window.showWarningMessage(
        "FormQL language server is idle. Configure formql.schemaPath for offline mode or formql.databaseUrl for live DB mode.",
      );
    }
    return;
  }
  warningShown = false;

  const commandArgs = [...serverArgs, "lsp"];
  if (schemaPath) {
    commandArgs.push("-schema", schemaPath);
  }
  if (databaseUrl) {
    commandArgs.push("-database-url", databaseUrl);
  }

  const serverOptions = {
    command: serverPath,
    args: commandArgs,
    options: {
      cwd: workspaceFolder ? workspaceFolder.uri.fsPath : undefined,
    },
  };

  const clientOptions = {
    documentSelector: [{ scheme: "file", language: "formql" }],
    initializationOptions: {},
    outputChannel,
  };

  client = new LanguageClient(
    "formql",
    "FormQL Language Server",
    serverOptions,
    clientOptions,
  );

  context.subscriptions.push(client.start());
  await client.onReady();
}

function primaryWorkspaceFolder() {
  if (!vscode.workspace.workspaceFolders || vscode.workspace.workspaceFolders.length === 0) {
    return undefined;
  }
  return vscode.workspace.workspaceFolders[0];
}

function resolveConfiguredArgs(values, workspaceFolder) {
  if (!Array.isArray(values)) {
    return [];
  }
  return values.map((value) => resolveConfiguredValue(String(value), workspaceFolder, true));
}

function resolveConfiguredPath(value, workspaceFolder) {
  const resolved = resolveConfiguredValue(value, workspaceFolder, true);
  if (!resolved) {
    return resolved;
  }
  if (!workspaceFolder || path.isAbsolute(resolved) || !looksLikePath(resolved)) {
    return resolved;
  }
  return path.join(workspaceFolder.uri.fsPath, resolved);
}

function resolveConfiguredValue(value, workspaceFolder, allowRelativePaths = false) {
  if (typeof value !== "string" || value.length === 0) {
    return "";
  }

  let resolved = value;
  if (workspaceFolder) {
    resolved = resolved.replace(/\$\{workspaceFolder\}/g, workspaceFolder.uri.fsPath);
    resolved = resolved.replace(
      /\$\{workspaceFolderBasename\}/g,
      path.basename(workspaceFolder.uri.fsPath),
    );
  }

  if (allowRelativePaths && workspaceFolder && looksLikePath(resolved) && !path.isAbsolute(resolved)) {
    return path.join(workspaceFolder.uri.fsPath, resolved);
  }

  return resolved;
}

function looksLikePath(value) {
  return value.startsWith(".") || value.includes("/") || value.includes(path.sep);
}

module.exports = {
  activate,
  deactivate,
};
