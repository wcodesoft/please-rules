import { expect, it } from "vitest";
import { both, classify } from "@domain/branches";

it("classify and both", () => {
  expect(classify(-1)).toBe("neg");
  expect(classify(5)).toBe("pos");
  expect(both(false, true)).toBe(false);
});
