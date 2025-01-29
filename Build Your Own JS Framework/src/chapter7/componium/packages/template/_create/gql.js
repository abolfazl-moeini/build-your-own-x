/**
 * `typeDefs` define the structure of the GraphQL schema. The schema
 * specifies the types of queries that can be made against the API,
 * and what kind of data can be retrieved.
 *
 * In this case, we have a single `Query` type, which allows one to
 * retrieve a property named `prop` of type `String`.
 *
 * Syntax:
 * `#graphql` at the beginning is a tagged template literal
 * indicating that the string represents a GraphQL schema.
 */
const typeDefs = `#graphql
  type Query {
    prop: String
  }
`;

/**
 * `resolvers` are responsible for defining how the data for the
 * queries specified in `typeDefs` is actually fetched or computed.
 *
 * Here, for the `prop` query, we simply return a static string
 * "Property". In a real-world application, this function might
 * retrieve data from a database, compute some value, or call
 * another API.
 */
const resolvers = {
  Query: {
    prop: () => "Property",
  },
};
// Export both `typeDefs` and `resolvers` so they can be used elsewhere in the application.
export { typeDefs, resolvers };
