import { PREFIX } from "@transitive/base/prefix.ts";

export function greeting(name: string): string {
  return PREFIX + " " + name;
}
