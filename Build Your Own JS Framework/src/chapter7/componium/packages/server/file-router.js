/**
 * File routing middleware.
 * Automatically registers routes based on the file structure in /routes.
 * Supports GET, POST, and GraphQL schema files.
 */

import fs from "fs";
import path from "path";
import makeDebug from "debug";
import { makeExecutableSchema } from "@graphql-tools/schema";

const debug = makeDebug("componium:file-router");

/**
 * Registers route handlers based on files in the routes directory.
 * @param {string} dir - The routes directory
 * @param {object} app - Express app instance
 * @param {object} gql - Apollo Server instance (optional)
 * @returns {object} - GraphQL schemas
 */
const fileRoutes = async (dir, app, gql) => {
  let schemas = [];

  // Recursive function to load routes
  const fileRoutesInternal = async (dir, app, gql) => {
    let files = [];
    const root = dir;
    try {
      files = fs.readdirSync(root);
    } catch (ex) {
      debug("No file based routes found, skipping...", ex);
    }

    for (let file of files) {
      // Build file path
      const filePath = path.join(root, file);

      // Get file stats
      const stat = fs.statSync(filePath);

      if (stat.isDirectory()) {
        // Recursively process subdirectories
        await fileRoutesInternal(filePath, app, gql);
      } else if (stat.isFile() && path.extname(filePath) === ".js") {
        // Build route path from file path
        const pathArray = filePath.split(path.sep);
        const routePath =
          "/" + pathArray.slice(pathArray.indexOf("routes") + 1, -1).join("/");

        // Get route name and type from filename
        const fileBlocks = path.basename(filePath).split(".");
        const routeName = fileBlocks[0];
        const fileType = fileBlocks[1];

        // Import route handler
        const routeHandler = await import(`file:///${filePath}`);

        debug(`Registering ${fileType.toUpperCase()} ${routePath} `);

        // Build API path from route path and name
        let apiPath =
          routePath === "/" ? "/" + routeName : routePath + "/" + routeName;

        // Register route based on file type
        if (fileType === "js" && routeName === "_index") {
          app.get(routePath, routeHandler.default);
        } else if (fileType === "get" || fileType === "js") {
          app.get(apiPath, routeHandler.default);
        } else if (fileType === "post") {
          
        } else if (fileType === "gql" && gql) {
          const { typeDefs, resolvers } = await import(`file:///${filePath}`);
          const schema = makeExecutableSchema({ typeDefs, resolvers });
          schemas.push(schema);
        }
      }
    }
  };

  await fileRoutesInternal(dir, app, gql);

  return {
    schemas,
  };
};

export { fileRoutes };
