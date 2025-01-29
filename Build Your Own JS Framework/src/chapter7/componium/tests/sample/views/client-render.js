export default function clientRender(request, response) {
  return `
    <script src="/components/client-render-two.js" type="module"></script>
    <script src="/components/nested-component.js" type="module"></script>
    <script src="/components/navigated-component.js" type="module"></script>
    <h1>Client Component!</h1>
    <div id="client-render"><client-render-two></client-render-two></div>
  `;
}
