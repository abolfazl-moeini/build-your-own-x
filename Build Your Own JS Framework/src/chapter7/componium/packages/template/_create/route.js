/**
 * Default exported function for handling HTTP requests.
 *
 * This function is a basic route handler typically used with frameworks
 * like Express.js or similar server-side frameworks. When a request
 * hits this route, the function sends back a response with a message
 * and the current timestamp.
 *
 * @param {Object} req - The request object containing information about the HTTP request.
 * @param {Object} res - The response object used to send back the desired HTTP response.
 */
export default (req, res) => {
  res.send(`Welcome to the route. Timestamp: ${Date.now()}`);
};
