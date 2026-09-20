// Direct imports without barrel index.ts (subpath mapping)
import { createBadge } from "@repo/dashboard/components/Badge";
import { formatMetricCard } from "@repo/dashboard/components/MetricCard";
import { renderModal } from "@repo/dashboard/components/Modal";

// Also test with explicit extension
import { createBadge as createBadgeWithExt } from "@repo/dashboard/components/Badge.ts";
import { formatMetricCard as formatMetricCardWithExt } from "@repo/dashboard/components/MetricCard.ts";

// Test auto-derived module name imports (derives from test/ts/lib/components/auto_named)
import { createBadge as autoBadge } from "test/ts/lib/components/auto_named/Badge";
import { formatMetricCard as autoMetric } from "test/ts/lib/components/auto_named/MetricCard";

Deno.test("direct component imports work without barrel files", () => {
  const badge = createBadge({ text: "Active", variant: "success" });
  if (badge !== "[Badge:success] Active") {
    throw new Error(`unexpected badge: ${badge}`);
  }

  const metric = formatMetricCard({ title: "CPU", value: 99 });
  if (metric !== "CPU: 99") {
    throw new Error(`unexpected metric: ${metric}`);
  }

  const modal = renderModal({ title: "Settings", isOpen: true });
  if (modal !== "Modal(Settings)") {
    throw new Error(`unexpected modal: ${modal}`);
  }
});

Deno.test("direct component imports with .ts extension work", () => {
  const badge = createBadgeWithExt({ text: "Pending" });
  if (badge !== "[Badge:default] Pending") {
    throw new Error(`unexpected badge with ext: ${badge}`);
  }

  const metric = formatMetricCardWithExt({ title: "RAM", value: 42 });
  if (metric !== "RAM: 42") {
    throw new Error(`unexpected metric with ext: ${metric}`);
  }
});

Deno.test("auto-derived module_name imports work", () => {
  const badge = autoBadge({ text: "Auto" });
  if (badge !== "[Badge:default] Auto") {
    throw new Error(`unexpected auto badge: ${badge}`);
  }

  const metric = autoMetric({ title: "Disk", value: 10 });
  if (metric !== "Disk: 10") {
    throw new Error(`unexpected auto metric: ${metric}`);
  }
});
