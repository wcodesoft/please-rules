import { describe, expect, it } from "vitest";
import { highlight } from "@test/highlight";

describe("an npm package under Vitest", () => {
  it("is imported by a first-party library through subpaths of a CommonJS package", () => {
    expect(highlight("package main")).toContain("hljs-keyword");
  });
});
