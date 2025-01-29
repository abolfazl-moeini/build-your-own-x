import { html, ComponiumComponent } from "componium";

export class ClientRender extends ComponiumComponent {
  static properties = {};

  constructor() {
    super();
  }

  render() {
    return html`
      <div>
        <h1>Client Component!</h1>
      </div>
    `;
  }
}

customElements.define("client-render", ClientRender);
