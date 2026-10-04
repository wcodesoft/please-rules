import { goLanguage } from "@test/modes_auto";

Deno.test("legacy-modes (auto-resolved): an ESM package that only has an exports map", () => {
  const language = goLanguage();
  if (!language || typeof language.parser !== "object") {
    throw new Error("expected a StreamLanguage with a parser");
  }
});
