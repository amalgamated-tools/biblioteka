export interface User {
  id: string;
  email: string;
  oidc_linked: boolean;
  is_admin: boolean;
}

export type ArrServiceType = "radarr" | "sonarr" | "prowlarr" | "seerr";

export interface ArrService {
  id: string;
  name: string;
  type: ArrServiceType;
  url: string;
  api_key: string;
  created_at: string;
  updated_at: string;
}

export interface ArrServiceInput {
  name: string;
  type: ArrServiceType;
  url: string;
  api_key: string;
}

export interface Movie {
  id: string;
  title: string;
  overview?: string;
  year?: number;
  tmdb_id?: number;
  poster_url?: string;
  status: string;
  created_at: string;
}

export interface MovieSearchResult {
  tmdb_id: number;
  title: string;
  year: number;
  overview: string;
  poster_url: string;
  liked: boolean;
  id?: string;
}

export interface WatchProvider {
  id: number;
  name: string;
  logo_url: string;
}

export interface MovieProviders {
  tmdb_id: number;
  link?: string;
  stream: WatchProvider[];
  rent: WatchProvider[];
  buy: WatchProvider[];
}

export interface TvSeries {
  id: string;
  title: string;
  overview?: string;
  year?: number;
  tmdb_id?: number;
  poster_url?: string;
  status: string;
  created_at: string;
}

export interface TvSeriesSearchResult {
  tmdb_id: number;
  title: string;
  year: number;
  overview: string;
  poster_url: string;
  liked: boolean;
  id?: string;
}

export interface TvSeriesProviders {
  tmdb_id: number;
  link?: string;
  stream: WatchProvider[];
  buy: WatchProvider[];
}

export interface StreamingProvider {
  provider_id: number;
  provider_name: string;
  logo_path: string;
  display_priority: number;
}
