import createDebug from "debug";

/** Formats a duration in milliseconds with the `ms` package, which `debug` re-exports. */
export function humanize(milliseconds: number): string {
  return (createDebug as any).humanize(milliseconds);
}
