/// <reference lib="dom" />

export interface CounterOptions {
  initial?: number;
  step?: number;
}

export class CounterComponent {
  private count: number;
  private step: number;
  private root: HTMLElement;
  private countEl: HTMLElement;

  constructor(root: HTMLElement, options: CounterOptions = {}) {
    this.count = options.initial ?? 0;
    this.step = options.step ?? 1;
    this.root = root;

    this.root.innerHTML = `
      <div class="counter-container">
        <span class="counter-value">${this.count}</span>
        <button class="btn-increment">+ Increment</button>
        <button class="btn-decrement">- Decrement</button>
        <button class="btn-reset">Reset</button>
      </div>
    `;

    this.countEl = this.root.querySelector(".counter-value")!;
    const incBtn = this.root.querySelector(".btn-increment")!;
    const decBtn = this.root.querySelector(".btn-decrement")!;
    const resetBtn = this.root.querySelector(".btn-reset")!;

    incBtn.addEventListener("click", () => this.increment());
    decBtn.addEventListener("click", () => this.decrement());
    resetBtn.addEventListener("click", () => this.reset());
  }

  increment(): number {
    this.count += this.step;
    this.update();
    return this.count;
  }

  decrement(): number {
    this.count -= this.step;
    this.update();
    return this.count;
  }

  reset(): void {
    this.count = 0;
    this.update();
  }

  getValue(): number {
    return this.count;
  }

  private update(): void {
    this.countEl.textContent = String(this.count);
  }
}
