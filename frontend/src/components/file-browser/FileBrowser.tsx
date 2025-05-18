import { useState, useEffect } from "react";
import { FileGrid } from "./FileGrid";
import { FileList } from "./FileList";
import { Breadcrumb } from "./Breadcrumb";
import { Button } from "@/components/ui/button";
import {
  FolderPlus,
  Upload,
  RefreshCw,
  Filter,
  LayoutGrid,
  List as ListIcon,
  CloudUpload,
  Cloud,
} from "lucide-react";
import { toast } from "sonner";
import { FileBrowserAPI } from "@/lib/file-api";
import { File, Folder } from "@/types/file-types";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";

export function FileUpload({ onUpload, isUploading }) {
  const [selectedFile, setSelectedFile] = useState(null);
  const [distribute, setDistribute] = useState(false);
  const [modelType, setModelType] = useState("ollama");
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);

  const handleFileChange = (e) => {
    if (e.target.files && e.target.files[0]) {
      setSelectedFile(e.target.files[0]);
    }
  };

  const handleUpload = async () => {
    if (!selectedFile) {
      toast.error("Please select a file to upload");
      return;
    }

    try {
      await onUpload(selectedFile, distribute, modelType);
      setSelectedFile(null);
      setUploadDialogOpen(false);
      setDistribute(false);
      setModelType("ollama");
    } catch (error) {
      toast.error(`Upload failed: ${error.message}`);
    }
  };

  return (
    <Dialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" onClick={() => setUploadDialogOpen(true)}>
          <Upload className="h-4 w-4 mr-2" />
          Upload File
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Upload File</DialogTitle>
          <DialogDescription>Choose a file to upload to the current directory.</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="file">File</Label>
            <Input id="file" type="file" onChange={handleFileChange} disabled={isUploading} />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="model-type">AI Model for Content Analysis</Label>
            <Select value={modelType} onValueChange={setModelType} disabled={isUploading}>
              <SelectTrigger id="model-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ollama">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                    <span>Ollama (Local)</span>
                  </div>
                </SelectItem>
                <SelectItem value="gemini">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                    <span>Google Gemini (Cloud)</span>
                  </div>
                </SelectItem>
                <SelectItem value="simple">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-gray-500 rounded-full"></div>
                    <span>Simple Analysis (No AI)</span>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
            <div className="text-xs text-muted-foreground">
              {modelType === "ollama" && "Uses your local Ollama installation for content analysis"}
              {modelType === "gemini" &&
                "Uses Google Gemini API for advanced content analysis (requires API key)"}
              {modelType === "simple" && "Basic keyword extraction without AI (fastest option)"}
            </div>
          </div>

          <div className="flex items-center space-x-2">
            <Switch
              id="distribute"
              checked={distribute}
              onCheckedChange={setDistribute}
              disabled={isUploading}
            />
            <Label htmlFor="distribute" className="flex items-center cursor-pointer">
              <Cloud className="h-4 w-4 mr-2 text-primary" />
              Distribute across network nodes
              <Badge variant="outline" className="ml-2 text-xs">
                Beta
              </Badge>
            </Label>
          </div>
          <div className="text-xs text-muted-foreground">
            {distribute
              ? "The file will be chunked, encrypted, and distributed across available nodes in the network. This improves redundancy and availability."
              : "The file will be stored locally on this device only."}
          </div>
        </div>
        <DialogFooter>
          <Button type="submit" onClick={handleUpload} disabled={!selectedFile || isUploading}>
            {isUploading ? (
              <>
                <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                Uploading...
              </>
            ) : distribute ? (
              <>
                <CloudUpload className="h-4 w-4 mr-2" />
                Distribute
              </>
            ) : (
              <>
                <Upload className="h-4 w-4 mr-2" />
                Upload
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

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
  const [networkNodes, setNetworkNodes] = useState([]);
  const [error, setError] = useState<string | null>(null);

  // Fetch network nodes
  useEffect(() => {
    const fetchNetworkNodes = async () => {
      try {
        const nodes = await FileBrowserAPI.getNetworkNodes();
        setNetworkNodes(nodes);
      } catch (err) {
        console.error("Failed to fetch network nodes:", err);
      }
    };

    fetchNetworkNodes();
    const interval = setInterval(fetchNetworkNodes, 30000); // Refresh every 30 seconds

    return () => clearInterval(interval);
  }, []);

  // Fetch directory contents
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

  const handleUpload = async (file: File, distribute: boolean, modelType: string) => {
    setIsUploading(true);
    try {
      await FileBrowserAPI.uploadFile(currentPath, file, distribute, modelType);
      toast.success(`${file.name} ${distribute ? "distributed" : "uploaded"} successfully`);
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
        case "Download":
          if (item.type === "file") {
            const blob = await FileBrowserAPI.downloadFile(item.path);
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = item.name;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
            toast.success(`Downloaded ${item.name}`);
          }
          break;
        default:
          toast.info(`${action} ${item.name} (coming soon)`);
      }
    } catch (err) {
      toast.error(`Failed to ${action.toLowerCase()}: ${err}`);
    }
  };

  // Count available online nodes
  const onlineNodesCount = networkNodes.filter(
    (node) => node.status === "online" || node.status === "locked",
  ).length;

  return (
    <div className="h-full flex flex-col">
      <div className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <Breadcrumb path={currentPath} onNavigate={navigateToFolder} />
          <div className="flex items-center gap-2">
            <Badge variant={onlineNodesCount > 1 ? "success" : "secondary"} className="mr-2">
              <Cloud className="h-3 w-3 mr-1" />
              {onlineNodesCount} {onlineNodesCount === 1 ? "node" : "nodes"} online
            </Badge>

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
