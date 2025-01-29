import * as Eta from "eta";
import path, { dirname } from "node:path";
import { fileURLToPath } from "node:url";

// Get file path references
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Configure Eta template renderer
Eta.configure({
  views: path.join(__dirname, "views"),
});

let template = "<%~ includeFile('./index.eta', it) %>";

/**
 * Render layout using Eta templates.
 *
 * @param {Object[]} testFiles - List of test files to include.
 * @returns {string} Rendered HTML layout.
 */
function render(testFiles) {
  const layout = Eta.render(template, {
    message: "Open browser console for test results...",
    testFiles: testFiles,
  });
  return layout;
}

export default render;
