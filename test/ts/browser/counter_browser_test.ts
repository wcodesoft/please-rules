/// <reference lib="dom" />
import { CounterComponent } from "@test/browser/counter";

declare const Deno: {
  test: (name: string, fn: () => void | Promise<void>) => void;
};
declare const expect: (actual: any) => {
  toBe: (expected: any) => void;
  toEqual: (expected: any) => void;
  toBeTruthy: () => void;
  toBeFalsy: () => void;
};

Deno.test("CounterComponent initializes with default value", () => {
  const container = document.createElement("div");
  const body = document.body || document.documentElement;
  body.appendChild(container);

  const counter = new CounterComponent(container);
  expect(counter.getValue()).toBe(0);

  const valueEl = container.querySelector(".counter-value");
  expect(valueEl?.textContent).toBe("0");

  body.removeChild(container);
});

Deno.test("CounterComponent increments, decrements, and resets", () => {
  const container = document.createElement("div");
  const body = document.body || document.documentElement;
  body.appendChild(container);

  const counter = new CounterComponent(container, { initial: 10, step: 2 });
  expect(counter.getValue()).toBe(10);

  const incBtn = container.querySelector<HTMLButtonElement>(".btn-increment")!;
  const decBtn = container.querySelector<HTMLButtonElement>(".btn-decrement")!;
  const resetBtn = container.querySelector<HTMLButtonElement>(".btn-reset")!;
  const valueEl = container.querySelector(".counter-value")!;

  incBtn.click();
  expect(counter.getValue()).toBe(12);
  expect(valueEl.textContent).toBe("12");

  decBtn.click();
  expect(counter.getValue()).toBe(10);
  expect(valueEl.textContent).toBe("10");

  resetBtn.click();
  expect(counter.getValue()).toBe(0);
  expect(valueEl.textContent).toBe("0");

  body.removeChild(container);
});

Deno.test("Browser Web APIs localStorage and userAgent work hermetically", () => {
  localStorage.setItem("please_browser_test_key", "hermetic_chrome");
  expect(localStorage.getItem("please_browser_test_key")).toBe("hermetic_chrome");
  localStorage.removeItem("please_browser_test_key");
  expect(localStorage.getItem("please_browser_test_key")).toBe(null);

  expect(typeof window.location.href).toBe("string");
  expect(navigator.userAgent.includes("Chrome")).toBe(true);
});
