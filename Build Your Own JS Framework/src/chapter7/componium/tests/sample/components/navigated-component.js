import { css, html, ComponiumComponent } from "componium";

export class NavigatedComponent extends ComponiumComponent {
  static styles = css`
    div {
      border: 1px solid black;
      padding: 1rem;
      display: flex;
      flex-direction: column;
      align-items: center;
    }
  `;

  constructor() {
    super();
  }

  render() {
    return html`
      <div>
        <h2>Navigated Component</h2>
      </div>
    `;
  }
}

customElements.define("navigated-component", NavigatedComponent);
