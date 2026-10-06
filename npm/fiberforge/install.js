const fs = require('fs');
const https = require('https');
const path = require('path');
const os = require('os');
const crypto = require('crypto');
const { execSync } = require('child_process');

const PLATFORM_MAP = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows'
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64'
};

function getPlatformAndArch(platform = os.platform(), arch = os.arch()) {
  const mappedPlatform = PLATFORM_MAP[platform];
  const mappedArch = ARCH_MAP[arch];
  if (!mappedPlatform || !mappedArch) {
    throw new Error(`Unsupported platform/architecture: ${platform}-${arch}`);
  }
  return { platform: mappedPlatform, arch: mappedArch };
}

function parseChecksum(checksumsContent, filename) {
  if (!checksumsContent || typeof checksumsContent !== 'string') {
    throw new Error('Checksums file is empty or invalid');
  }
  const lines = checksumsContent.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const parts = trimmed.split(/\s+/);
    if (parts.length >= 2) {
      const hash = parts[0];
      const name = parts[1].replace(/^\*/, ''); // strip optional binary mode indicator
      if (name === filename) {
        return hash;
      }
    }
  }
  throw new Error(`Checksum for ${filename} not found in checksums file`);
}

function verifyChecksum(filePath, expectedSha256) {
  if (!expectedSha256) {
    throw new Error('Expected SHA-256 checksum is required');
  }
  const fileBuffer = fs.readFileSync(filePath);
  const actualHash = crypto.createHash('sha256').update(fileBuffer).digest('hex');
  if (actualHash.toLowerCase() !== expectedSha256.toLowerCase()) {
    throw new Error(
      `SHA-256 checksum mismatch for ${path.basename(filePath)}:\nExpected: ${expectedSha256}\nActual:   ${actualHash}`
    );
  }
  return true;
}

function fetchWithRedirect(url, maxRedirects = 5) {
  return new Promise((resolve, reject) => {
    if (maxRedirects < 0) {
      return reject(new Error('Too many HTTP redirects'));
    }
    const parsedUrl = new URL(url);
    if (parsedUrl.protocol !== 'https:' && parsedUrl.protocol !== 'http:') {
      return reject(new Error(`Untrusted protocol: ${parsedUrl.protocol}`));
    }
    const client = parsedUrl.protocol === 'https:' ? https : require('http');

    client.get(url, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        const nextUrl = new URL(res.headers.location, url).toString();
        return resolve(fetchWithRedirect(nextUrl, maxRedirects - 1));
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`HTTP ${res.statusCode} downloading ${url}`));
      }
      resolve(res);
    }).on('error', reject);
  });
}

async function downloadText(url) {
  const res = await fetchWithRedirect(url);
  return new Promise((resolve, reject) => {
    let data = '';
    res.setEncoding('utf8');
    res.on('data', (chunk) => { data += chunk; });
    res.on('end', () => resolve(data));
    res.on('error', reject);
  });
}

async function downloadToFile(url, destPath) {
  const res = await fetchWithRedirect(url);
  return new Promise((resolve, reject) => {
    const fileStream = fs.createWriteStream(destPath);
    res.pipe(fileStream);
    fileStream.on('finish', () => {
      fileStream.close(() => resolve(destPath));
    });
    fileStream.on('error', (err) => {
      fs.unlink(destPath, () => reject(err));
    });
  });
}

function extractArchive(archivePath, destDir, ext) {
  fs.mkdirSync(destDir, { recursive: true });
  try {
    if (ext === 'zip') {
      execSync(`unzip -o -q "${archivePath}" -d "${destDir}"`, { stdio: 'pipe' });
    } else {
      execSync(`tar -xzf "${archivePath}" -C "${destDir}"`, { stdio: 'pipe' });
    }
  } catch (err) {
    throw new Error(`Failed to extract archive: ${err.message}`);
  }
}

async function install(options = {}) {
  const pkgPath = options.pkgPath || path.join(__dirname, 'package.json');
  const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
  const version = options.version || process.env.FIBERFORGE_VERSION || pkg.version;

  const { platform, arch } = getPlatformAndArch(options.platform, options.arch);
  const ext = platform === 'windows' ? 'zip' : 'tar.gz';
  const archiveName = `fiberforge_${version}_${platform}_${arch}.${ext}`;
  const binDir = options.binDir || path.join(__dirname, 'bin');
  const binName = platform === 'windows' ? 'fiberforge.exe' : 'fiberforge';
  const binPath = path.join(binDir, binName);

  const baseUrl = options.baseUrl || process.env.FIBERFORGE_RELEASE_URL || `https://github.com/v-pat/fiberforge/releases/download/v${version}`;
  const archiveUrl = `${baseUrl}/${archiveName}`;
  const checksumUrl = `${baseUrl}/checksums.txt`;

  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  console.log(`Downloading checksums for FiberForge v${version}...`);
  const checksumsContent = await (options.fetchChecksums
    ? options.fetchChecksums(checksumUrl)
    : downloadText(checksumUrl));

  const expectedChecksum = parseChecksum(checksumsContent, archiveName);

  console.log(`Downloading FiberForge v${version} for ${platform}-${arch}...`);
  const archivePath = path.join(binDir, `temp_${archiveName}`);

  try {
    if (options.downloadArchive) {
      await options.downloadArchive(archiveUrl, archivePath);
    } else {
      await downloadToFile(archiveUrl, archivePath);
    }

    // Verify SHA-256 before extraction
    verifyChecksum(archivePath, expectedChecksum);

    // Extract archive
    extractArchive(archivePath, binDir, ext);

    if (platform !== 'windows' && fs.existsSync(binPath)) {
      fs.chmodSync(binPath, 0o755);
    }

    console.log('FiberForge installed successfully!');
    return binPath;
  } finally {
    if (fs.existsSync(archivePath)) {
      try { fs.unlinkSync(archivePath); } catch (_) {}
    }
  }
}

if (require.main === module) {
  install().catch((err) => {
    console.error('Installation failed:', err.message);
    process.exit(1);
  });
}

module.exports = {
  getPlatformAndArch,
  parseChecksum,
  verifyChecksum,
  extractArchive,
  install
};
