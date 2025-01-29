export default function frameworks(request, response) {
  const hitsComponium = 1000;
  const hitsAngular = 0;
  const hitsVue = 0;

  return `
    <style>
    .framework-list {
      display: flex;
      justify-content: center;
    } 
    </style>
    <h1 style="font-family: sans-serif; text-align: center;">List of frameworks</h1>
    <div class="framework-list">
      <framework-item
        name="Componium"
        count="${hitsComponium}"
      ></framework-item>
      <framework-item
        name="Angular"
        count="${hitsAngular}"
      ></framework-item>
      <framework-item name="vue.js" count="${hitsVue}"></framework-item>
    </div>
  `;
}
