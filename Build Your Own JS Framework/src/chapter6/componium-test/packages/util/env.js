const Enviroments = {
  Node: "node",
  Browser: "browser",
};
/**
 * Returns true if environment is a browser.
 *
 * @returns {boolean}
 */
const ComponiumStatus = {
  Fail: "COMPONIUM FAIL",
  Pass: "COMPONIUM PASS",
};
const isBrowser = () => {
  return typeof window !== "undefined";
};

function getExitStatus(result) {
  return result.status === ComponiumStatus.Fail ? 1 : 0;
}

export { isBrowser, Enviroments, ComponiumStatus, getExitStatus };
