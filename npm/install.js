const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

const version = require('./package.json').version;
const platform = os.platform();
const arch = os.arch();

// Map Node.js os.platform() and os.arch() to GoReleaser naming convention
const osMap = {
  win32: 'Windows',
  darwin: 'Darwin',
  linux: 'Linux'
};

const archMap = {
  x64: 'x86_64',
  arm64: 'arm64',
  ia32: 'i386'
};

const goOs = osMap[platform];
const goArch = archMap[arch];

if (!goOs || !goArch) {
  console.error(`Unsupported platform/architecture: ${platform}/${arch}`);
  process.exit(1);
}

const fileName = `repowalk_${goOs}_${goArch}.tar.gz`;
const downloadUrl = `https://github.com/QubeUtils/repowalk/releases/download/v${version}/${fileName}`;
const vendorDir = path.join(__dirname, 'vendor');
const tarPath = path.join(vendorDir, fileName);

if (!fs.existsSync(vendorDir)) {
  fs.mkdirSync(vendorDir, { recursive: true });
}

console.log(`Downloading repowalk v${version} for ${goOs}_${goArch}...`);

function download(url, dest) {
  return new Promise((resolve, reject) => {
    https.get(url, (response) => {
      // Handle redirects
      if (response.statusCode === 301 || response.statusCode === 302) {
        return download(response.headers.location, dest).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        return reject(new Error(`Failed to download: ${response.statusCode} - ${response.statusMessage}`));
      }

      const file = fs.createWriteStream(dest);
      response.pipe(file);
      file.on('finish', () => {
        file.close();
        resolve();
      });
    }).on('error', (err) => {
      fs.unlink(dest, () => reject(err));
    });
  });
}

download(downloadUrl, tarPath)
  .then(() => {
    console.log('Extracting archive...');
    // We unified all GoReleaser outputs to tar.gz, so we can use tar directly!
    execSync(`tar -xzf "${fileName}"`, { cwd: vendorDir, stdio: 'inherit' });
    
    // Cleanup tarball
    fs.unlinkSync(tarPath);
    console.log('Repowalk installed successfully!');
  })
  .catch((err) => {
    console.error('Installation failed:', err.message);
    console.error('You can try installing manually from GitHub Releases.');
    process.exit(1);
  });
