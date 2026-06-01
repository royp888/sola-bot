import type {
  BotRecord,
  BanRecord,
  ChatID,
  ChatAdminConfig,
  ChatPointConfig,
  ChatRecord,
  DashboardSummary,
  LotteryRecord,
  PointLogRecord,
  PointRankRecord,
  ScheduledPostRecord,
  StatsOverview,
  UserProfile,
  UserRecord,
  WarnRecord,
} from "@/types/api";

export const demoUser: UserProfile = {
  id: "usr_demo",
  name: "Sola Admin",
  email: "admin@botops.local",
  role: "owner",
  language: "zh-CN",
};

export function createDemoDashboardSummary(): DashboardSummary {
  return {
    metrics: [
      { label: "今日消息", value: "18.2k", delta: "+12.4%", tone: "primary" },
      { label: "在线 Bot", value: "6", delta: "100%", tone: "success" },
      { label: "待发任务", value: "14", delta: "+3", tone: "warning" },
      { label: "异常告警", value: "2", delta: "-1", tone: "danger" },
    ],
    activity: [
      { title: "频道同步完成", detail: "YC 朋友圈 与频道已刷新成员统计", time: "2 分钟前", status: "success" },
      { title: "定时发布排队", detail: "晚间推送将于 20:30 执行", time: "13 分钟前", status: "info" },
      { title: "积分接口限流", detail: "来自 @newsdesk 的短时高频请求被限速", time: "28 分钟前", status: "warning" },
      { title: "关键词规则命中", detail: "spam 规则已写入违规记录", time: "1 小时前", status: "success" },
    ],
    jobs: [
      { title: "晚间快讯", schedule: "每日 20:30", nextRun: "今天 20:30", status: "live" },
      { title: "签到积分结算", schedule: "每 10 分钟", nextRun: "11:20", status: "live" },
      { title: "群公告刷新", schedule: "每周一 09:00", nextRun: "下周一 09:00", status: "paused" },
    ],
    health: [
      { label: "API 延迟", value: 82, note: "最近 5 分钟平均" },
      { label: "队列积压", value: 41, note: "Redis worker backlog" },
      { label: "命令成功率", value: 96, note: "24 小时窗口" },
    ],
  };
}

export function createDemoBots(): BotRecord[] {
  return [
    {
      id: "bot_1",
      name: "Sola Core",
      username: "@sola_core_bot",
      status: "online",
      boundChats: 18,
      lastHeartbeat: "刚刚",
      language: "zh-CN",
    },
    {
      id: "bot_2",
      name: "YC Manager",
      username: "@yc_manager_bot",
      status: "degraded",
      boundChats: 7,
      lastHeartbeat: "6 分钟前",
      language: "zh-CN",
    },
    {
      id: "bot_3",
      name: "Ops Mirror",
      username: "@ops_mirror_bot",
      status: "offline",
      boundChats: 2,
      lastHeartbeat: "42 分钟前",
      language: "en-US",
    },
  ];
}

export function createDemoChats(): ChatRecord[] {
  return [
    {
      id: "chat_1",
      title: "YC 朋友圈",
      kind: "channel",
      permission: "管理员 + 发布",
      members: 147641,
      syncStatus: "synced",
      owner: "富贵电子",
    },
    {
      id: "chat_2",
      title: "项目群 Alpha",
      kind: "group",
      permission: "消息 + 积分",
      members: 4287,
      syncStatus: "pending",
      owner: "运营组",
    },
    {
      id: "chat_3",
      title: "外部合作频道",
      kind: "channel",
      permission: "只读 + 统计",
      members: 31820,
      syncStatus: "blocked",
      owner: "渠道组",
    },
  ];
}

export function createDemoPointConfig(chatId: ChatID): ChatPointConfig {
  return {
    chat_id: chatId,
    enabled: true,
    cooldown_seconds: 60,
    point_text: 1,
    point_photo: 3,
    point_sticker: 2,
    point_video: 3,
    point_file: 2,
    point_voice: 3,
  };
}

export function createDemoStats(): StatsOverview {
  return {
    metrics: [
      { label: "活跃用户", value: "12.8k", delta: "+8.2%", tone: "primary" },
      { label: "命令调用", value: "94.1k", delta: "+4.9%", tone: "success" },
      { label: "消息触达", value: "2.3M", delta: "+15.1%", tone: "warning" },
      { label: "失败率", value: "0.7%", delta: "-0.2%", tone: "danger" },
    ],
    series: [
      { label: "周一", value: 56 },
      { label: "周二", value: 68 },
      { label: "周三", value: 82 },
      { label: "周四", value: 74 },
      { label: "周五", value: 93 },
      { label: "周六", value: 66 },
      { label: "周日", value: 71 },
    ],
    topPointsUsers: [
      { rank: 1, user_id: 883001, label: "@mika_ops", points: 18942, share: 38 },
      { rank: 2, user_id: 883002, label: "@alex_push", points: 11033, share: 22 },
      { rank: 3, user_id: 883003, label: "@cyan", points: 7240, share: 14 },
      { rank: 4, user_id: 883004, label: "@nova", points: 5218, share: 10 },
    ],
    sources: [
      { label: "群组入口", value: 48, color: "#5ecdc3" },
      { label: "频道入口", value: 31, color: "#f0b35d" },
      { label: "私聊入口", value: 21, color: "#7d8ca8" },
    ],
  };
}

