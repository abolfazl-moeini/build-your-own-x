import CliServer from "./server.js";
import puppeteer from "puppeteer";
import { ComponiumStatus } from "../util/env.js";

/**
 * Browser test executor using Puppeteer.
 */
class BrowserExecutor {
  /**
   * Create executor instance.
   *
   * @param {Object[]} targets - Targets to test.
   * @param {Object} [options] - Options.
   * @param {boolean} [options.keepAlive=false] - Keep browser open after test.
   */
  constructor(targets, options = {}) {
    // Targets to execute
    this.targets = targets;
    // Options object
    this.options = options;
    // Puppeteer options
    this.puppeteerOptions = {
      headless: options.keepAlive ? false : "new",
      devtools: true,
    };
  }
  /**
   * Run the browser test.
   *
   * @returns {Promise<Object>} Test result status.
   */
  async execute() {
    // Create and start the Componium Test server
    const cliServer = new CliServer(this.targets);
    const server = await cliServer.listen();

    // Launch Puppeteer browser
    const browser = await puppeteer.launch(this.puppeteerOptions);
    const page = await browser.newPage();

    // Handle test completion
    return new Promise(async (resolve) => {
      page.on("console", async (message) => {
        // Handle pass/fail logging
        if (
          message.text() === ComponiumStatus.Pass ||
          message.text() === ComponiumStatus.Fail
        ) {
          // Close resources if not keeping alive
          if (!this.options.keepAlive) {
            await browser.close();
            await cliServer.close();
          }

          if (message.text() === ComponiumStatus.Fail) {
            return resolve({
              status: ComponiumStatus.Fail,
              keepAlive: this.options.keepAlive,
            });
          } else {
            return resolve({
              status: ComponiumStatus.Pass,
              keepAlive: this.options.keepAlive,
            });
          }
        }
        console.log(message.text());
      });
      // Navigate browser to CLI server
      await page.goto(`http://127.0.0.1:${server.port.port}`);
    });
  }
}

export default BrowserExecutor;
