import Package from "../../models/package.js";

export default async (req, res) => {
  // Create a new instance of the `ExampleModel` with the given data.
  const model = await Package();
  const sample = await model.create({
    title: "Paper Delivery",
    address: "123 Main St.",
    created: new Date(2023, 1, 1),
  });

  console.log(sample.toJSON());

  const packages = await model.findAll();
  componium.logger.info(`Found packages ${JSON.stringify(packages)}`);
  res.json(packages);
};
