#!/usr/bin/env node
const { spawnSync } = require('child_process');
const path = require('path');
const os = require('os');

const exe = os.platform() === 'win32' ? 'repowalk.exe' : 'repowalk';
const binPath = path.join(__dirname, '..', 'vendor', exe);

// Forward all arguments to the Go binary
const result = spawnSync(binPath, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  console.error(`Failed to start repowalk: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status !== null ? result.status : 1);
