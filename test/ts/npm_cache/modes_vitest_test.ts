import { describe, expect, it } from "vitest";
import { goLanguage } from "@test/modes";

describe("an exports-only npm package and its dependency closure under Vitest", () => {
  it("builds a CodeMirror language from a legacy mode", () => {
    expect(goLanguage().name).toBe("go");
  });
});
