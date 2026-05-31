import { readFile, readdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const webRoot = path.join(repoRoot, "apps", "web");
const sourceRoots = [path.join(webRoot, "app"), path.join(webRoot, "components"), path.join(webRoot, "lib")];
const nativeDialogPattern = /\b(?:window\.)?(alert|confirm|prompt)\s*\(/g;

const files = [];
for (const root of sourceRoots) {
  await collectSourceFiles(root, files);
}

const violations = [];
for (const file of files) {
  const source = await readFile(file, "utf8");
  for (const match of source.matchAll(nativeDialogPattern)) {
    violations.push(`${path.relative(repoRoot, file)} uses ${match[1]}()`);
  }
}

assertNoViolations(violations, "Native browser dialogs are not allowed in the Memoir web UI.");

const confirmDialogPath = path.join(webRoot, "components", "ConfirmDialog.tsx");
const dialogShellPath = path.join(webRoot, "components", "DialogShell.tsx");
const confirmDialog = await readFile(confirmDialogPath, "utf8");
const dialogShell = await readFile(dialogShellPath, "utf8");

const requiredConfirmDialogSnippets = [
  "export type ConfirmDialogTone",
  '"warning"',
  '"danger"',
  "DialogShell",
  'role="alertdialog"',
  "pendingLabel",
  "confirm-modal-error",
];
const requiredDialogShellSnippets = [
  "createPortal",
  "closeOnBackdrop",
  "closeOnEscape",
  'event.key === "Escape"',
  'event.key !== "Tab"',
  "lockBodyScroll",
];

assertContains(confirmDialog, requiredConfirmDialogSnippets, "ConfirmDialog is missing required behavior.");
assertContains(dialogShell, requiredDialogShellSnippets, "DialogShell is missing required modal behavior.");

console.log(`Dialog policy check passed for ${files.length} frontend source files.`);

async function collectSourceFiles(dir, output) {
  const entries = await readdir(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if ([".next", "node_modules", "out"].includes(entry.name)) {
        continue;
      }
      await collectSourceFiles(fullPath, output);
      continue;
    }
    if (/\.(tsx?|jsx?)$/.test(entry.name)) {
      output.push(fullPath);
    }
  }
}

function assertContains(source, snippets, message) {
  const missing = snippets.filter((snippet) => !source.includes(snippet));
  assertNoViolations(missing.map((snippet) => `missing ${snippet}`), message);
}

function assertNoViolations(violations, message) {
  if (violations.length === 0) {
    return;
  }
  console.error(message);
  for (const violation of violations) {
    console.error(`- ${violation}`);
  }
  process.exit(1);
}
