import select from "@inquirer/select";
import { execSync } from "child_process";

console.log("Welcome to Chapter 5!");

const answer = await select({
  message: "Select the demo to run:",
  choices: [
    {
      name: "Angular with RxJS",
      value: "angular-rxjs",
      description: "Utilizing the RxJS library in Angular",
    },
    {
      name: "esbuild tooling",
      value: "esbuild",
      description: "Build tooling and compilers",
    },
  ],
});

const dir = answer;

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
