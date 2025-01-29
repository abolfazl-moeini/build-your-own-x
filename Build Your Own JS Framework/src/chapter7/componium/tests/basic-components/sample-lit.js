import { html, css, LitElement } from "lit";
import { customElement, property } from "lit/decorators.js";

@customElement("year-component")
export class YearComponent extends LitElement {
  static styles = css`
    #yearCount {
      color: green;
    }
  `;

  @property()
  year = 2024;

  render() {
    return html`<button id="addYears">Add Years</button>
      <div id="yearCount">Year: ${this.year}</div>`;
  }
}

export class YearComponent extends LitElement {
  static styles = css`
    p {
      color: green;
    }
  `;

  static properties = {
    year: { type: Number },
  };

  constructor() {
    super();
    this.year = 2024;
  }

  render() {
    return html` <button id="addYears">Add Years</button>
      <div id="yearCount">Year: ${this.year}</div>`;
  }
}
