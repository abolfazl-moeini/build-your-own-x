/**
 * Event emitter class.
 */
class EventEmitter {
  /**
   * Create new EventEmitter instance.
   */
  constructor() {
    /**
     * Map of event names to arrays of callback functions.
     * @private
     */
    this.callbacks = {};
  }

  /**
   * Subscribe to an event.
   *
   * @param {string} event - Name of the event.
   * @param {Function} cb - Callback function.
   */
  on(event, cb) {
    if (!this.callbacks[event]) this.callbacks[event] = [];
    this.callbacks[event].push(cb);
  }

  /**
   * Emit an event. Calls all subscribed callbacks.
   *
   * @param {string} event - Name of the event.
   * @param {...*} [args] - Arguments to pass to callbacks.
   */
  emit(event, ...stuff) {
    let cbs = this.callbacks[event];
    if (cbs) {
      cbs.forEach((cb) => cb(...stuff));
    }
  }
}

export default EventEmitter;
