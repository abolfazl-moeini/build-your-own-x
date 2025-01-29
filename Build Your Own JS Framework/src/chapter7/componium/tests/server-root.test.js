import test from "ava";
import { spawn } from "child_process";
import axios from "axios";
import path from "path";
import url from "url";

const __dirname = path.dirname(url.fileURLToPath(import.meta.url));

test('Server starts and serves HTML containing "List of frameworks"', async (t) => {
  let server;

  const componiumPath = path.resolve(__dirname, "../packages/bin/componium.js");

  await new Promise((resolve, reject) => {
    // Spawn the process running "componium dev" from the tests/sample directory
    server = spawn("node", [componiumPath, "dev"], {
      cwd: path.resolve(__dirname, "sample"),
    });

    // Attach error handler
    server.on("error", reject);

    // Attach stdout handler to capture server ready message
    server.stdout.on("data", async (data) => {
      const str = data.toString();

      // Logging to see the server output (optional)
      console.log(`Server Output: ${str}`);

      // You might need to adapt the following line depending on the actual log message
      if (str.includes("Componium Server")) {
        try {
          // Wait for a few more seconds to ensure the server is fully up
          // You can adjust this delay as per your requirements
          await new Promise((resolve) => setTimeout(resolve, 2000));

          // Fetch data from localhost:9000 and check if it contains "List of frameworks"
          const response = await axios.get("http://localhost:9000");
          t.true(response.data.includes("List of frameworks"));

          // Terminate the server process
          server.kill("SIGTERM");

          // Resolve the promise to indicate that the test has passed
          resolve();
        } catch (error) {
          reject(error);
        }
      }
    });
  });
});
