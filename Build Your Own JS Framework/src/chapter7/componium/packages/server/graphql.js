/**
 * Creates and configures an Apollo GraphQL server.
 * Merges schemas and attaches to the provided Express app.
 */

import { ApolloServer } from "@apollo/server";
import { mergeSchemas } from "@graphql-tools/schema";
import { ApolloServerPluginDrainHttpServer } from "@apollo/server/plugin/drainHttpServer";
import bodyParser from "body-parser";
import { expressMiddleware } from "@apollo/server/express4";

/**
 * Creates an Apollo GraphQL server and attaches it to Express.
 *
 * @param {object} app - Express app instance
 * @param {object} httpServer - Node HTTP server instance
 * @param {array} schemas - List of GraphQL schemas to merge
 * @returns {Promise<void>}
 */
async function createGraphqlServer(app, httpServer, schemas) {

  // Create Apollo Server instance
  const apolloServer = new ApolloServer({
    introspection: true,
    schema: mergeSchemas({
      schemas
    }),
    plugins: [
      // Drain HTTP server on stop
      ApolloServerPluginDrainHttpServer({ httpServer })
    ],
  });

  // Start Apollo Server
  await apolloServer.start();

  // Attach Apollo middleware to Express app
  app.use(
      "/graphql",
      bodyParser.json(),
      expressMiddleware(apolloServer)
  );

}

export { createGraphqlServer };