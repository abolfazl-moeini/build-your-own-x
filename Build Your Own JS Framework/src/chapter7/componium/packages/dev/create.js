import input from "@inquirer/input";
import select from "@inquirer/select";
import fs from "fs/promises";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

/**
 * Interactively allows the user to choose a scaffold type and
 * then performs corresponding actions.
 */
export default async function create() {
  const componentTemplate = await select({
    message: "What would you like to scaffold?",
    choices: [
      {
        name: "Model",
        value: "model",
        description: "Create a new database model",
      },
      {
        name: "Route",
        value: "route",
        description: "Create a new route file",
      },
      {
        name: "GraphQL Schema",
        value: "gql",
      },
      {
        name: "Client View",
        value: "clientview",
      },
    ],
  });

  const name = await input({
    message: `Enter a name for your ${componentTemplate.toUpperCase()}`,
  });

  const templateDir = path.join(
    __dirname,
    "..",
    "template",
    "_create",
    `${componentTemplate}.js`
  );
  const sourcePath = path.resolve(templateDir);
  const currentDir = process.cwd();

  let destDir = path.join(currentDir, "models");
  switch (componentTemplate) {
    case "gql":
      destDir = path.join(currentDir, "routes", "gql");
      break;
    case "route":
      destDir = path.join(currentDir, "routes", "api");
      break;
    case "clientview":
      destDir = path.join(currentDir, "views");
      break;
  }

  try {
    await fs.mkdir(destDir, { recursive: true });
  } catch (e) {
    throw new Error(e);
  }

  const destPath = path.resolve(path.join(destDir, `${toSnakeCase(name)}.js`));
  await fs.copyFile(sourcePath, destPath);
  console.log(`Created ${destPath}!`);
}

function toSnakeCase(str) {
  return str
    .replace(/[\s_]+/g, "_")
    .replace(/([a-z\d_])([A-Z])/g, "$1_$2")
    .toLowerCase();
}
