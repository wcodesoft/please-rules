import { go } from "@codemirror/legacy-modes/mode/go";
import { StreamLanguage } from "@codemirror/language";

/** A CodeMirror language for Go, built from a legacy mode. */
export function goLanguage(): StreamLanguage<unknown> {
  return StreamLanguage.define(go);
}
