import { execSync } from "child_process";

console.log("Welcome to Chapter 6! \n\n");
console.log("Installing Componium Test framework...\n\n");

const dir = "componium-test";

try {
  console.log(`Navigating into ${dir}...\n\n`);
  process.chdir(dir);

  console.log("Installing dependencies...\n\n");
  execSync("npm install", { stdio: "inherit" }); // show output in console

  console.log("Running tests for the project...\n\n");
  execSync("npm run test", { stdio: "inherit" }); // show output in console
} catch (err) {
  console.error(`Error: ${err}`);
}
