/**
 * Server class - wraps and configures back-end servers.
 */

import express from "express";
import http from "http";
import makeDebug from "debug";
import { fileRoutes } from "./file-router.js";
import path, { dirname } from "path";
import { createGraphqlServer } from "./graphql.js";
import Router from "./router.js";
import { cwd } from "node:process";
const debug = makeDebug("componium:server");

import { ClientView } from "../frontend/client-view.js";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Valid HTTP methods
const VALID_METHODS = [
  "all",
  "get",
  "post",
  "put",
  "delete",
  "patch",
  "options",
  "head",
];

class Server {
  /**
   * Constructor
   * Initialize express and other directory options
   */
  constructor(options) {
    this.app = express();
    this.port = options.port;
    this.routesRootDir = options.routesRootDir;
    this.gql = options.gql;
  }
  /**
   * Bootstraps and configures the server.
   */
  async create() {
    // Express app and HTTP server
    const app = this.app;
    const httpServer = http.createServer(app);
    // Start listening
    app.listen(this.port);
    console.log(
      `Componium Server (PID: ${process.pid}) started on port: ${this.port}`
    );
    // Load routes
    const result = await fileRoutes(
      path.join(cwd(), this.routesRootDir),
      app,
      this.gql
    );

    // Add GraphQL server
    if (this.gql) {
      await createGraphqlServer(app, httpServer, result.schemas);
    }
    // Additional framework routes...
    const frameworkPaths = {
      "/_framework/lit-element-hydrate-support.js": [
        "@lit-labs",
        "ssr-client",
        "lit-element-hydrate-support.js",
      ],
      "/_framework/lit.js": ["lit", "index.js"],
      "/_framework/lit-html.js": ["lit-html", "lit-html.js"],
      "/_framework/lit-reactive-element.js": [
        "@lit",
        "reactive-element",
        "reactive-element.js",
      ],
      "/_framework/lit-element.js": ["lit-element", "lit-element.js"],
      "/_framework/lit-html/is-server.js": ["lit-html", "is-server.js"],
      "/_framework/css-tag.js": ["@lit", "reactive-element", "css-tag.js"],
      "/_framework/lib/hydrate-lit-html.js": [
        "@lit-labs",
        "ssr-client",
        "lib",
        "hydrate-lit-html.js",
      ],
      "/_framework/private-ssr-support.js": [
        "lit-html",
        "private-ssr-support.js",
      ],
      "/_framework/directive.js": ["lit-html", "directive.js"],
      "/_framework/directive-helpers.js": ["lit-html", "directive-helpers.js"],
      "/_framework/template-shadowroot.js": [
        "@webcomponents",
        "template-shadowroot",
        "template-shadowroot.js",
      ],
      "/_framework/_implementation/feature_detect.js": [
        "@webcomponents",
        "template-shadowroot",
        "_implementation",
        "feature_detect.js",
      ],
      "/_framework/_implementation/default_implementation.js": [
        "@webcomponents",
        "template-shadowroot",
        "_implementation",
        "default_implementation.js",
      ],
      "/_framework/_implementation/manual_walk.js": [
        "@webcomponents",
        "template-shadowroot",
        "_implementation",
        "manual_walk.js",
      ],
      "/_framework/_implementation/util.js": [
        "@webcomponents",
        "template-shadowroot",
        "_implementation",
        "util.js",
      ],
    };

    for (let [route, segments] of Object.entries(frameworkPaths)) {
      app.get(route, (request, response) => {
        response.sendFile(
          path.join(__dirname, "..", "..", "node_modules", ...segments)
        );
      });
    }

    const localFrameworkPaths = {
      "/_framework/componium.js": ["..", "frontend", "componium.js"],
      "/_framework/router.js": ["..", "frontend", "router.js"],
    };

    for (let [route, segments] of Object.entries(localFrameworkPaths)) {
      app.get(route, (request, response) => {
        response.sendFile(path.join(__dirname, ...segments));
      });
    }

    // Static assets
    if (process.env.NODE_ENV === "production") {
      app.use("/components", express.static(".componium/components"));
    } else {
      app.use("/components", express.static("components"));
    }

    app.use("/views", express.static("views"));
    app.use("/static", express.static("static"));

    return this;
  }
  /**
   * Registers a route handler.
   */
  addRoute(path, handler, type = "get") {
    type = type.toLowerCase();

    if (VALID_METHODS.includes(type) && path && handler) {
      this.app[type](path, handler);
    } else {
      throw new Error(
        `Invalid route for ${path} with type: '${type}' and handler: ${handler}`
      );
    }
  }
  /**
   * Registers a client-side view route.
   */
  addClientView(path, name, options) {
    const server = this;
    server.addRoute(path, async function (request, response) {
      const view = await server.createClientView(name, request, response, {
        title: options.title,
      });
      response.setHeader("Content-Type", "text/html");
      return response.send(view);
    });
  }
  /**
   * Registers middleware.
   */
  addMiddleware(name, middleware) {
    this.app.use(function (req, res, next) {
      middleware(req, res);
      next();
    });
  }
  /**
   * Renders a client-side view.
   */
  async createClientView(name, request, response, options) {
    return await new ClientView(name, options).create(request, response);
  }

  /**
   * Creates a nested router.
   */
  createRouter(route) {
    const newRouter = new Router();
    this.app.use(route, newRouter.router);
    return newRouter;
  }

  /**
   * Registers a static directory handler.
   *
   * @param {string} path - Route path
   * @param {function} directory - Route directory to serve files from
   */
  addStaticDirectory(path, directory) {
    this.app.use(path, express.static(directory));
  }
}

export { Server };
