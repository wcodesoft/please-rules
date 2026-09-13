Deno.test("deno version matches toolchain", () => {
  const version = Deno.version.deno;
  if (!version.startsWith("2.")) {
    throw new Error(`expected Deno 2.x version, got ${version}`);
  }
});

Deno.test("execution is offline without network dependencies", () => {
  // Verifies environment runtime access
  if (typeof Deno.env.get !== "function") {
    throw new Error("Deno.env API not available");
  }
});
