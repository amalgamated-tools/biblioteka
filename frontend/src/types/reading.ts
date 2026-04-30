export interface ReadingProgressItem {
  document: string;
  percentage: number;
  device?: string;
  last_synced: string;
  estimated_minutes_remaining?: number;
}

export interface ReadingProgressStats {
  current_streak: number;
  total_tracked: number;
  total_finished: number;
  in_progress: ReadingProgressItem[];
}

export interface MonthlyDownloads {
  month: string; // "YYYY-MM"
  count: number;
}

export interface YearInBooks {
  year: number;
  books_finished: number;
  active_days: number;
  longest_streak: number;
  total_downloads: number;
}

export interface ReadingGroup {
  id: string;
  owner_id: string;
  name: string;
  description: string | null;
  member_count: number;
  created_at: string;
  updated_at: string;
}

export interface ReadingGroupInput {
  name: string;
  description?: string | null;
}

export interface ReadingGroupUpdateInput {
  name: string;
  description: string | null;
}

export interface ReadingGroupMember {
  group_id: string;
  user_id: string;
  user_name: string;
  role: "owner" | "member";
  joined_at: string;
}

export interface GroupMemberProgress {
  user_id: string;
  user_name: string;
  percentage: number;
  updated_at: string | null;
}

export interface ReadingList {
  id: string;
  name: string;
  description: string | null;
  book_count: number;
  created_at: string;
  updated_at: string;
}

export interface ReadingListInput {
  name: string;
  description?: string | null;
}
