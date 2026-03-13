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
