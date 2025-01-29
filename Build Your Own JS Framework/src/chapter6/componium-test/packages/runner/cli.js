import makeDebug from "debug";
const debug = makeDebug("runner");
import { glob } from "glob";
import path from "node:path";
import { Enviroments as Environments, getExitStatus } from "../util/env.js";
import NodeExecutor from "../node/executor.js";
import BrowserExecutor from "../browser/executor.js";

/**
 * Execute the test runner given a directory of tests
 *
 * @param {string[]} targetDirectory - The directories containing the test files to run.
 * @param {string} environment - The environment in which to run the tests (either Node or Browser).
 * @param {object} options - Additional options for the executor.
 * @returns {Promise<void>}
 */
async function runner(targetDirectory = ["tests"], environment, options) {
  debug("Calling runner: ", targetDirectory, environment);
  // currently we support one directory at a time
  const testFiles = await getTestFiles(targetDirectory[0]);
  if (testFiles.length === 0) {
    console.log("No tests found...");
    return process.exit(1);
  }

  // Create an executor based on the environment
  const executor = createExecutor(environment, testFiles, options);
  const result = await executor.execute();
  debug("result", result);
  // Determine the exit status based on the test result
  const status = getExitStatus(result);
  if (!result.keepAlive) {
    process.exit(status);
  }
}

async function getTestFiles(directory) {
  const globResult = await glob(`${directory}/**/*.js`, { ignore: "node_modules/**" });
  return globResult;
}

function createExecutor(environment, testFiles, options) {
  if (environment === Environments.Node) {
    return new NodeExecutor(testFiles, options);
  } else {
    return new BrowserExecutor(testFiles, options);
  }
}

export default runner;
