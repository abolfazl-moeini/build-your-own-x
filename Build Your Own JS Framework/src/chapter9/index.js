import select from "@inquirer/select";
import { execSync } from "child_process";

console.log("Welcome to Chapter 9!");

const dir = await select({
  message: "Select the demo to run:",
  choices: [
    {
      name: "Commitizen",
      value: "commitizen",
      description: "Project configured to use Commitizen",
    },
    {
      name: "Continuous Integration configuration examples",
      value: "ci",
      description: "Series of configuration scripts for Circle CI and GitHub",
    },
    {
      name: "Maintenance Tools",
      value: "maintenance-tools",
      description: "Series of maintenance tools to explore",
    },
  ],
});

try {
  console.log(`Navigating into ${dir}...`);
  process.chdir(dir);
  console.log(`Current directory: ${process.cwd()}`);

  console.log("Installing dependencies...");
  execSync("npm install", { stdio: "inherit" }); // show output in console

  console.log("Starting the project...");
  execSync("npm run dev", { stdio: "inherit" }); // show output in console
} catch (err) {
  console.error(`Error: ${err}`);
}
