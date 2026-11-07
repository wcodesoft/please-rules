import clsx from "clsx";

export function classes(active: boolean): string {
  return clsx("item", { active });
}
