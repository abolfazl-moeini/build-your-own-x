const typeDefs = `#graphql
  type Query {
    date: String
  }
`;

const resolvers = {
  Query: {
    date: () => new Date().toString(),
  },
};
export { typeDefs, resolvers };
