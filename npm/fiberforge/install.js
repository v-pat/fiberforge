const fs = require('fs');
const https = require('https');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

const version = '1.0.0';
const binName = os.platform() === 'win32' ? 'fiberforge.exe' : 'fiberforge';
const binDir = path.join(__dirname, 'bin');
const binPath = path.join(binDir, binName);

const PLATFORM_MAP = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows'
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64'
};

const platform = PLATFORM_MAP[os.platform()];
const arch = ARCH_MAP[os.arch()];

if (!platform || !arch) {
  console.error(`Unsupported platform/architecture: ${os.platform()}-${os.arch()}`);
  process.exit(1);
}

const ext = platform === 'windows' ? 'zip' : 'tar.gz';
const url = `https://github.com/v-pat/fiberforge/releases/download/v${version}/fiberforge_${version}_${platform}_${arch}.${ext}`;

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

console.log(`Downloading FiberForge v${version} for ${platform}-${arch}...`);

https.get(url, (res) => {
  if (res.statusCode !== 200 && res.statusCode !== 302) {
    console.error(`Failed to download binary: HTTP ${res.statusCode}`);
    process.exit(1);
  }
  
  // Follow redirect
  if (res.statusCode === 302) {
    https.get(res.headers.location, handleResponse);
  } else {
    handleResponse(res);
  }
}).on('error', (err) => {
  console.error('Download failed:', err.message);
  process.exit(1);
});

function handleResponse(res) {
  const archivePath = path.join(binDir, `archive.${ext}`);
  const fileStream = fs.createWriteStream(archivePath);
  
  res.pipe(fileStream);
  
  fileStream.on('finish', () => {
    fileStream.close();
    
    try {
      if (ext === 'zip') {
        execSync(`unzip -o -q "${archivePath}" -d "${binDir}"`);
      } else {
        execSync(`tar -xzf "${archivePath}" -C "${binDir}"`);
      }
      
      if (platform !== 'windows') {
        fs.chmodSync(binPath, 0o755);
      }
      
      fs.unlinkSync(archivePath);
      console.log('FiberForge installed successfully!');
    } catch (err) {
      console.error('Failed to extract archive:', err.message);
      process.exit(1);
    }
  });
}

