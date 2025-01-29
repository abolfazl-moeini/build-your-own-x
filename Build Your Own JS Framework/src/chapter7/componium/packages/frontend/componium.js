// Importing required modules from the 'lit' library and local files.

// Importing `LitElement` and renaming it as `ComponiumComponent`
// alongside the `html` and `css` tag functions from the "lit" library.
import { LitElement as ComponiumComponent, html, css } from "lit";

// `ComponiumRouter` is a custom router module responsible for managing application routes.
import ComponiumRouter from "./router.js";

/**
 * ComponiumInternal class represents the core of the Componium application.
 * Currently, it serves as a placeholder with no internal logic.
 */
class ComponiumInternal {
  /**
   * Constructor for the ComponiumInternal class.
   */
  constructor() {}
}

/**
 * Componium function provides an instance of the ComponiumInternal class.
 *
 * @return {ComponiumInternal} An instance of the ComponiumInternal class.
 */
function Componium() {
  return new ComponiumInternal();
}

// Exporting the main `Componium` function as the default export.
export default Componium;

// Additional named exports for external use.
export { html, css, ComponiumComponent, ComponiumRouter };
