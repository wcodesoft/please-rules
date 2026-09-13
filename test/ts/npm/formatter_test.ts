import { formatClass } from "@test/formatter";

Deno.test("formatClass formats css classes with clsx", () => {
  const result = formatClass(true, false);
  if (result !== "btn btn-active") {
    throw new Error(`expected 'btn btn-active', got '${result}'`);
  }
});
