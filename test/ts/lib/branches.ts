export function classify(n: number): string {
  if (n < 0) {
    return "neg";
  } else if (n === 0) {
    return "zero";
  } else {
    return "pos";
  }
}

export function both(a: boolean, b: boolean): boolean {
  return a && b;
}

export function unused(): number {
  return 42;
}
