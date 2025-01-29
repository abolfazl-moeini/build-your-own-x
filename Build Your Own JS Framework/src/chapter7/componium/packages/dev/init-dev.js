import path from "path";
import fs from "fs/promises";
import { fileURLToPath } from "url";
import { exec } from "child_process";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

/**
 * Initialize app directory from template
 */
export default async function initDev(dirPath) {
  const templateDir = path.join(__dirname, "..", "template", "_app");
  const currentDir = dirPath || process.cwd();

  try {
    const currentDirContents = await fs.readdir(currentDir);

    if (currentDirContents.includes("package.json")) {
      throw new Error(
        "Directory contains package.json. Aborting to prevent overwriting existing project."
      );
    }

    if (currentDirContents.length > 0) {
      console.log(
        "Warning: New application directory not empty. New files will be added to existing ones."
      );
    }

    await copyDirectoryRecursive(templateDir, currentDir);
    // Install npm dependencies
    await installDependencies(currentDir);

    console.log(
      "New Componium application initialized. Run `componium dev` to start the new app or `componium create` to scaffold!"
    );
  } catch (error) {
    console.error("An error occurred:", error.message);
  }
}

async function copyDirectoryRecursive(source, target) {
  const files = await fs.readdir(source);

  for (const file of files) {
    const sourcePath = path.join(source, file);
    const destPath = path.join(target, file);

    const stat = await fs.stat(sourcePath);
    console.log(`Creating ${destPath}...`);

    if (stat.isDirectory()) {
      try {
        await fs.mkdir(destPath, { recursive: true });
      } catch (e) {}
      await copyDirectoryRecursive(sourcePath, destPath);
    } else {
      await fs.copyFile(sourcePath, destPath);
    }
  }
}

function installDependencies(dirPath) {
  return new Promise((resolve, reject) => {
    console.log("Installing dependencies...");

    exec("npm install", { cwd: dirPath }, (error, stdout, stderr) => {
      if (error) {
        console.error(`exec error: ${error}`);
        reject(error);
        return;
      }

      resolve();
    });
  });
}
