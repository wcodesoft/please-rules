import { describe, expect, it } from "vitest";
import { add, multiply } from "@domain/calculator";

describe("Calculator with Vitest runner", () => {
  it("adds two numbers correctly", () => {
    expect(add(2, 3)).toBe(5);
  });

  it("multiplies two numbers correctly", () => {
    expect(multiply(3, 4)).toBe(12);
  });
});
