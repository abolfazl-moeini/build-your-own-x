import { html, ComponiumComponent } from "componium";

export class ComponiumSample extends ComponiumComponent {
  static properties = {};

  constructor() {
    super();
  }

  render() {
    return html`
      <div>
        <h1>Componium Sample</h1>
      </div>
    `;
  }
}

customElements.define("componium-sample", ComponiumSample);
