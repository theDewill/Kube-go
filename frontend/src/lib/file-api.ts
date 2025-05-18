// src/lib/file-api.ts
import { File as FileType, Folder } from "@/types/file-types";
import {
  ListDirectory,
  CreateFolder,
  RenameItem,
  DeleteItem,
  UploadFile,
  DownloadFile,
  UploadFileWithModel,
  SearchFiles,
  ReadSettingsFile,
  WriteSettingsFile,
} from "@/../wailsjs/go/kfiles/FileBrowser";
import { GetNodesForFrontend } from "@/../wailsjs/go/kubenet/NodeRegistry";

// Define return type for the ListDirectory method
export interface DirectoryContents {
  files: FileType[];
  folders: Folder[];
}

export interface SearchResult {
  file: FileType;
  score: number;
  matchType: string;
  excerpt: string;
}

// Wrapper for calling the FileBrowser Go backend
export class FileBrowserAPI {
  // List contents of a directory
  static async listDirectory(path: string = "/"): Promise<DirectoryContents> {
    try {
      const items = await ListDirectory(path);

      // Separate files and folders
      const files: FileType[] = [];
      const folders: Folder[] = [];

      for (const item of items) {
        if (item.type === "folder") {
          folders.push({
            id: item.id,
            name: item.name,
            type: "folder",
            path: item.path,
            itemCount: item.itemCount,
            lastModified: item.lastModified,
            lastModifiedDate: new Date(item.lastModifiedDate),
            owner: item.owner,
            isShared: item.isShared,
            sharedWith: item.sharedWith || [],
          });
        } else {
          files.push({
            id: item.id,
            name: item.name,
            type: "file",
            extension: item.extension,
            path: item.path,
            size: item.size,
            sizeInBytes: item.sizeInBytes,
            lastModified: item.lastModified,
            lastModifiedDate: new Date(item.lastModifiedDate),
            isRefrigerated: item.isRefrigerated,
            compressionRatio: item.compressionRatio,
            owner: item.owner,
            isShared: item.isShared,
            sharedWith: item.sharedWith || [],
            isDistributed: item.isDistributed || false,
          });
        }
      }

      return { files, folders };
    } catch (error) {
      console.error("Error listing directory:", error);
      throw error;
    }
  }

  // Create a new folder
  static async createFolder(path: string, name: string): Promise<void> {
    try {
      await CreateFolder(path, name);
    } catch (error) {
      console.error("Error creating folder:", error);
      throw error;
    }
  }

  // Rename a file or folder
  static async renameItem(path: string, newName: string): Promise<void> {
    try {
      await RenameItem(path, newName);
    } catch (error) {
      console.error("Error renaming item:", error);
      throw error;
    }
  }

  // Delete a file or folder
  static async deleteItem(path: string): Promise<void> {
    try {
      await DeleteItem(path);
    } catch (error) {
      console.error("Error deleting item:", error);
      throw error;
    }
  }

  // Upload a file
  // static async uploadFile(path: string, file: File, distribute: boolean = false): Promise<void> {
  //   try {
  //     const buffer = await file.arrayBuffer();
  //     const bytes = new Uint8Array(buffer);
  //     // Convert Uint8Array to regular array for Wails compatibility
  //     const byteArray = Array.from(bytes);
  //     await UploadFile(path, file.name, byteArray, distribute);
  //   } catch (error) {
  //     console.error("Error uploading file:", error);
  //     throw error;
  //   }
  // }
  //
  static async uploadFile(
    path: string,
    file: File,
    distribute: boolean = false,
    modelType: string = "ollama",
  ): Promise<void> {
    try {
      const buffer = await file.arrayBuffer();
      const bytes = new Uint8Array(buffer);
      // Convert Uint8Array to regular array for Wails compatibility
      const byteArray = Array.from(bytes);
      await UploadFileWithModel(path, file.name, byteArray, distribute, modelType);
    } catch (error) {
      console.error("Error uploading file:", error);
      throw error;
    }
  }

  static async searchFiles(query: string, limit: number = 20): Promise<SearchResult[]> {
    try {
      return await SearchFiles(query, limit);
    } catch (error) {
      console.error("Error searching files:", error);
      throw error;
    }
  }

  // SMart Finder
  static async downloadSettingsFile(): Promise<Blob> {
    try {
      const data = await ReadSettingsFile();
      return new Blob([data], { type: "application/json" });
    } catch (error) {
      console.error("Error downloading settings file:", error);
      throw error;
    }
  }

  static async uploadSettingsFile(data: ArrayBuffer): Promise<void> {
    try {
      const bytes = new Uint8Array(data);
      const byteArray = Array.from(bytes);
      await WriteSettingsFile(byteArray);
    } catch (error) {
      console.error("Error uploading settings file:", error);
      throw error;
    }
  }

  // Download a file
  static async downloadFile(path: string): Promise<Blob> {
    try {
      const data = await DownloadFile(path);
      return new Blob([data]);
    } catch (error) {
      console.error("Error downloading file:", error);
      throw error;
    }
  }

  // Get nodes in the network
  static async getNetworkNodes(): Promise<any[]> {
    try {
      return await GetNodesForFrontend();
    } catch (error) {
      console.error("Error getting network nodes:", error);
      throw error;
    }
  }
}
