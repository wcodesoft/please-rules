/// <reference lib="dom" />
import { renderCard } from "@test/css/card";

declare const expect: (actual: unknown) => {
  toBe: (expected: unknown) => void;
};

Deno.test("co-located ts_library CSS is injected into browser test DOM", () => {
  document.body.innerHTML = `<main class="dashboard">${
    renderCard("Metrics")
  }</main>`;
  const card = document.querySelector(".card") as HTMLElement;
  const btn = document.querySelector(".btn") as HTMLElement;
  expect(getComputedStyle(card).paddingTop).toBe("8px");
  expect(getComputedStyle(card).backgroundColor).toBe("rgb(0, 0, 0)");
  expect(getComputedStyle(btn).marginTop).toBe("4px");
  expect(getComputedStyle(btn).marginLeft).toBe("8px");
});
