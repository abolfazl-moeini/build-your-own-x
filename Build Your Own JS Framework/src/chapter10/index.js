import select from "@inquirer/select";
import { execSync } from "child_process";

console.log("Welcome to Chapter 10!");

const dir = await select({
  message: "Select the option to run:",
  choices: [
    {
      name: "AdonisJS",
      value: "adonisjs",
      description: "AdonisJS practical example",
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
