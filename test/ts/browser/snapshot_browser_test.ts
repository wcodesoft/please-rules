/// <reference lib="dom" />
declare const Deno: {
  test: (name: string, fn: () => void | Promise<void>) => void;
};
declare const expectComputedStyles: (
  name: string,
  selector: string,
  props: string[],
) => void;

Deno.test("computed styles of a styled card match the golden", () => {
  const style = document.createElement("style");
  style.textContent = `
    .card { display: flex; padding: 8px; color: rgb(200, 0, 0); }
    .card.narrow { color: rgb(0, 0, 200); }
    @media (max-width: 700px) { .card { color: rgb(0, 0, 200); } }
  `;
  document.head.appendChild(style);

  const container = document.createElement("div");
  container.innerHTML = `<div class="card">a</div><div class="card narrow">b</div>`;
  document.body.appendChild(container);

  expectComputedStyles("card", ".card", ["display", "padding-top", "color"]);

  document.body.removeChild(container);
  document.head.removeChild(style);
});
