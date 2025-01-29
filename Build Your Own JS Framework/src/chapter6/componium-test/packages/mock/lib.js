import { isBrowser } from "../util/env.js";

let sinon;

if (isBrowser()) {
  sinon = await import("sinon");
} else {
  sinon = await import("sinon");
}

const stub = sinon.stub;
const spy = sinon.spy;
const mock = sinon.mock;
const replace = sinon.replace;
const fake = sinon.fake;
const restore = sinon.fake;

export { stub, spy, mock, replace, fake, restore };
