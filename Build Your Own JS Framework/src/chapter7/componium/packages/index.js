// Importing required modules
import process from "node:process"; // Built-in node module to access process-level information.

// Importing application-specific modules
import Logger from "./app/logger.js"; // Custom logger for the application.
import { Server } from "./server/server.js"; // Server class for creating and managing the server instance.
import { Optimize } from "./frontend/optimize.js"; // Class to optimize front-end components.
import Database from "./db/index.js"; // Database class to handle database operations.

/**
 * Main ComponiumApp class to set up and manage the application.
 */
class ComponiumApp {
  /**
   * Constructor to set up the initial state of the application.
   * It creates a global `componium` object and initializes the models.
   * Additionally, it optimizes front-end component files if the node environment is set to production.
   */
  constructor() {
    global.componium = {}; // Creating a global object to hold application-level data.

    // Optimize front-end component files if the environment is production.
    if (process.env.NODE_ENV === "production") {
      const optimize = new Optimize();
      optimize.minifyScripts();
    }
  }

  /**
   * Initializes core components of the application such as the logger and database.
   *
   * @return {ComponiumApp} Returns the current instance of the ComponiumApp for chaining.
   */
  async initialize() {
    componium.logger = new Logger(); // Initializing logger
    this.db = new Database(); // Setting up the database.
    await this.db.authenticate(); // Authenticating with the database.
    componium.db = this.db; // Assigning the database instance to the global object.

    return this;
  }

  /**
   * Creates a new server instance with the given options or default settings.
   *
   * @param {object} serverOpts - Configuration options for the server.
   *                              Includes port, routesRootDir, and gql flag.
   * @return {Server} Returns the server instance.
   */
  async createServer(serverOpts = {}) {
    const defaults = { port: 8081, routesRootDir: process.cwd(), gql: false }; // Default server settings.
    const options = { ...defaults, ...serverOpts }; // Merging user options with default settings.

    return await new Server(options).create(); // Creating and returning a server instance.
  }
}

// Create an instance of the ComponiumApp.
let Componium = new ComponiumApp();

// Exporting the main ComponiumApp instance for external use.
export default Componium;
