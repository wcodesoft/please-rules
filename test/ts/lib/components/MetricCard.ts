export interface MetricCardProps {
  title: string;
  value: number;
}

export function formatMetricCard(props: MetricCardProps): string {
  return `${props.title}: ${props.value}`;
}