export function createDemoUsers(): UserRecord[] {
  return [
    {
      id: 883001,
      username: "@mika_ops",
      display_name: "Mika",
      chat_id: "chat_2",
      total_points: 12880,
      status: "active",
      last_seen_at: "2026-05-29 19:35",
    },
    {
      id: 883002,
      username: "@chen_builds",
      display_name: "Chen",
      chat_id: "chat_2",
      total_points: 9420,
      status: "active",
      last_seen_at: "2026-05-29 18:12",
    },
    {
      id: 883003,
      username: "@slow_reply",
      display_name: "Lowkey",
      chat_id: "chat_1",
      total_points: 2100,
      status: "muted",
      last_seen_at: "2026-05-28 23:07",
    },
  ];
}

export function createDemoPointRank(): PointRankRecord[] {
  return createDemoUsers().map((user, index) => ({
    rank: index + 1,
    user_id: user.id,
    username: user.username,
    total_points: user.total_points,
  }));
}

export function createDemoPointLogs(chatId: ChatID, userId: ChatID = 883001): PointLogRecord[] {
  return [
    {
      id: 1,
      user_id: userId,
      username: "@mika_ops",
      chat_id: chatId,
      delta: 3,
      reason: "photo",
      created_at: "2026-05-29 19:22",
    },
    {
      id: 2,
      user_id: userId,
      username: "@mika_ops",
      chat_id: chatId,
      delta: 1,
      reason: "text",
      created_at: "2026-05-29 18:54",
    },
    {
      id: 3,
      user_id: 883002,
      username: "@chen_builds",
      chat_id: chatId,
      delta: -30,
      reason: "manual_adjust",
      created_at: "2026-05-29 17:40",
    },
  ];
}

export function createDemoAdminConfig(chatId: ChatID): ChatAdminConfig {
  return {
    chat_id: chatId,
    welcome_text: "欢迎 {name} 加入！",
    verify_enabled: true,
    verify_timeout: 60,
    warn_limit: 3,
    updated_at: "2026-05-29 18:00",
  };
}

export function createDemoBans(chatId: ChatID): BanRecord[] {
  return [
    {
      id: 1,
      user_id: 772001,
      username: "@spam_case",
      chat_id: chatId,
      reason: "广告刷屏",
      banned_by: 883001,
      banned_at: "2026-05-29 11:04",
    },
    {
      id: 2,
      user_id: 772002,
      username: "@risk_user",
      chat_id: chatId,
      reason: "多次触发警告上限",
      banned_by: 883002,
      banned_at: "2026-05-28 21:16",
      unbanned_at: "2026-05-29 09:30",
    },
  ];
}

export function createDemoWarns(chatId: ChatID, userId: ChatID): WarnRecord[] {
  return [
    {
      id: 1,
      user_id: userId,
      username: "@risk_user",
      chat_id: chatId,
      reason: "重复发送相同内容",
      warned_by: 883001,
      created_at: "2026-05-29 12:20",
      cleared: false,
    },
    {
      id: 2,
      user_id: userId,
      username: "@risk_user",
      chat_id: chatId,
      reason: "外链未审核",
      warned_by: 883001,
      created_at: "2026-05-29 13:02",
      cleared: true,
    },
  ];
}

export function createDemoScheduledPosts(): ScheduledPostRecord[] {
  return [
    {
      id: 9001,
      chat_id: "chat_1",
      title: "晚间快讯",
      content: "今日群内精选内容汇总",
      media_type: "text",
      cron_expr: "30 20 * * *",
      enabled: true,
      last_run_at: "2026-05-28 20:30",
      created_at: "2026-05-20 10:00",
    },
    {
      id: 9002,
      chat_id: "chat_2",
      title: "规则更新提醒",
      content: "本周关键词规则已更新",
      media_type: "photo",
      media_url: "https://example.com/banner.jpg",
      run_once_at: "2026-05-30 10:00",
      enabled: false,
      created_at: "2026-05-27 15:42",
    },
  ];
}

export function createDemoLotteries(): LotteryRecord[] {
  return [
    {
      id: 1201,
      chat_id: "chat_2",
      title: "周末活跃抽奖",
      prize: "运营工具包",
      cost_points: 30,
      max_participants: 300,
      winner_count: 3,
      end_at: "2026-05-31 21:00",
      status: "active",
      created_by: 883001,
      created_at: "2026-05-29 10:00",
      participants: 184,
    },
    {
      id: 1202,
      chat_id: "chat_1",
      title: "新成员欢迎礼",
      prize: "置顶位 5 个",
      cost_points: 0,
      max_participants: 0,
      winner_count: 5,
      end_at: "2026-06-02 12:00",
      status: "active",
      created_by: 883002,
      created_at: "2026-05-28 16:10",
      participants: 97,
    },
  ];
}
