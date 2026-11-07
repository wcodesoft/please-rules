import { hello } from "@transitive/components/index.ts";

declare const Deno: { test: (name: string, fn: () => void) => void };

Deno.test("a package's own imports resolve when they are transitive dependencies", () => {
  const got = hello("world");
  if (got !== "hello world at /home") {
    throw new Error("unexpected result: " + got);
  }
});
