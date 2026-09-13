import { add, multiply } from "@domain/calculator";

Deno.test("add returns sum of two numbers", () => {
  const result = add(2, 3);
  if (result !== 5) {
    throw new Error(`expected 2 + 3 = 5, got ${result}`);
  }
});

Deno.test("multiply returns product of two numbers", () => {
  const result = multiply(4, 5);
  if (result !== 20) {
    throw new Error(`expected 4 * 5 = 20, got ${result}`);
  }
});
