import test from "ava";
import { initDev } from "../../../packages/dev/commands.js";
import fs, { rm } from "fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

test("initDev copies template files", async (t) => {
  t.timeout(60 * 1000);
  const tmpPath = path.join(__dirname, "_tmp");
  await rm(tmpPath, { recursive: true });
  await fs.mkdir(tmpPath);
  await initDev(tmpPath);

  async function checkFilesExist(files) {
    const missing = [];

    for (const file of files) {
      try {
        await fs.access(path.join(tmpPath, file));
      } catch {
        console.log(file);
        missing.push(file);
      }
    }

    return missing;
  }
  const filesToCheck = [
    "app.js",
    "package.json",
    "config/default.json",
    "node_modules",
  ];
  const missingFiles = await checkFilesExist(filesToCheck);
  t.is(missingFiles.length, 0);
});
