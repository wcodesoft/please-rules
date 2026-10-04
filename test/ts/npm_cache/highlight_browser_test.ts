import { highlight } from "@test/highlight";

declare const Deno: {
  test: (name: string, fn: () => void | Promise<void>) => void;
};
declare const expect: (actual: any) => {
  toBeTruthy: () => void;
};

Deno.test("highlight.js (an npm package) runs inside the browser", () => {
  expect(highlight("package main").includes("hljs-keyword")).toBeTruthy();
});
