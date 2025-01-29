import { assert, should, expect } from "../assert/lib.js";
import { spy, mock, stub, replace, fake, restore } from "../mock/lib.js";
import { ComponiumStatus, getExitStatus, isBrowser } from "../util/env.js";
let Emit;
// Parent port for workers
let parentPort;

// Suite methods to filter
const INTERNAL_SUITE_METHODS = [
  "describe",
  "before",
  "beforeEach",
  "afterEach",
  "after",
];

// Get emitter based on environment
if (isBrowser()) {
  Emit = await import("../browser/emit.js");
} else {
  Emit = await import("../node/emit.js");
  let workerThreads = await import("node:worker_threads");
  parentPort = workerThreads.parentPort;
}

let testSuites = 0;
let totalTestsFailed = 0;
// Global emitter
let globalEmitter = new Emit.default();
// Handle test run completion
globalEmitter.on("suitesDone", (report) => {
  let result = ComponiumStatus.Pass;
  if (report.totalTestsFailed > 0) {
    console.log(`${report.totalTestsFailed} test suite(s) failed`);
    result = ComponiumStatus.Fail;
  }
  if (isBrowser()) {
    console.log(result);
  } else {
    if (parentPort) {
      parentPort.postMessage(result);
    } else {
      process.exit(getExitStatus(result));
    }
  }
});

/**
 * Componium test runner class.
 */
class ComponiumTest {
  constructor() {
    this.emitter = new Emit.default();
    this.test = this.test.bind(this);

    this.emitter.on("passed", (name, time) => {
      console.log("PASS:", name, time.toFixed(2), "ms");
    });

    this.emitter.on("failed", (name, e) => {
      console.log("FAIL:", name);
      if (isBrowser()) {
        console.warn(e.stack.toString());
      } else {
        console.warn(e);
      }
    });

    this.emitter.on("done", (report) => {
      if (report.failed > 0) {
        console.log(`${report.failed} out of ${report.total} tests failed`);
      }
    });
  }
  /**
   * Main test function that accepts an object of unit tests.
   *
   * @param {{}} suiteObject
   */
  async test(suiteObject) {
    testSuites++;
    let failed = 0;
    let start;
    let end;
    let errorOccurred = false;
    const tests = Object.keys(suiteObject)
      .filter((key) => !INTERNAL_SUITE_METHODS.includes(key))
      .reduce((obj, key) => {
        obj[key] = suiteObject[key];
        return obj;
      }, {});
    const testsTotal = Object.keys(tests).length;
    if (suiteObject.hasOwnProperty("describe")) {
      console.log(`\nSUITE: ${suiteObject.describe}`);
    } else {
      console.log(`\nSUITE: Unnamed suite`);
    }
    if (suiteObject.hasOwnProperty("before")) {
      await suiteObject.before();
    }
    for (const [key, fn] of Object.entries(tests)) {
      try {
        start = performance.now();
        if (suiteObject.hasOwnProperty("beforeEach")) {
          await suiteObject.beforeEach();
        }
        await fn();
      } catch (e) {
        this.emitter.emit("failed", key, e);
        errorOccurred = true;
        failed++;
        totalTestsFailed++;
      } finally {
        if (suiteObject.hasOwnProperty("afterEach")) {
          suiteObject.afterEach();
        }
        end = performance.now();
        if (!errorOccurred) {
          this.emitter.emit("passed", key, end - start);
        }
        errorOccurred = null;
      }
    }
    if (INTERNAL_SUITE_METHODS.hasOwnProperty("after")) {
      INTERNAL_SUITE_METHODS.after();
    }
    this.emitter.emit("done", {
      total: testsTotal,
      failed: failed,
    });
    testSuites--;
    if (testSuites === 0) {
      globalEmitter.emit("suitesDone", {
        totalTestsFailed: totalTestsFailed,
        totalTestsPassed: testsTotal - totalTestsFailed,
      });
    }
  }
}

let ct = new ComponiumTest();

export default ct.test;
export { assert, should, expect, mock, fake, stub, spy, replace, restore };
