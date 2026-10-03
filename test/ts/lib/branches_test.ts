import { both, classify } from "@domain/branches";

Deno.test("classify and both", () => {
  if (classify(-1) !== "neg" || classify(5) !== "pos" || both(false, true)) {
    throw new Error("unexpected result");
  }
});
