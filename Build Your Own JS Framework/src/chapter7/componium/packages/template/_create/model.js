// Import required modules and classes from the "sequelize" package.
// - `DataTypes`: Provides utility functions for defining the types of attributes in a model.
import { DataTypes } from "sequelize";

export default async function () {
  // Initialize the `ExampleModel` class with its attributes and configuration.
  const sequelize = componium.db.sequelize;

  const Example = sequelize.define(
    "Example",
    {
      name: DataTypes.STRING,
    },
    {}
  );

  await sequelize.sync();

  return Example;
}
