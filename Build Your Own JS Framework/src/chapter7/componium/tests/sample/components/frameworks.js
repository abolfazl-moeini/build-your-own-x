import { css, html, ComponiumComponent } from "componium";
export class Framework extends ComponiumComponent {
  static styles = css`
    div {
      border: 2px solid grey;
      border-radius: 5px;
      margin: 10px;
      padding: 10px;
      min-width: 300px;
    }
    button {
      background: #fff;
      border: 1px solid black;
      border-radius: 5px;
      padding: 5px;
    }
    aside {
      font-size: 20px;
      padding: 5px;
    }
  `;

  static properties = {
    name: {},
    count: { type: Number },
  };

  constructor() {
    super();
    this.count = 0;
  }

  render() {
    return html`
      <div>
        <h1>${this.name}</h1>
        <aside>⭐: ${this.count}</aside>
        <button @click=${() => this.count++}>Add stars</button>
      </div>
    `;
  }
}

customElements.define("framework-item", Framework);
