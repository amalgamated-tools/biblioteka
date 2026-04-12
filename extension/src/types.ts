// Shared types used by both the popup and the background service worker.

export interface Settings {
  serverURL: string;
  apiKey: string;
  defaultLibraryID: string;
}

export interface Library {
  id: string;
  name: string;
}

export interface CaptureRequest {
  url: string;
  library_id: string;
  title?: string;
  author?: string;
}

export interface CaptureResponse {
  message: string;
  url: string;
  library_id: string;
}

export interface ErrorResponse {
  error: string;
}
