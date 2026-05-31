#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { cpSync, existsSync, mkdirSync, readdirSync, rmSync } from "node:fs";
import { join, resolve } from "node:path";

const root = resolve(new URL("..", import.meta.url).pathname);
const webDir = join(root, "apps", "web");
const webOutDir = join(webDir, "out");
const embedDir = join(root, "internal", "webassets", "static");
const distDir = join(root, "dist");

const defaultTargets = ["darwin/arm64", "linux/amd64", "windows/amd64"];
const targets = (process.env.MEMOIR_TARGETS || defaultTargets.join(","))
  .split(",")
  .map((target) => target.trim())
  .filter(Boolean);

try {
  run("npm", ["ci"], { cwd: webDir });
  run("npm", ["run", "build"], {
    cwd: webDir,
    env: {
      ...process.env,
      MEMOIR_STATIC_EXPORT: "1",
      NEXT_PUBLIC_API_BASE_URL: "",
    },
  });

  if (!existsSync(join(webOutDir, "index.html"))) {
    throw new Error(`Expected static export at ${webOutDir}/index.html`);
  }

  cleanEmbeddedAssets();
  cpSync(webOutDir, embedDir, { recursive: true });

  mkdirSync(distDir, { recursive: true });
  for (const target of targets) {
    const [goos, goarch] = target.split("/");
    if (!goos || !goarch) {
      throw new Error(`Invalid MEMOIR_TARGETS entry "${target}". Use goos/goarch.`);
    }
    const suffix = goos === "windows" ? ".exe" : "";
    const output = join(distDir, `memoir-${goos}-${goarch}${suffix}`);
    run("go", ["build", "-trimpath", "-ldflags=-s -w", "-o", output, "./cmd/memoir"], {
      cwd: root,
      env: {
        ...process.env,
        CGO_ENABLED: "0",
        GOOS: goos,
        GOARCH: goarch,
      },
    });
    console.log(`built ${output}`);
  }
} finally {
  cleanEmbeddedAssets();
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd || root,
    env: options.env || process.env,
    stdio: "inherit",
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(" ")} failed with exit code ${result.status}`);
  }
}

function cleanEmbeddedAssets() {
  mkdirSync(embedDir, { recursive: true });
  for (const entry of readdirSync(embedDir)) {
    if (entry === ".gitkeep" || entry === "placeholder.txt") {
      continue;
    }
    rmSync(join(embedDir, entry), { recursive: true, force: true });
  }
}
