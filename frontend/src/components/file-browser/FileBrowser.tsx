import { useState, useEffect } from "react";
import { FileGrid } from "./FileGrid";
import { FileList } from "./FileList";
import { Breadcrumb } from "./Breadcrumb";
import { FileUpload } from "./FileUpload";
import { Button } from "@/components/ui/button";
import { FolderPlus, Upload, RefreshCw, Filter, LayoutGrid, List as ListIcon } from "lucide-react";
import { toast } from "sonner";
import { mockFiles, mockFolders } from "@/data/mock-files";
import { File, Folder } from "@/types/file-types";
import { FileBrowserAPI } from "@/lib/file-api";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
// export function FileBrowser() {
//   const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
//   const [currentPath, setCurrentPath] = useState("/");
//   const [isRefreshing, setIsRefreshing] = useState(false);
//   const [isUploading, setIsUploading] = useState(false);

//   const toggleViewMode = () => {
//     setViewMode(viewMode === "grid" ? "list" : "grid");
//   };

//   const refreshFiles = () => {
//     setIsRefreshing(true);
//     setTimeout(() => {
//       setIsRefreshing(false);
//       toast.success("Files refreshed");
//     }, 1000);
//   };

//   const createFolder = () => {
//     toast.info("Create folder dialog coming soon");
//   };

//   const showFilters = () => {
//     toast.info("Advanced filters coming soon");
//   };

//   const handleUpload = () => {
//     setIsUploading(true);
//     setTimeout(() => {
//       setIsUploading(false);
//       toast.success("File uploaded successfully");
//     }, 2000);
//   };

//   const navigateToFolder = (path: string) => {
//     setCurrentPath(path);
//     toast.info(`Navigated to ${path}`);
//   };

//   return (
//     <div className="h-full flex flex-col">
//       <div className="flex flex-col gap-4">
//         <div className="flex items-center justify-between">
//           <Breadcrumb path={currentPath} onNavigate={navigateToFolder} />
//           <div className="flex items-center gap-2">
//             <Button
//               variant="outline"
//               size="sm"
//               onClick={refreshFiles}
//               disabled={isRefreshing}
//             >
//               <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? "animate-spin" : ""}`} />
//               Refresh
//             </Button>
//             <Button
//               variant="outline"
//               size="sm"
//               onClick={showFilters}
//             >
//               <Filter className="h-4 w-4 mr-2" />
//               Filters
//             </Button>
//             <Button
//               variant="outline"
//               size="sm"
//               onClick={toggleViewMode}
//             >
//               {viewMode === "grid" ? (
//                 <>
//                   <ListIcon className="h-4 w-4 mr-2" />
//                   List
//                 </>
//               ) : (
//                 <>
//                   <LayoutGrid className="h-4 w-4 mr-2" />
//                   Grid
//                 </>
//               )}
//             </Button>
//           </div>
//         </div>

//         <div className="flex flex-wrap items-center gap-2">
//           <Button onClick={createFolder} variant="outline" size="sm">
//             <FolderPlus className="h-4 w-4 mr-2" />
//             New Folder
//           </Button>
//           <FileUpload onUpload={handleUpload} isUploading={isUploading} />
//         </div>
//       </div>

//       <div className="mt-6 flex-1 overflow-auto">
//         {viewMode === "grid" ? (
//           <FileGrid
//             files={mockFiles}
//             folders={mockFolders}
//             onFolderClick={navigateToFolder}
//           />
//         ) : (
//           <FileList
//             files={mockFiles}
//             folders={mockFolders}
//             onFolderClick={navigateToFolder}
//           />
//         )}
//       </div>
//     </div>
//   );
// }

