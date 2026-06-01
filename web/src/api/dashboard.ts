import { request } from "@/api/http";
import type { DashboardSummary } from "@/types/api";

export function fetchDashboardSummary(): Promise<DashboardSummary> {
  return request<DashboardSummary>("/dashboard/summary");
}
