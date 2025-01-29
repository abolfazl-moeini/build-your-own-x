# ct // Componium Test - JavaScript testing framework

> ct (componium-test) is a simple JavaScript testing full-stack framework
 
## Features

* Runs tests in node.js and in the browser.
* Powered by Chai.js assertions.
* Mocking powered by the Sinon.js library.

### Public API

Here's an example of available APIs for this framework:

```js
// include the framework
import ct, { assert } from "componium-test";

// create test suites
ct({
  // set the title
  describe: "Unit test suite two",
    
  // before functions  
  before: () => {
    console.log("called before");
  },
  beforeEach: () => {
    console.log("called beforeEach");
  },
  "your test": async function () {
    // add logic relevant to your components here  
    assert.equal(2, 2, "2 is 2");
  },
  
  // after methods  
  afterEach: () => {
    console.log("called afterEach");
  },
  after: () => {
    console.log("called after");
  },
});

```

## Release History

See the [CHANGELOG](CHANGELOG.md).

## License

[MIT](LICENSE)