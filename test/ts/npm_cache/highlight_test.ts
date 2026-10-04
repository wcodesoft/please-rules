import { highlight } from "@test/highlight";

Deno.test("highlight.js: subpath imports of a CommonJS package", () => {
  const html = highlight("package main");
  if (!html.includes("hljs-keyword")) {
    throw new Error(`expected a highlighted keyword, got ${html}`);
  }
});
