import select from "@inquirer/select";
import { execSync } from "child_process";

console.log("Welcome to Chapter 4!");

const answer = await select({
  message: "Select the demo to run:",
  choices: [
    {
      name: "Next.js with Tailwind CSS",
      value: "next-example-app",
      description: "Example of using Tailwind CSS with the Next.js framework",
    },
    {
      name: "Practical Docus.dev example",
      value: "practical-docus",
      description:
        "Practical sample of using Docus to enhance the quality of documentation in framework projects",
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
