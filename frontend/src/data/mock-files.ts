
import { File, Folder } from "@/types/file-types";

// Generate a random date in the last 30 days
const randomDate = () => {
  const date = new Date();
  date.setDate(date.getDate() - Math.floor(Math.random() * 30));
  return date;
};

// Format date for display
const formatDate = (date: Date) => {
  const now = new Date();
  const diffInDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24));
  
  if (diffInDays === 0) {
    return "Today";
  } else if (diffInDays === 1) {
    return "Yesterday";
  } else if (diffInDays < 7) {
    return `${diffInDays} days ago`;
  } else if (diffInDays < 30) {
    return `${Math.floor(diffInDays / 7)} weeks ago`;
  } else {
    return date.toLocaleDateString();
  }
};

// Format file size
const formatSize = (bytes: number) => {
  if (bytes < 1024) return bytes + " B";
  else if (bytes < 1048576) return (bytes / 1024).toFixed(1) + " KB";
  else if (bytes < 1073741824) return (bytes / 1048576).toFixed(1) + " MB";
  else return (bytes / 1073741824).toFixed(1) + " GB";
};

// Create mock folders
export const mockFolders: Folder[] = [
  {
    id: "folder-1",
    name: "Documents",
    type: "folder",
    path: "/Documents",
    itemCount: 24,
    lastModifiedDate: randomDate(),
    lastModified: "2 days ago",
    owner: "You",
    isShared: false,
  },
  {
    id: "folder-2",
    name: "Images",
    type: "folder",
    path: "/Images",
    itemCount: 156,
    lastModifiedDate: randomDate(),
    lastModified: "Yesterday",
    owner: "You",
    isShared: true,
    sharedWith: ["Mark Wilson", "Sarah Chen"],
  },
  {
    id: "folder-3",
    name: "Projects",
    type: "folder",
    path: "/Projects",
    itemCount: 7,
    lastModifiedDate: randomDate(),
    lastModified: "Today",
    owner: "You",
    isShared: false,
  },
  {
    id: "folder-4",
    name: "Backups",
    type: "folder",
    path: "/Backups",
    itemCount: 3,
    lastModifiedDate: randomDate(),
    lastModified: "1 week ago",
    owner: "System",
    isShared: false,
  },
];

// Create mock files
export const mockFiles: File[] = [
  {
    id: "file-1",
    name: "Quarterly Report.docx",
    type: "file",
    extension: "docx",
    path: "/Quarterly Report.docx",
    sizeInBytes: 2500000,
    size: "2.5 MB",
    lastModifiedDate: randomDate(),
    lastModified: "2 hours ago",
    isRefrigerated: false,
    owner: "You",
    isShared: false,
  },
  {
    id: "file-2",
    name: "Budget 2023.xlsx",
    type: "file",
    extension: "xlsx",
    path: "/Budget 2023.xlsx",
    sizeInBytes: 1800000,
    size: "1.8 MB",
    lastModifiedDate: randomDate(),
    lastModified: "Yesterday",
    isRefrigerated: true,
    compressionRatio: 0.45,
    owner: "You",
    isShared: true,
    sharedWith: ["Finance Team"],
  },
  {
    id: "file-3",
    name: "Product Presentation.pptx",
    type: "file",
    extension: "pptx",
    path: "/Product Presentation.pptx",
    sizeInBytes: 8500000,
    size: "8.5 MB",
    lastModifiedDate: randomDate(),
    lastModified: "3 days ago",
    isRefrigerated: false,
    owner: "Mark Wilson",
    isShared: true,
    sharedWith: ["Marketing Team"],
  },
  {
    id: "file-4",
    name: "Team Photo.jpg",
    type: "file",
    extension: "jpg",
    path: "/Team Photo.jpg",
    sizeInBytes: 4200000,
    size: "4.2 MB",
    lastModifiedDate: randomDate(),
    lastModified: "1 week ago",
    isRefrigerated: true,
    compressionRatio: 0.32,
    owner: "Sarah Chen",
    isShared: true,
    sharedWith: ["Everyone"],
  },
  {
    id: "file-5",
    name: "Project Roadmap.pdf",
    type: "file",
    extension: "pdf",
    path: "/Project Roadmap.pdf",
    sizeInBytes: 5700000,
    size: "5.7 MB",
    lastModifiedDate: randomDate(),
    lastModified: "Today",
    isRefrigerated: false,
    owner: "You",
    isShared: false,
  },
  {
    id: "file-6",
    name: "Code Backup.zip",
    type: "file",
    extension: "zip",
    path: "/Code Backup.zip",
    sizeInBytes: 154000000,
    size: "154 MB",
    lastModifiedDate: randomDate(),
    lastModified: "2 weeks ago",
    isRefrigerated: true,
    compressionRatio: 0.64,
    owner: "System",
    isShared: false,
  },
  {
    id: "file-7",
    name: "Meeting Notes.txt",
    type: "file",
    extension: "txt",
    path: "/Meeting Notes.txt",
    sizeInBytes: 45000,
    size: "45 KB",
    lastModifiedDate: randomDate(),
    lastModified: "4 days ago",
    isRefrigerated: false,
    owner: "You",
    isShared: false,
  },
  {
    id: "file-8",
    name: "Employee Handbook.pdf",
    type: "file",
    extension: "pdf",
    path: "/Employee Handbook.pdf",
    sizeInBytes: 12500000,
    size: "12.5 MB",
    lastModifiedDate: randomDate(),
    lastModified: "3 weeks ago",
    isRefrigerated: true,
    compressionRatio: 0.41,
    owner: "HR Department",
    isShared: true,
    sharedWith: ["All Employees"],
  },
];
