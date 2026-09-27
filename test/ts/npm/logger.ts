// @ts-ignore
import * as debugModule from "debug";

const debug = typeof debugModule === "function" ? debugModule : (debugModule as any).default || debugModule;
const log = debug("test:logger");

export function formatDuration(msVal: number): string {
  log("formatting duration %d", msVal);
  return `duration: ${msVal}ms`;
}
