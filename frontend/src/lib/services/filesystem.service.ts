import { apiClient } from "$lib/api";

export interface FileItem {
  id: string;
  name: string;
  type: "file" | "folder";
  path: string;
  size?: number;
  modified?: string;
  extension?: string;
  isHidden: boolean;
}

export interface ListDirectoryResponse {
  path: string;
  items: FileItem[];
  parentDir?: string;
}

export interface FileContentResponse {
  path: string;
  name: string;
  content: string;
  size: number;
  mimeType: string;
}

export interface QuickAccessPath {
  name: string;
  path: string;
  icon: string;
}

/**
 * Helper to handle API responses
 */
async function handleResponse<T>(response: Response): Promise<T> {
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || data.detail || "Request failed");
  }
  return data;
}

/**
 * Filesystem service for interacting with the backend filesystem API
 */
export const filesystemService = {
  /**
   * List contents of a directory
   */
  async listDirectory(
    path: string = "~",
    showHidden: boolean = false,
  ): Promise<ListDirectoryResponse> {
    const params = new URLSearchParams({
      path,
      showHidden: String(showHidden),
    });
    const response = await apiClient.get(`/filesystem/list?${params}`);
    return handleResponse<ListDirectoryResponse>(response);
  },

  /**
   * Read the content of a file (for text preview)
   */
  async readFile(path: string): Promise<FileContentResponse> {
    const params = new URLSearchParams({ path });
    const response = await apiClient.get(`/filesystem/read?${params}`);
    return handleResponse<FileContentResponse>(response);
  },

  /**
   * Get download URL for a file
   */
  getDownloadUrl(path: string): string {
    const params = new URLSearchParams({ path });
    return `/api/filesystem/download?${params}`;
  },

  /**
   * Get information about a file or directory
   */
  async getFileInfo(path: string): Promise<FileItem> {
    const params = new URLSearchParams({ path });
    const response = await apiClient.get(`/filesystem/info?${params}`);
    return handleResponse<FileItem>(response);
  },

  /**
   * Get quick access paths (Home, Desktop, Documents, etc.)
   */
  async getQuickAccessPaths(): Promise<{ paths: QuickAccessPath[] }> {
    const response = await apiClient.get("/filesystem/quick-access");
    return handleResponse<{ paths: QuickAccessPath[] }>(response);
  },

  /**
   * Create a new directory
   */
  async createDirectory(parentPath: string, name: string): Promise<FileItem> {
    const response = await apiClient.post("/filesystem/mkdir", {
      path: parentPath,
      name,
    });
    return handleResponse<FileItem>(response);
  },

  /**
   * Delete a file or empty directory
   */
  async delete(path: string): Promise<void> {
    const params = new URLSearchParams({ path });
    const response = await apiClient.delete(`/filesystem/delete?${params}`);
    if (!response.ok) {
      const data = await response.json();
      throw new Error(data.error || "Delete failed");
    }
  },

  /**
   * Upload a file to a directory
   */
  async uploadFile(destPath: string, file: File): Promise<FileItem> {
    const formData = new FormData();
    formData.append("path", destPath);
    formData.append("file", file);

    const response = await apiClient.postFormData(
      "/filesystem/upload",
      formData,
    );
    return handleResponse<FileItem>(response);
  },

  /**
   * Format file size for display
   */
  formatSize(bytes?: number): string {
    if (!bytes) return "-";
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024)
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  },

  /**
   * Format date for display
   */
  formatDate(dateStr?: string): string {
    if (!dateStr) return "-";
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  },

  /**
   * Get file icon info based on extension
   */
  getFileIcon(extension?: string): { icon: string; color: string } {
    const icons: Record<string, { icon: string; color: string }> = {
      pdf: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#E53935",
      },
      doc: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#1565C0",
      },
      docx: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#1565C0",
      },
      xls: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#2E7D32",
      },
      xlsx: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#2E7D32",
      },
      md: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#455A64",
      },
      json: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#FFA000",
      },
      js: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#F7DF1E",
      },
      ts: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#3178C6",
      },
      py: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#3776AB",
      },
      go: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#00ADD8",
      },
      rs: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#CE422B",
      },
      html: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#E44D26",
      },
      css: {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#264DE4",
      },
      png: {
        icon: "M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z",
        color: "#7B1FA2",
      },
      jpg: {
        icon: "M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z",
        color: "#7B1FA2",
      },
      jpeg: {
        icon: "M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z",
        color: "#7B1FA2",
      },
      gif: {
        icon: "M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z",
        color: "#7B1FA2",
      },
      svg: {
        icon: "M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z",
        color: "#FFB300",
      },
      mp4: {
        icon: "M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z",
        color: "#9C27B0",
      },
      mp3: {
        icon: "M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3",
        color: "#E91E63",
      },
      zip: {
        icon: "M8 4H6a2 2 0 00-2 2v12a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-2m-4-1v8m0 0l3-3m-3 3L9 8",
        color: "#795548",
      },
      dmg: {
        icon: "M8 4H6a2 2 0 00-2 2v12a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-2m-4-1v8m0 0l3-3m-3 3L9 8",
        color: "#546E7A",
      },
      txt: {
        icon: "M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z",
        color: "#607D8B",
      },
      log: {
        icon: "M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z",
        color: "#607D8B",
      },
    };
    return (
      icons[extension?.toLowerCase() || ""] || {
        icon: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z",
        color: "#78909C",
      }
    );
  },
};

export default filesystemService;
