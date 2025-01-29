import { Sequelize } from "sequelize";
import { dirname } from "path";
import { fileURLToPath } from "url";
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
import config from "config";

/**
 * Represents the application's database.
 * Utilizes Sequelize for ORM functionalities.
 */
class Database {
  /**
   * Constructs the database instance.
   * Uses a configuration-defined connection string or defaults to an in-memory SQLite database.
   */
  constructor() {
    let dbConfig = "sqlite::memory:";
    try {
      dbConfig = config.get("database.connection_uri");
    } catch (e) {
      console.log("Using default database configuration...");
    }
    this.sequelize = new Sequelize(dbConfig);
  }

  /**
   * Authenticates the connection to the database.
   * @throws {Error} Throws an error if the authentication fails.
   */
  async authenticate() {
    await this.sequelize.authenticate();
  }
}

export default Database;
