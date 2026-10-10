// Profile API Types

export interface ProfileUpdateData {
  name: string;
}

export interface ProfileUpdateResponse {
  message: string;
  name: string;
}

export interface PhotoUploadResponse {
  url: string;
  filename: string;
  message: string;
}
