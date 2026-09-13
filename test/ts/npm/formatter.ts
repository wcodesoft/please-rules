import { clsx } from "clsx";

export function formatClass(active: boolean, disabled: boolean): string {
  return clsx("btn", { "btn-active": active, "btn-disabled": disabled });
}
