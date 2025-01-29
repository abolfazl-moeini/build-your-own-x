import http from "node:http";
import fs from "node:fs";
import path, { dirname } from "node:path";
import render from "./render.js";

import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// select a random port
const port = 0;

/**
 * Componium CLI test server using Node.js HTTP server.
 */
class CliServer {
  /**
   * Create CLI server instance.
   *
   * @param {string[]} testFiles - Test files to serve.
   */
  constructor(testFiles) {
    this.html = render(testFiles);
    this.server = http.createServer((req, res) => {
      const url = req.url;
      res.statusCode = 200;

      let response;
      res.setHeader("Content-Type", "application/javascript; charset=utf-8");

      switch (url) {
        case "/":
          // special handler for the test runner
          res.setHeader("Content-Type", "text/html; charset=utf-8");
          response = this.html;
          break;
        case "/chai.js":
          // special handler for the chai assertion library
          response = fs.readFileSync(
            path.join(__dirname, "..", "..", "node_modules", "chai", "chai.js")
          );
          break;
        case "/sinon.js":
          // special handler for the sinon mocking library
          response = fs.readFileSync(
            path.join(
              __dirname,
              "..",
              "..",
              "node_modules",
              "sinon",
              "pkg",
              "sinon-esm.js"
            )
          );
          break;
        case "/favicon.ico":
          // special handler for the favicon logic
          res.statusCode = 204;
          return res.end();
        default:
          // handler for all framework packages
          let fsLoad = path.join(__dirname, "..", "..", "packages", url);
          if (url.startsWith("/__tests/")) {
            fsLoad = path.join(url.split("/__tests/")[1]);
          }

          response = fs.readFileSync(fsLoad);
      }

      res.write(response);
      res.end();
    });
  }
  /**
   * Start listening for HTTP requests.
   *
   * @returns {Object} Server address object.
   */
  async listen() {
    await this.server.listen(port);
    console.log(this.server.address());
    return { port: this.server.address(), server: this.server };
  }
  /**
   * Close the server.
   */
  async close() {
    await this.server.close();
  }
}

export default CliServer;
