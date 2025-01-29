import fs from "fs";
import select from "@inquirer/select";
import { execSync } from "child_process";

const sources = {
  "./node_modules/@angular/core": "angular",
  "./node_modules/vue/dist": "vue",
  "./node_modules/@sveltejs/adapter-node": "svelte-adapter-node",
};

console.log("Welcome to Chapter 3! \n");

console.log("Extracting packaged framework sources...");
Object.keys(sources).forEach((key) => {
  fs.cpSync(key, sources[key], { recursive: true });
});

console.log(
  "Done! You can view the sources in this directory using your code editor..."
);

const answer = await select({
  message: "Select the demo to run:",
  choices: [
    {
      name: "Next.js Example App",
      value: "next",
      description:
        "Example of basic Next.js app built using the Next.jsframework",
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
