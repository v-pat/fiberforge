#!/usr/bin/env node

const { spawnSync } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

const binName = os.platform() === 'win32' ? 'fiberforge.exe' : 'fiberforge';
const binPath = path.join(__dirname, 'bin', binName);

if (!fs.existsSync(binPath)) {
  console.error('FiberForge binary not found. Please reinstall the package.');
  process.exit(1);
}

const args = process.argv.slice(2);
const result = spawnSync(binPath, args, { stdio: 'inherit' });

if (result.error) {
  console.error(`Failed to start FiberForge: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status);

