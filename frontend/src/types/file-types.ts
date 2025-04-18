
export interface File {
  id: string;
  name: string;
  type: "file";
  extension: string;
  path: string;
  size: string;
  sizeInBytes: number;
  lastModified: string;
  lastModifiedDate: Date;
  isRefrigerated: boolean;
  compressionRatio?: number;
  owner: string;
  isShared: boolean;
  sharedWith?: string[];
  tags?: string[];
}

export interface Folder {
  id: string;
  name: string;
  type: "folder";
  path: string;
  itemCount: number;
  lastModified: string;
  lastModifiedDate: Date;
  owner: string;
  isShared: boolean;
  sharedWith?: string[];
  tags?: string[];
}
