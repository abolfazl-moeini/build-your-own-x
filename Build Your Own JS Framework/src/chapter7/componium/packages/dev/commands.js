import chokidar from "chokidar";
import { spawn } from "node:child_process";
import initDev from "./init-dev.js";
import create from "./create.js";
// `makeDebug` is a function from the "debug" package for creating a debugger instance.
import makeDebug from "debug";

// Creating a debug instance specifically for "componium:frontend-client-view".
const debug = makeDebug("componium:commands");

// Variable to store the server process instance
let server;

/**
 * Kills the current server process (if running) and starts a new one.
 */
function restartServer() {
  // Check if a server process is currently running
  if (server) {
    // Kill the server process
    server.kill("SIGTERM");
  }

  // Start a new server process
  server = spawn("node", ["--no-deprecation", "app.js"], { stdio: "inherit" });

  // Event listener for when the server process closes
  server.on("close", function (code, signal) {
    if (signal) {
      debug(`Server process was killed with signal ${signal}`);
    } else if (code !== null) {
      console.log(`Server process exited with code ${code}`);
    } else {
      console.log("Server process exited");
    }
  });
}

/**
 * Watches for changes in the current directory and restarts the server
 * whenever a change is detected. Ignores changes in the 'node_modules' directory
 * and hidden files/folders.
 */
function restartDev() {
  const ignores = ["node_modules", /(^|[\/\\])\../];

  chokidar
    .watch(".", {
      ignored: ignores,
    })
    .on("all", (event, path) => {
      restartServer();
    });
}

// Export the functions for external use
export { restartDev, create, initDev };
