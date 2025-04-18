
import { useState } from "react";
import { FileGrid } from "./FileGrid";
import { FileList } from "./FileList";
import { Breadcrumb } from "./Breadcrumb";
import { FileUpload } from "./FileUpload";
import { Button } from "@/components/ui/button";
import { 
  FolderPlus, 
  Upload, 
  RefreshCw, 
  Filter,
  LayoutGrid,
  List as ListIcon
} from "lucide-react";
import { toast } from "sonner";
import { mockFiles, mockFolders } from "@/data/mock-files";

export function FileBrowser() {
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [currentPath, setCurrentPath] = useState("/");
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  
  const toggleViewMode = () => {
    setViewMode(viewMode === "grid" ? "list" : "grid");
  };
  
  const refreshFiles = () => {
    setIsRefreshing(true);
    setTimeout(() => {
      setIsRefreshing(false);
      toast.success("Files refreshed");
    }, 1000);
  };
  
  const createFolder = () => {
    toast.info("Create folder dialog coming soon");
  };
  
  const showFilters = () => {
    toast.info("Advanced filters coming soon");
  };

  const handleUpload = () => {
    setIsUploading(true);
    setTimeout(() => {
      setIsUploading(false);
      toast.success("File uploaded successfully");
    }, 2000);
  };
  
  const navigateToFolder = (path: string) => {
    setCurrentPath(path);
    toast.info(`Navigated to ${path}`);
  };

  return (
    <div className="h-full flex flex-col">
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <Breadcrumb path={currentPath} onNavigate={navigateToFolder} />
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={refreshFiles}
              disabled={isRefreshing}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? "animate-spin" : ""}`} />
              Refresh
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={showFilters}
            >
              <Filter className="h-4 w-4 mr-2" />
              Filters
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={toggleViewMode}
            >
              {viewMode === "grid" ? (
                <>
                  <ListIcon className="h-4 w-4 mr-2" />
                  List
                </>
              ) : (
                <>
                  <LayoutGrid className="h-4 w-4 mr-2" />
                  Grid
                </>
              )}
            </Button>
          </div>
        </div>
        
        <div className="flex flex-wrap items-center gap-2">
          <Button onClick={createFolder} variant="outline" size="sm">
            <FolderPlus className="h-4 w-4 mr-2" />
            New Folder
          </Button>
          <FileUpload onUpload={handleUpload} isUploading={isUploading} />
        </div>
      </div>
      
      <div className="mt-6 flex-1 overflow-auto">
        {viewMode === "grid" ? (
          <FileGrid 
            files={mockFiles} 
            folders={mockFolders} 
            onFolderClick={navigateToFolder}
          />
        ) : (
          <FileList 
            files={mockFiles} 
            folders={mockFolders} 
            onFolderClick={navigateToFolder}
          />
        )}
      </div>
    </div>
  );
}