export function FileBrowser() {
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [currentPath, setCurrentPath] = useState("/");
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [files, setFiles] = useState<File[]>([]);
  const [folders, setFolders] = useState<Folder[]>([]);
  const [newFolderName, setNewFolderName] = useState("");
  const [newFolderDialogOpen, setNewFolderDialogOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch directory contents when path changes or refresh is triggered
  useEffect(() => {
    loadDirectoryContents();
  }, [currentPath]);

  const loadDirectoryContents = async () => {
    try {
      setIsLoading(true);
      setError(null);

      const result = await FileBrowserAPI.listDirectory(currentPath);
      setFiles(result.files);
      setFolders(result.folders);

      setIsLoading(false);
    } catch (err) {
      console.error("Error loading directory contents:", err);
      setError("Failed to load directory contents. Please try again.");
      setIsLoading(false);
    }
  };

  const toggleViewMode = () => {
    setViewMode(viewMode === "grid" ? "list" : "grid");
  };

  const refreshFiles = async () => {
    setIsRefreshing(true);
    try {
      await loadDirectoryContents();
      toast.success("Files refreshed");
    } catch (err) {
      toast.error("Failed to refresh files");
    } finally {
      setIsRefreshing(false);
    }
  };

  const createFolder = async () => {
    if (!newFolderName.trim()) {
      toast.error("Folder name cannot be empty");
      return;
    }

    try {
      await FileBrowserAPI.createFolder(currentPath, newFolderName);
      setNewFolderName("");
      setNewFolderDialogOpen(false);
      toast.success(`Folder "${newFolderName}" created`);
      await loadDirectoryContents();
    } catch (err) {
      toast.error(`Failed to create folder: ${err}`);
    }
  };

  const showFilters = () => {
    toast.info("Advanced filters coming soon");
  };

  const handleUpload = async (file: File) => {
    setIsUploading(true);
    try {
      await FileBrowserAPI.uploadFile(currentPath, file);
      toast.success(`${file.name} uploaded successfully`);
      await loadDirectoryContents();
    } catch (err) {
      toast.error(`Failed to upload file: ${err}`);
    } finally {
      setIsUploading(false);
    }
  };

  const navigateToFolder = (path: string) => {
    setCurrentPath(path);
  };

  const handleContextAction = async (action: string, item: File | Folder) => {
    try {
      switch (action) {
        case "Delete":
          await FileBrowserAPI.deleteItem(item.path);
          toast.success(`${item.name} deleted`);
          await loadDirectoryContents();
          break;
        case "Rename":
          // This would typically open a dialog to get the new name
          const newName = prompt("Enter new name:", item.name);
          if (newName && newName !== item.name) {
            await FileBrowserAPI.renameItem(item.path, newName);
            toast.success(`Renamed ${item.name} to ${newName}`);
            await loadDirectoryContents();
          }
          break;
        case "Refrigerate":
          if (item.type === "file") {
            await FileBrowserAPI.refrigerateFile(item.path);
            toast.success(`${item.name} refrigerated`);
            await loadDirectoryContents();
          }
          break;
        case "Unrefrigerate":
          if (item.type === "file") {
            await FileBrowserAPI.unrefrigerateFile(item.path);
            toast.success(`${item.name} unrefrigerated`);
            await loadDirectoryContents();
          }
          break;
        default:
          toast.info(`${action} ${item.name} (coming soon)`);
      }
    } catch (err) {
      toast.error(`Failed to ${action.toLowerCase()}: ${err}`);
    }
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
              disabled={isRefreshing || isLoading}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? "animate-spin" : ""}`} />
              Refresh
            </Button>
            <Button variant="outline" size="sm" onClick={showFilters}>
              <Filter className="h-4 w-4 mr-2" />
              Filters
            </Button>
            <Button variant="outline" size="sm" onClick={toggleViewMode}>
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
          <Dialog open={newFolderDialogOpen} onOpenChange={setNewFolderDialogOpen}>
            <DialogTrigger asChild>
              <Button onClick={() => setNewFolderDialogOpen(true)} variant="outline" size="sm">
                <FolderPlus className="h-4 w-4 mr-2" />
                New Folder
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[425px]">
              <DialogHeader>
                <DialogTitle>Create New Folder</DialogTitle>
                <DialogDescription>
                  Enter a name for the new folder at {currentPath}
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid grid-cols-4 items-center gap-4">
                  <Label htmlFor="name" className="text-right">
                    Name
                  </Label>
                  <Input
                    id="name"
                    value={newFolderName}
                    onChange={(e) => setNewFolderName(e.target.value)}
                    className="col-span-3"
                    placeholder="New Folder"
                    autoFocus
                  />
                </div>
              </div>
              <DialogFooter>
                <Button type="submit" onClick={createFolder}>
                  Create Folder
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          <FileUpload onUpload={handleUpload} isUploading={isUploading} />
        </div>
      </div>

      <div className="mt-6 flex-1 overflow-auto">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            <span className="ml-2">Loading files...</span>
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-full text-destructive">
            <p>{error}</p>
            <Button variant="outline" className="mt-4" onClick={refreshFiles}>
              Try Again
            </Button>
          </div>
        ) : viewMode === "grid" ? (
          <FileGrid
            files={files}
            folders={folders}
            onFolderClick={navigateToFolder}
            onContextAction={handleContextAction}
          />
        ) : (
          <FileList
            files={files}
            folders={folders}
            onFolderClick={navigateToFolder}
            onContextAction={handleContextAction}
          />
        )}
      </div>
    </div>
  );
}
