import { humanize } from "@test/humanize";

Deno.test("humanize formats durations through a CommonJS package and its dependency", () => {
  const got = humanize(90061000);
  if (got !== "1d") {
    throw new Error(`expected 1d, got ${got}`);
  }
});
