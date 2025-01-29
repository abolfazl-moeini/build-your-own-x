import { html, ComponiumComponent, ComponiumRouter } from "componium";

const routes = [
  {
    path: "/client-render",
    template: "<nested-component></nested-component>",
  },
  {
    path: "/page",
    template: "<navigated-component></navigated-component>",
  },
];

export class ClientRender extends ComponiumComponent {
  static properties = {};

  constructor() {
    super();
  }

  connectedCallback() {
    super.connectedCallback();
    this.router = new ComponiumRouter("client-render", routes, this.shadowRoot);
  }

  render() {
    return html``;
  }
}

customElements.define("client-render-two", ClientRender);
