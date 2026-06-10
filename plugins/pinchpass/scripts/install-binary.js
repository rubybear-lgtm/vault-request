#!/usr/bin/env node

import { execSync } from "child_process";
import { existsSync, mkdirSync, writeFileSync, chmodSync } from "fs";
import { homedir, platform } from "os";

const VERSION = "latest";
const REPO = "rubybear-lgtm/PinchPass";

function detectBinary() {
  const arch = process.arch === "arm64" ? "arm64" : "amd64";
  const osMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  };
  const plat = osMap[process.platform];
  if (!plat) throw new Error(`Unsupported platform: ${process.platform}`);
  return { name: `pinchpass-${plat}-${arch}`, plat, arch };
}

async function install() {
  const { name } = detectBinary();
  const installDir = `${homedir()}/.openclaw/bin`;
  const dest = `${installDir}/pinchpass`;

  if (existsSync(dest)) {
    console.log(`pinchpass already installed at ${dest}`);
    return;
  }

  const url =
    VERSION === "latest"
      ? `https://github.com/${REPO}/releases/latest/download/${name}`
      : `https://github.com/${REPO}/releases/download/v${VERSION}/${name}`;

  console.log(`Downloading pinchpass from ${url}...`);

  mkdirSync(installDir, { recursive: true });

  if (process.platform === "win32") {
    execSync(`curl -sL "${url}" -o "${dest}.exe"`, { stdio: "inherit" });
  } else {
    execSync(`curl -sL "${url}" -o "${dest}"`, { stdio: "inherit" });
    chmodSync(dest, 0o755);
  }

  console.log(`Installed pinchpass to ${dest}`);
  console.log(`Add ${installDir} to your PATH if not already present.`);
}

install().catch((err) => {
  console.error(`Failed to install pinchpass binary: ${err.message}`);
  console.log(
    `To build from source: cd pinchpass-plugin && go build -o pinchpass .`
  );
  process.exit(1);
});
