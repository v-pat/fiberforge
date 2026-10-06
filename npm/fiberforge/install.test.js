const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const path = require('path');
const os = require('os');
const crypto = require('crypto');
const {
  getPlatformAndArch,
  parseChecksum,
  verifyChecksum,
  install
} = require('./install');

test('getPlatformAndArch maps supported platforms correctly', () => {
  const darwinArm = getPlatformAndArch('darwin', 'arm64');
  assert.equal(darwinArm.platform, 'darwin');
  assert.equal(darwinArm.arch, 'arm64');

  const linuxX64 = getPlatformAndArch('linux', 'x64');
  assert.equal(linuxX64.platform, 'linux');
  assert.equal(linuxX64.arch, 'amd64');

  const winX64 = getPlatformAndArch('win32', 'x64');
  assert.equal(winX64.platform, 'windows');
  assert.equal(winX64.arch, 'amd64');
});

test('getPlatformAndArch throws on unsupported platform/architecture', () => {
  assert.throws(() => getPlatformAndArch('freebsd', 'x64'), /Unsupported/);
  assert.throws(() => getPlatformAndArch('linux', 's390x'), /Unsupported/);
});

test('parseChecksum parses GoReleaser checksums format correctly', () => {
  const sampleChecksums = `
# Checksums for fiberforge
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  fiberforge_1.0.2_darwin_arm64.tar.gz
d41d8cd98f00b204e9800998ecf8427e00000000000000000000000000000000  fiberforge_1.0.2_linux_amd64.tar.gz
`;

  const hash = parseChecksum(sampleChecksums, 'fiberforge_1.0.2_darwin_arm64.tar.gz');
  assert.equal(hash, 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855');

  assert.throws(
    () => parseChecksum(sampleChecksums, 'fiberforge_1.0.2_windows_amd64.zip'),
    /Checksum for .* not found/
  );
});

test('verifyChecksum passes on matching SHA-256', () => {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ff-test-'));
  const testFile = path.join(tmpDir, 'test.bin');
  const content = 'FiberForge binary content test';
  fs.writeFileSync(testFile, content);

  const expectedHash = crypto.createHash('sha256').update(content).digest('hex');
  assert.equal(verifyChecksum(testFile, expectedHash), true);

  fs.rmSync(tmpDir, { recursive: true, force: true });
});

test('verifyChecksum rejects mismatched SHA-256 and throws error', () => {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ff-test-'));
  const testFile = path.join(tmpDir, 'tampered.bin');
  fs.writeFileSync(testFile, 'Tampered content');

  assert.throws(
    () => verifyChecksum(testFile, '0000000000000000000000000000000000000000000000000000000000000000'),
    /SHA-256 checksum mismatch/
  );

  fs.rmSync(tmpDir, { recursive: true, force: true });
});

test('install validates checksum and refuses corrupted/mismatched archive', async () => {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ff-install-test-'));
  const fakePkgPath = path.join(tmpDir, 'package.json');
  fs.writeFileSync(fakePkgPath, JSON.stringify({ version: '1.0.2' }));

  const binDir = path.join(tmpDir, 'bin');
  const fakeArchiveName = 'fiberforge_1.0.2_darwin_arm64.tar.gz';
  const fakeChecksums = `badhash00000000000000000000000000000000000000000000000000000000  ${fakeArchiveName}\n`;

  await assert.rejects(
    () =>
      install({
        pkgPath: fakePkgPath,
        binDir,
        platform: 'darwin',
        arch: 'arm64',
        fetchChecksums: async () => fakeChecksums,
        downloadArchive: async (url, dest) => {
          fs.writeFileSync(dest, 'corrupt or malicious payload');
        }
      }),
    /SHA-256 checksum mismatch/
  );

  // Ensure corrupt archive was deleted and no binary was extracted
  assert.equal(fs.existsSync(path.join(binDir, `temp_${fakeArchiveName}`)), false);
  assert.equal(fs.existsSync(path.join(binDir, 'fiberforge')), false);

  fs.rmSync(tmpDir, { recursive: true, force: true });
});
