import { BRAND_HEX } from "@test/css/tokens";

export function renderButton(label: string): string {
  return `<button class="btn" data-brand="${BRAND_HEX}">${label}</button>`;
}
