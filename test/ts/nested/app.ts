import { render } from "@scope/app/components/widget";

export function page(name: string): string {
  return `<page>${render(name)}</page>`;
}
