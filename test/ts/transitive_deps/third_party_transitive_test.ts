import { classes } from "@transitive/styled/styled.ts";

declare const Deno: { test: (name: string, fn: () => void) => void };

Deno.test("a library's third-party import resolves when the module is a transitive dependency", () => {
  if (classes(true) !== "item active") {
    throw new Error("unexpected result: " + classes(true));
  }
});
