/**
 * Custom router class that wraps Express router.
 */

import express from "express";

class Router {
  /**
   * Constructor - initializes Express router.
   */
  constructor() {
    this.router = express.Router();
  }

  /**
   * Registers a route handler.
   *
   * @param {string} path - Route path
   * @param {function} handler - Route handler function
   * @param {string} type - HTTP method (default: 'get')
   */
  addRoute(path, handler, type = "get") {
    this.router[type](path, handler);
  }

  /**
   * Registers middleware.
   *
   * @param {string} name - Middleware name
   * @param {function} middleware - Middleware function
   */
  addMiddleware(name, middleware) {
    this.router.use(function (req, res, next) {
      middleware(req, res);
      next();
    });
  }
}

export default Router;
