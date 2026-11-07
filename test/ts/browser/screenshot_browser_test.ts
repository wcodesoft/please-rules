/// <reference lib="dom" />
declare const Deno: {
  test: (name: string, fn: () => void | Promise<void>) => void;
};
declare const expectScreenshot: (
  name: string,
  options?: { selector?: string; fullPage?: boolean },
) => Promise<void>;

// The page has no text, so the goldens do not depend on the fonts of the machine.
function mount(): () => void {
  const style = document.createElement("style");
  style.textContent = `
    body { margin: 0; background: rgb(240, 240, 240); }
    .card { width: 120px; height: 60px; margin: 10px; border: 4px solid rgb(30, 60, 120);
            border-radius: 12px; background: linear-gradient(90deg, rgb(200, 0, 0), rgb(0, 0, 200)); }
    .tall { height: 900px; width: 40px; margin: 10px; background: rgb(0, 150, 80); }
  `;
  document.head.appendChild(style);
  const container = document.createElement("div");
  container.innerHTML = `<div class="card"></div><div class="tall"></div>`;
  document.body.appendChild(container);
  return () => {
    document.body.removeChild(container);
    document.head.removeChild(style);
  };
}

Deno.test("an element screenshot matches the golden", async () => {
  const unmount = mount();
  await expectScreenshot("card", { selector: ".card" });
  unmount();
});

Deno.test("a full page screenshot covers content below the viewport", async () => {
  const unmount = mount();
  await expectScreenshot("page", { fullPage: true });
  unmount();
});
