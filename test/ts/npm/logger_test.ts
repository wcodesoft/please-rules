import { formatDuration } from "@test/logger";
// @ts-ignore
import * as msModule from "ms";

const ms = typeof msModule === "function" ? msModule : (msModule as any).default || msModule;

Deno.test("logger uses transitive ms and explicit ms", () => {
  const formatted = formatDuration(1000);
  if (formatted !== "duration: 1000ms") {
    throw new Error(`unexpected formatted: ${formatted}`);
  }

  const parsed = ms("1 minute");
  if (parsed !== 60000) {
    throw new Error(`expected ms('1 minute') to be 60000, got ${parsed}`);
  }
});
