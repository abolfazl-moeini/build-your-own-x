// Importing required libraries and modules.

// `makeDebug` is a function from the "debug" package for creating a debugger instance.
import makeDebug from "debug";

// Creating a debug instance specifically for "componium:frontend-client-view".
const debug = makeDebug("componium:frontend-client-view");

// `Renderer` is a custom module responsible for rendering components.
import { Renderer } from "./renderer.js";

// Importing `cwd` function from node's process module to get the current working directory.
import { cwd } from "node:process";

// `collectResult` is a function to aggregate rendering results from the "@lit-labs/ssr" library.
import { collectResult } from "@lit-labs/ssr/lib/render-result.js";

// The `path` module provides utilities for working with file and directory paths.
import path from "path";

/**
 * ClientView class responsible for managing the view components for the client-side.
 */
class ClientView {
  /**
   * Constructs a ClientView instance.
   *
   * @param {string} name - The name of the view component.
   * @param {object} options - Configuration options for the view.
   */
  constructor(name, options) {
    this.name = name; // Component name.
    this.options = options; // Component options.
  }

  /**
   * Asynchronously loads and renders the client-side component view.
   *
   * @param {object} request - The incoming HTTP request object.
   * @param {object} response - The outgoing HTTP response object.
   * @return {Promise<object>} Returns a promise that resolves to the collected rendering result.
   * @throws Will throw an error if the component is not found or not a function.
   */
  async create(request, response) {
    // Dynamically importing the component using the provided name from the views directory.
    const viewPath = path.join(cwd(), "/views/", `${this.name}.js`);
    debug("Creating view path:", viewPath);
    const component = await import(`file:///${viewPath}`);

    // Check if the imported component has a default export and it's a function.
    if (!component.default) {
      throw new Error(
        `Component ${this.name} is not a function or could not be found!`
      );
    }

    // Create a new Renderer instance for the component and render it.
    const result = await new Renderer(this.name).render(
      component.default,
      request,
      response,
      this.options
    );

    // Aggregate the rendering results and return.
    return await collectResult(result);
  }
}

// Exporting the ClientView class for external use.
export { ClientView };
