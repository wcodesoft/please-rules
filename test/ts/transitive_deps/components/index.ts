import { greeting } from "@transitive/lib/greeting.ts";
import { currentPath } from "@transitive/routes";

export function hello(name: string): string {
  return greeting(name) + " at " + currentPath();
}
