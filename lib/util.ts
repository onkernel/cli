import archiver from 'archiver';
import fs from 'fs';
import fsExtra from 'fs-extra';
import walk from 'ignore-walk';
import path from 'path';
import type { PackageJson } from 'type-fest';

/**
 * Get the version from the package.json file.
 *
 * @returns Promise resolving to the package version string
 */
export function getPackageVersion(): string {
  const pkgJsonPath = path.join(__dirname, '..', '..', '..', 'package.json');
  const content = fsExtra.readJSONSync(pkgJsonPath) as PackageJson;
  if (!content.version) {
    throw new Error('package.json does not contain a version');
  }
  return content.version;
}

/**
 * Zips a directory into a file
 *
 * @param sourceDir Directory to zip
 * @param outPath Path to output zip file
 * @returns Promise that resolves when zip is complete
 */
export async function zipDirectory(inputDir: string, outputZip: string): Promise<void> {
  const entries = await walk({
    path: inputDir,
    ignoreFiles: ['.gitignore', '.dockerignore'],
    includeEmpty: true,
    follow: false,
  });

  const output = fs.createWriteStream(outputZip);
  const archive = archiver('zip', { zlib: { level: 9 } });

  const finalizePromise = new Promise<void>((resolve, reject) => {
    output.on('close', resolve);
    archive.on('error', reject);
  });

  archive.pipe(output);

  for (const entry of entries) {
    const fullPath = path.join(inputDir, entry);
    const stat = fs.statSync(fullPath);
    const archivePath = entry.split(path.sep).join('/'); // Normalize to Unix slashes

    if (stat.isFile()) {
      archive.file(fullPath, { name: archivePath });
    } else if (stat.isDirectory()) {
      archive.append('', { name: archivePath.endsWith('/') ? archivePath : archivePath + '/' });
    }
  }

  await archive.finalize();
  await finalizePromise;
}
