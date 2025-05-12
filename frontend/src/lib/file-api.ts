// src/lib/file-api.ts
import { File, Folder } from "@/types/file-types";
import {
  ListDirectory,
  CreateFolder,
  RenameItem,
  DeleteItem,
  UploadFile,
  DownloadFile,
  CopyItem,
  MoveItem,
  RefrigerateFile,
  UnrefrigerateFile,
  GetFileInfo,
} from "@/../wailsjs/go/kfiles/FileBrowser";

// Define return type for the ListDirectory method
export interface DirectoryContents {
  files: File[];
  folders: Folder[];
}

// Wrapper for calling the FileBrowser Go backend
export class FileBrowserAPI {
  // List contents of a directory
  static async listDirectory(path: string = "/"): Promise<DirectoryContents> {
    try {
      const items = await ListDirectory(path);

      // Separate files and folders
      const files: File[] = [];
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
  static async uploadFile(path: string, file: File): Promise<void> {
    try {
      const buffer = await file.arrayBuffer();
      const bytes = new Uint8Array(buffer);
      await UploadFile(path, file.name, bytes);
    } catch (error) {
      console.error("Error uploading file:", error);
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

  // Copy a file or folder
  static async copyItem(sourcePath: string, destinationPath: string): Promise<void> {
    try {
      await CopyItem(sourcePath, destinationPath);
    } catch (error) {
      console.error("Error copying item:", error);
      throw error;
    }
  }

  // Move a file or folder
  static async moveItem(sourcePath: string, destinationPath: string): Promise<void> {
    try {
      await MoveItem(sourcePath, destinationPath);
    } catch (error) {
      console.error("Error moving item:", error);
      throw error;
    }
  }

  // Refrigerate (compress) a file
  static async refrigerateFile(path: string): Promise<void> {
    try {
      await RefrigerateFile(path);
    } catch (error) {
      console.error("Error refrigerating file:", error);
      throw error;
    }
  }

  // Unrefrigerate (decompress) a file
  static async unrefrigerateFile(path: string): Promise<void> {
    try {
      await UnrefrigerateFile(path);
    } catch (error) {
      console.error("Error unrefrigerating file:", error);
      throw error;
    }
  }

  // Get file or folder info
  static async getFileInfo(path: string): Promise<File | Folder> {
    try {
      const item = await GetFileInfo(path);

      if (item.type === "folder") {
        return {
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
        };
      } else {
        return {
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
        };
      }
    } catch (error) {
      console.error("Error getting file info:", error);
      throw error;
    }
  }
}
