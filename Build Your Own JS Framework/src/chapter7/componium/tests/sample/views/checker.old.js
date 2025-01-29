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
  callback() {
    this.render();
    this.update();
  }
  render() {
    this.shadowRoot.innerHTML = this.template();
  }
  // methods that child components will override
  update() {}
  data() {
    return {};
  }
  template() {
    return "";
  }
}

class ChildComponent extends ReactiveComponent {
  data() {
    return {
      childCount: 0,
    };
  }

  template() {
    return `
      <button id="incrementChild">Increment Child</button>
      <div id="childCount">Child Count: 0</div>
    `;
  }

  callback() {
    super.callback();
    this.shadowRoot
      .querySelector("#incrementChild")
      .addEventListener("click", () => {
        this.state.childCount++;
      });
  }

  update() {
    this.shadowRoot.querySelector(
      "#childCount"
    ).textContent = `Child Count: ${this.state.childCount}`;
  }
}

customElements.define("child-component", ChildComponent);

class ParentComponent extends ReactiveComponent {
  data() {
    return {
      parentCount: 0,
    };
  }

  template() {
    return `
      <button id="incrementParent">Increment Parent</button>
      <div id="parentCount">Parent Count: 0</div>
      <child-component></child-component>
    `;
  }

  callback() {
    super.callback();
    this.shadowRoot
      .querySelector("#incrementParent")
      .addEventListener("click", () => {
        this.state.parentCount++;
      });
  }

  update() {
    this.shadowRoot.querySelector(
      "#parentCount"
    ).textContent = `Parent Count: ${this.state.parentCount}`;
  }
}

customElements.define("parent-component", ParentComponent);
