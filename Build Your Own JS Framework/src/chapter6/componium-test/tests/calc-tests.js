// import framework modules
import ct, { assert, fake, replace } from "componium-test";
// fixture to test
import Calculator from "../fixtures/calculator.js";

let calc;

ct({
  describe: "Calculator Tests",
  before: () => {
    console.log("Starting my tests");
  },
  beforeEach: () => {
    calc = new Calculator();
  },
  multiply: function () {
    assert.equal(calc.multiply(3, 2), 6, "3 * 2 is 6");
  },
  mockMultiply: function () {
    const myFake = fake.returns(42);
    replace(calc, "multiply", myFake);
    assert.strictEqual(calc.multiply(1, 1), 42, "fake interface is working");
  },
  afterEach: () => {
    console.log("called afterEach");
  },
  after: () => {
    console.log("called after");
  },
});
