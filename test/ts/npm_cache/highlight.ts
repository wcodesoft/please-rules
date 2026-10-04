import core from "highlight.js/lib/core";
import go from "highlight.js/lib/languages/go";

const hljs: any = core;
hljs.registerLanguage("go", go);

/** Highlights Go source as HTML. */
export function highlight(source: string): string {
  return hljs.highlight(source, { language: "go" }).value;
}
