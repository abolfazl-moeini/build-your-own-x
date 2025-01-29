import { css, html, ComponiumComponent } from "componium";

export class NestedComponent extends ComponiumComponent {
  static styles = css`
    div {
      border: 1px solid black;
      padding: 1rem;
      display: flex;
      flex-direction: column;
      align-items: center;
    }
  `;

  static properties = {
    name: { type: String },
  };

  constructor() {
    super();
    this.name = "Nested Component Page";
  }

  render() {
    const navigate = (url) => (eve) => {
      eve.preventDefault();
      const nav_url = url || eve.target.href;
      history.pushState(null, "", nav_url);
      window.dispatchEvent(new Event("pushstate"));
    };

    return html`
      <div>
        <h2>Nested Component</h2>
        <p>${this.name}</p>
        <button @click=${navigate("/page")}>Navigate >></button>
      </div>
    `;
  }
}

customElements.define("nested-component", NestedComponent);
