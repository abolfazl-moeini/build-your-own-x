import ct, { assert } from "componium-test";

ct({
  one: function () {
    let a = 1;
    assert.strictEqual(a, 1, "a is 1");
  },
});
