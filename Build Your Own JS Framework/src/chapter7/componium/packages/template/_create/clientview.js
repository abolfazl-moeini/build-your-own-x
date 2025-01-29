/**
 * ClientView function that returns a simple HTML representation.
 *
 * This function provides a basic HTML string that can be used to display a heading
 * with some associated styles. The main purpose is to serve this HTML on the client-side
 * possibly as a part of a web page or web application.
 *
 * @param {Object} request - The request object containing information about the incoming HTTP request.
 * @param {Object} response - The response object used to send back the desired HTTP response.
 * @returns {string} - An HTML string consisting of style definitions and a heading.
 */
export default function clientview(request, response) {
  return `
    <style>
    .example-heading {
      display: flex;
      justify-content: center;
    } 
    </style>
    <h1 class="example-heading">Example Client View</h1>
  `;
}
