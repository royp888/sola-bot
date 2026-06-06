import { request } from "@/api/http";
import type { StatsOverview } from "@/types/api";

interface StatsOverviewResponse {
  total_chats: number;
  total_posts: number;
  total_schedules: number;
  total_members: number;
  active_users: number;
  points_issued: number;
  open_tasks: number;
}

interface ActivityResponse {
  date: string;
  messages: number;
  commands: number;
  new_members: number;
  leaving_users: number;
}

interface PointsResponse {
  rank: number;
  user_id: number;
  username?: string;
  nickname?: string;
  points: number;
}

export function fetchStats(range: string): Promise<StatsOverview> {
  const { from, to } = resolveRange(range);
  const query = new URLSearchParams({ from, to }).toString();

  return Promise.all([
    request<StatsOverviewResponse>(`/stats/overview?${query}`),
    request<ActivityResponse[]>(`/stats/activity?${query}`),
    request<PointsResponse[]>(`/stats/points?${query}`),
  ]).then(([overview, activity, points]) => {
    const maxActivity = Math.max(
      ...activity.map((item) => item.messages + item.commands + item.new_members),
      1,
    );

    return {
      metrics: [
        { label: "绑定群组", value: String(overview.total_chats), delta: "实时", tone: "primary" },
        { label: "活跃用户", value: String(overview.active_users), delta: "所选周期", tone: "success" },
        { label: "发放积分", value: String(overview.points_issued), delta: "所选周期", tone: "warning" },
        { label: "定时任务", value: String(overview.total_schedules), delta: `${overview.open_tasks} 个待执行`, tone: "primary" },
      ],
      series: activity.map((item) => ({
        label: item.date.slice(5),
        value: Math.round(((item.messages + item.commands + item.new_members) / maxActivity) * 100),
      })),
      topPointsUsers: points.slice(0, 10).map((item) => ({
        rank: item.rank,
        user_id: item.user_id,
        label: item.username || item.nickname || String(item.user_id),
        points: item.points,
        share: overview.points_issued > 0 ? Math.round((item.points / overview.points_issued) * 100) : 0,
      })),
      sources: [
        { label: "积分发放量", value: overview.points_issued, color: "#5ecdc3" },
        { label: "命令调用数", value: activity.reduce((sum, item) => sum + item.commands, 0), color: "#f0b35d" },
      ],
    };
  });
}

function resolveRange(range: string): { from: string; to: string } {
  const days = range === "90d" ? 90 : range === "30d" ? 30 : 7;
  const to = new Date();
  const from = new Date(to);
  from.setDate(from.getDate() - days + 1);
  return {
    from: toDateString(from),
    to: toDateString(to),
  };
}

function toDateString(value: Date): string {
  return value.toISOString().slice(0, 10);
}
