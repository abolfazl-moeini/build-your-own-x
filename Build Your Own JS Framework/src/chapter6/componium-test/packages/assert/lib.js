import { isBrowser } from "../util/env.js";

let chai;

// Initialize Chai based on environment
if (isBrowser()) {
  chai = window.chai;
} else {
  chai = await import("chai");
}

// Chai interfaces
let expect = chai.expect;
let assert = chai.assert;
let should = chai.should;

// Export the assert lib
export { expect, assert, should };
