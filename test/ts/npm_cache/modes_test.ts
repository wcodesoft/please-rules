import { goLanguage } from "@test/modes";

Deno.test("legacy-modes: an ESM package that only has an exports map", () => {
  const language = goLanguage();
  if (!language || typeof language.parser !== "object") {
    throw new Error("expected a StreamLanguage with a parser");
  }
});
