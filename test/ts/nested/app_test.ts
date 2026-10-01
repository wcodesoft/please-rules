import { describe, expect, it } from "vitest";
import { page } from "@scope/app";

describe("nested module aliases with Vitest runner", () => {
  it("resolves a nested module_name imported by a parent module_name", () => {
    expect(page("x")).toBe("<page><widget>x</widget></page>");
  });
});
