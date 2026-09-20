export interface BadgeProps {
  text: string;
  variant?: "success" | "warning" | "error";
}

export function createBadge(props: BadgeProps): string {
  return `[Badge:${props.variant ?? "default"}] ${props.text}`;
}
