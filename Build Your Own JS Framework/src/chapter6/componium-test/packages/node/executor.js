import { Worker } from "node:worker_threads";
import { ComponiumStatus } from "../util/env.js";
import makeDebug from "debug";
import util from "util";

// Debug logger
const debug = makeDebug("NodeExecutor");

/**
 * Node.js test executor using worker threads.
 */
class NodeExecutor {
  /**
   * Create executor instance.
   *
   * @param {string[]} targets - Test targets.
   */
  constructor(targets) {
    this.targets = targets;
    debug("NodeExecutor targets:", targets);
  }

  /**
   * Execute tests.
   *
   * @returns {Promise<Object>} Test result status.
   */
  async execute() {
    let testSuites = this.targets.length;
    let failed = false;
    console.log("Testing in Node.js", process.version);

    return new Promise((resolve) => {
      const workers = [];

      this.targets.forEach((target) => {
        // Create the worker
        const worker = new Worker("./" + target);

        // Listen for messages and wrap console
        worker.on("message", (msg) => {
          if (msg === ComponiumStatus.Fail) {
            failed = true;
          }

          if (msg === ComponiumStatus.Pass || msg === ComponiumStatus.Fail) {
            testSuites--;
          }
        });

        workers.push(worker);
      });

      // Wait for all workers to finish
      Promise.all(
        workers.map((w) => new Promise((res) => w.once("exit", res)))
      ).then(() => {
        if (testSuites === 0) {
          if (failed) {
            return resolve({
              status: ComponiumStatus.Fail,
            });
          } else {
            return resolve({
              status: ComponiumStatus.Pass,
            });
          }
        }
      });
    });
  }
}

export default NodeExecutor;
