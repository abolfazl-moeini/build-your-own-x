import ct, { assert } from "componium-test";
import { helloReader } from "../basic.js";

ct({
  basic: function () {
    assert.strictEqual(helloReader(), "Hello reader!", "output is correct");
  },
});
