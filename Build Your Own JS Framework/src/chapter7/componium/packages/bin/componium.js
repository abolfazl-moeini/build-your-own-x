#!/usr/bin/env node
import yargs from "yargs/yargs";
import { initDev, restartDev, create } from "../dev/commands.js";
import { createRequire } from "node:module";

// Import the package.json file to get the version number by using the createRequire function
const require = createRequire(import.meta.url);
const { version } = require("../../package.json");

/**
 * The main CLI script for the Componium framework.
 * Utilizes yargs for argument parsing and command execution.
 */

// Set up yargs for command-line argument parsing
yargs(process.argv.slice(2))
  .scriptName("componium") // Name of the script as it would appear in help messages
  .usage("$0 <cmd> [args]") // Display usage string in help output

  // Init
  .command(
    "init",
    "initialize a new Componium app ",
    () => {},
    () => initDev()
  )
  // Command to watch for server changes and restart
  .command("dev", "watch for server changes and restart", () => {}, restartDev)

  // Command to create a new framework component
  .command("create", "create a framework component", () => {}, create)
  .version(version)
  .demandCommand(1, "")

  .help().argv; // Add a default help option // Parse the process arguments
