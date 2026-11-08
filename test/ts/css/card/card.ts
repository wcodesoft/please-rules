import { renderButton } from "@test/css/button";

export function renderCard(title: string): string {
  return `<div class="card"><h2>${title}</h2>${renderButton("OK")}</div>`;
}
