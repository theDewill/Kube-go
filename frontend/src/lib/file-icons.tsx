
import { 
  FileText, 
  FileImage, 
  FileAudio, 
  FileVideo, 
  FileCode, 
  FileSpreadsheet, 
  FilePieChart, 
  FileArchive,
  File
} from "lucide-react";

export const fileIconMap: Record<string, React.ElementType> = {
  // Documents
  'pdf': FileText,
  'doc': FileText,
  'docx': FileText,
  'txt': FileText,
  'rtf': FileText,
  'md': FileText,
  
  // Spreadsheets
  'xls': FileSpreadsheet,
  'xlsx': FileSpreadsheet,
  'csv': FileSpreadsheet,
  
  // Presentations
  'ppt': FilePieChart,
  'pptx': FilePieChart,
  
  // Images
  'jpg': FileImage,
  'jpeg': FileImage,
  'png': FileImage,
  'gif': FileImage,
  'svg': FileImage,
  'webp': FileImage,
  'bmp': FileImage,
  
  // Audio
  'mp3': FileAudio,
  'wav': FileAudio,
  'ogg': FileAudio,
  'flac': FileAudio,
  
  // Video
  'mp4': FileVideo,
  'mov': FileVideo,
  'avi': FileVideo,
  'webm': FileVideo,
  'mkv': FileVideo,
  
  // Code
  'js': FileCode,
  'ts': FileCode,
  'jsx': FileCode,
  'tsx': FileCode,
  'html': FileCode,
  'css': FileCode,
  'json': FileCode,
  'py': FileCode,
  'java': FileCode,
  'c': FileCode,
  'cpp': FileCode,
  'rs': FileCode,
  'go': FileCode,
  
  // Archives
  'zip': FileArchive,
  'rar': FileArchive,
  '7z': FileArchive,
  'tar': FileArchive,
  'gz': FileArchive,
  
  // Default
  'default': File
};
