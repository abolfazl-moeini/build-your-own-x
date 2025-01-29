class ReactiveComponent extends HTMLElement {
  constructor() {
    super();
    this.state = this.reactive(this.data());
    this.attachShadow({ mode: "open" });
  }

  reactive(data) {
    return new Proxy(data, {
      set: (target, key, value) => {
        target[key] = value;
        this.update();
        return true;
      },
    });
  }

  connectedCallback() {
    this.render();
    this.update();
  }

  render() {
    this.shadowRoot.innerHTML = this.template();
  }

  // override methods below
  update() {}

  data() {
    return {};
  }

  template() {
    return "";
  }
}

class MonthComponent extends ReactiveComponent {
  data() {
    return {
      monthCount: 5,
    };
  }

  template() {
    return `
      <button id="addMonths">Add Months</button>
      <div id="monthCount">Months: 0</div>
    `;
  }

  connectedCallback() {
    super.connectedCallback();
    this.shadowRoot
      .querySelector("#addMonths")
      .addEventListener("click", () => {
        this.state.monthCount++;
        if (this.state.monthCount > 12) {
          this.state.monthCount = 0;
        }
      });
  }

  update() {
    this.shadowRoot.querySelector(
      "#monthCount"
    ).textContent = `Months: ${this.state.monthCount}`;
  }
}

customElements.define("month-component", MonthComponent);

class YearComponent extends ReactiveComponent {
  data() {
    return {
      yearCount: 2024,
    };
  }

  template() {
    return `
      <button id="addYears">Add Years</button>
      <div id="yearCount">Year: 0</div>
      <month-component></month-component>
    `;
  }

  connectedCallback() {
    super.connectedCallback();
    this.shadowRoot.querySelector("#addYears").addEventListener("click", () => {
      this.state.yearCount++;
    });
  }

  update() {
    this.shadowRoot.querySelector(
      "#yearCount"
    ).textContent = `Year: ${this.state.yearCount}`;
  }
}

customElements.define("year-component", YearComponent);
