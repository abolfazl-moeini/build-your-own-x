/**
 * ComponiumRouter is a client-side routing mechanism that listens to browser
 * history changes, updates the URL, and displays the matching route template.
 */
class ComponiumRouter {
  /**
   * Constructor for the ComponiumRouter class.
   *
   * @param {HTMLElement} element - The DOM element to bind the router.
   * @param {Array} routes - An array of route objects with path and template properties.
   * @param {HTMLElement} root - The DOM element where matched templates will be rendered.
   */
  constructor(element, routes, root) {
    const router = this;
    this.element = element;
    this.routes = routes;
    this.root = root;
    this._loadInitialRoute(); // Load the initial route based on the current URL.

    // This function handles browser navigation events.
    function onNavigate() {
      // Split the current URL to get its segments.
      const pathNameSplit = window.location.pathname.split("/");
      const pathSegs = pathNameSplit.length > 1 ? pathNameSplit.slice(1) : "";

      // Load the matched route for the current URL segments.
      router.loadRoute(...pathSegs);
    }

    // Listen to browser navigation events.
    window.addEventListener("pushstate", onNavigate);
    window.addEventListener("popstate", onNavigate);
  }

  /**
   * Updates the browser URL and displays the matched route template.
   *
   * @param {...string} urlSegs - Segments of the URL.
   */
  loadRoute(...urlSegs) {
    // Find the route that matches the given URL segments.
    const matchedRoute = this._matchUrlToRoute(urlSegs);

    // Construct the full URL from the segments.
    const url = `/${urlSegs.join("/")}`;
    history.pushState({}, "", url);

    // Update the root element's content with the matched route's template.
    const routerOutElem = this.root;
    routerOutElem.innerHTML = matchedRoute.template;
  }

  /**
   * Match a given URL to one of the defined routes.
   *
   * @param {Array} urlSegs - Segments of the URL.
   * @returns {Object} The matched route or undefined if no match found.
   */
  _matchUrlToRoute(urlSegs) {
    return this.routes.find((route) => {
      const routePathSegs = route.path.split("/").slice(1);

      if (routePathSegs.length !== urlSegs.length) {
        return false;
      }

      // Check if every segment of the route's path matches the given URL segments.
      return routePathSegs.every(
        (routePathSeg, i) => routePathSeg === urlSegs[i]
      );
    });
  }

  /**
   * Load the route based on the current URL when the router is first initialized.
   */
  _loadInitialRoute() {
    const pathNameSplit = window.location.pathname.split("/");
    const pathSegs = pathNameSplit.length > 1 ? pathNameSplit.slice(1) : "";
    this.loadRoute(...pathSegs);
  }
}

// Export the ComponiumRouter class for external use.
export default ComponiumRouter;
