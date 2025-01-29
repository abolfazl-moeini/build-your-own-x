import Componium from "componium";

/**
 * Main application bootstrapping and setup.
 * This script demonstrates how to initialize a Componium application, set up the server,
 * add middleware, routes, routers, and client views.
 */

// Initialize Componium application
const app = await Componium.initialize();

// Create server instance with specified options
const server = await app.createServer({
  name: "package-service",
  routesRootDir: "routes",
  port: 9000,
  gql: true, // Indicates GraphQL is enabled
});

/**
 * Add middleware to the server.
 *
 * @name middleware1
 * @callback
 */
server.addMiddleware("middleware1", () => {
  console.log("Server middleware log");
});

/**
 * Add dynamic route to the server.
 *
 * @path /dynamic
 * @callback
 */
server.addRoute("/dynamic", function (request, response) {
  return response.send("Dynamic Path");
});

// Create a router with a specified base path
const router = server.createRouter("/router");

/**
 * Add middleware to the router.
 *
 * @name routerMiddleware
 * @callback
 */
router.addMiddleware("routerMiddleware", (request) => {
  console.log("Router middleware:", request.originalUrl);
});

/**
 * Add a sample route to the router.
 *
 * @path /sample
 * @callback
 */
router.addRoute("/sample", function (request, response) {
  return response.send("Router Sample");
});

// Add client views to the server
server.addClientView("/", "frameworks", {
  title: "Frameworks",
});
server.addClientView("/client-render", "client-render", {
  title: "Client Render",
});
