import { useState, useEffect } from "react";
import { AppLayout } from "@/components/layout/AppLayout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Upload, Hammer, FileText, RefreshCw, AlertCircle, ArrowDownToLine } from "lucide-react";
import { toast } from "sonner";
import { FileBrowserAPI } from "@/lib/file-api";
import { File as FileType } from "@/types/file-types";
import { DecompressItem } from "@/../wailsjs/go/kfiles/FileBrowser";

const Refrigerated = () => {
  // State for compressed files directory
  const [compressedFiles, setCompressedFiles] = useState<FileType[]>([]);
  const [isLoadingCompressed, setIsLoadingCompressed] = useState(true);
  const [compressedError, setCompressedError] = useState<string | null>(null);

  // State for file selection and compression
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [selectedFileInfo, setSelectedFileInfo] = useState<{ name: string; size: string } | null>(
    null,
  );
  const [isCompressing, setIsCompressing] = useState(false);
  const [dragActive, setDragActive] = useState(false);
  const [isRestoring, setIsRestoring] = useState<string | null>(null);

  // Load compressed files from the "kubecompress" directory
  const loadCompressedFiles = async () => {
    try {
      setIsLoadingCompressed(true);
      setCompressedError(null);

      const result = await FileBrowserAPI.listDirectoryRFG("/kubecompress");
      setCompressedFiles(result.files);
      setIsLoadingCompressed(false);
    } catch (err) {
      console.error("Error loading compressed files:", err);
      setCompressedError("Failed to load refrigerated files");
      setIsLoadingCompressed(false);
    }
  };

  // Load compressed files on component mount
  useEffect(() => {
    loadCompressedFiles();
  }, []);

  // Handle drag events
  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    // Handle the file drop
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFileSelection(e.dataTransfer.files);
    }
  };

  // Helper function to format file size
  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  // Handle file selection (either from drop or file picker)
  const handleFileSelection = (files: FileList) => {
    if (files.length > 0) {
      const file = files[0];
      setSelectedFile(file);
      setSelectedFileInfo({
        name: file.name,
        size: formatFileSize(file.size),
      });
      toast.info(`${file.name} selected for refrigeration`);
    }
  };

  // Handle compression of selected file - implemented like FileBrowser's handleUpload
  const compressFile = async () => {
    if (!selectedFile) {
      toast.error("Please select a file to refrigerate");
      return;
    }

    setIsCompressing(true);
    toast.info(`Starting refrigeration of ${selectedFile.name}`);

    try {
      // Step 1: Upload file to temp directory - EXACTLY like in FileBrowser
      const tempPath = "kubecompressqueue"; // Temporary directory for files to be compressed

      // Convert file to byte array
      const buffer = await selectedFile.arrayBuffer();
      const bytes = new Uint8Array(buffer);
      const byteArray = Array.from(bytes);

      // Upload the file to the temporary directory
      await FileBrowserAPI.uploadFileRFG(tempPath, selectedFile, false, "simple");

      // Step 2: Now compress the file using the path
      const filePath = `${tempPath}/${selectedFile.name}`;
      await FileBrowserAPI.compressItem(filePath, selectedFile.name);

      toast.success(`${selectedFile.name} refrigerated successfully`);

      // Step 3: Clean up - delete the temp file and refresh the list
      await FileBrowserAPI.deleteItem(filePath);

      // Clear selection and refresh compressed files list
      setSelectedFile(null);
      setSelectedFileInfo(null);
      await loadCompressedFiles();
    } catch (error: any) {
      toast.error(`Refrigeration failed: ${error}`);
      console.error("Refrigeration error:", error);
    } finally {
      setIsCompressing(false);
    }
  };

  // Handle restoration (decompression) of a file
  const restoreFile = async (file: FileType) => {
    setIsRestoring(file.id);
    toast.info(`Restoring ${file.name}...`);

    try {
      // Use the file path directly for decompression
      //await DecompressItem(file.path);
      await FileBrowserAPI.decompressItem(file.path);
      toast.success(`${file.name} restored successfully`);

      // Refresh compressed files list
      await loadCompressedFiles();
    } catch (error: any) {
      toast.error(`Restoration failed: ${error}`);
    } finally {
      setIsRestoring(null);
    }
  };

  return (
    <AppLayout>
      <div className="flex flex-col gap-4">
        <h1 className="text-2xl font-bold">Refrigerator Utility</h1>
        <p className="text-muted-foreground">
          Compress your files for long-term cold storage with advanced kube refrigeration
          technology.
        </p>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-4">
          {/* Compression Bay */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Upload className="mr-2 h-5 w-5 text-icebox-600" />
                Compression Bay
              </CardTitle>
            </CardHeader>
            <CardContent>
              {selectedFile && selectedFileInfo ? (
                <div className="flex flex-col">
                  <div className="bg-sidebar/5 p-4 rounded-lg">
                    <div className="flex items-center mb-4">
                      <FileText className="h-8 w-8 mr-3 text-primary" />
                      <div>
                        <p className="font-medium">{selectedFileInfo.name}</p>
                        <p className="text-xs text-muted-foreground">{selectedFileInfo.size}</p>
                      </div>
                    </div>

                    <div className="flex space-x-2">
                      <Button
                        className="relative flex-1 overflow-hidden group bg-orange-400 hover:bg-orange-500"
                        onClick={compressFile}
                        disabled={isCompressing}
                        variant="default"
                      >
                        <span className="flex items-center">
                          <Hammer className="h-5 w-5 mr-2 text-black group-hover:animate-pulse" />
                          {isCompressing ? "Refrigerating..." : "Refrigerate"}
                        </span>
                      </Button>
                      <Button
                        variant="outline"
                        onClick={() => {
                          setSelectedFile(null);
                          setSelectedFileInfo(null);
                        }}
                        disabled={isCompressing}
                      >
                        Cancel
                      </Button>
                    </div>
                  </div>

                  {isCompressing && (
                    <div className="mt-4 text-center">
                      <RefreshCw className="h-8 w-8 animate-spin mx-auto mb-2 text-icebox-600" />
                      <p className="text-sm text-muted-foreground">Refrigeration in progress...</p>
                    </div>
                  )}
                </div>
              ) : (
                <div
                  className={`flex flex-col items-center justify-center border-2 border-dashed rounded-lg p-8 h-64 transition-colors
                    ${dragActive ? "border-primary bg-primary/5" : "border-border"}
                    hover:border-primary hover:bg-primary/5`}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                >
                  <Upload className="h-10 w-10 text-muted-foreground mb-4" />
                  <p className="text-center text-muted-foreground mb-2">
                    Drag files here or click to select files for refrigeration
                  </p>
                  <Button
                    variant="outline"
                    onClick={() => {
                      const input = document.createElement("input");
                      input.type = "file";
                      input.onchange = (e) => {
                        const target = e.target as HTMLInputElement;
                        if (target.files) handleFileSelection(target.files);
                      };
                      input.click();
                    }}
                  >
                    Select Files
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Compressed Files Index */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="flex items-center">
                <Hammer className="mr-2 h-5 w-5 text-icebox-600" />
                Refrigerated Files
              </CardTitle>

              <Button
                variant="ghost"
                size="sm"
                onClick={loadCompressedFiles}
                disabled={isLoadingCompressed}
              >
                <RefreshCw className={`h-4 w-4 ${isLoadingCompressed ? "animate-spin" : ""}`} />
                <span className="sr-only">Refresh</span>
              </Button>
            </CardHeader>
            <CardContent>
              <div className="overflow-auto max-h-80">
                {isLoadingCompressed ? (
                  <div className="flex flex-col items-center justify-center h-40">
                    <RefreshCw className="h-8 w-8 animate-spin mb-2" />
                    <p className="text-sm text-muted-foreground">Loading refrigerated files...</p>
                  </div>
                ) : compressedError ? (
                  <div className="flex flex-col items-center justify-center h-40 text-destructive">
                    <AlertCircle className="h-8 w-8 mb-2" />
                    <p>{compressedError}</p>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={loadCompressedFiles}
                      className="mt-2"
                    >
                      Retry
                    </Button>
                  </div>
                ) : compressedFiles.length > 0 ? (
                  <div className="space-y-2">
                    {compressedFiles.map((file) => (
                      <div
                        key={file.id}
                        className="flex items-center justify-between p-3 bg-sidebar/5 rounded-md hover:bg-sidebar/10"
                      >
                        <div className="flex items-center space-x-3">
                          <div className="p-2 bg-icebox-300 dark:bg-icebox-900 rounded-md">
                            <Hammer className="h-4 w-4 dark:text-white text-icebox-60" />
                          </div>
                          <div>
                            <p className="font-medium text-sm">{file.name}</p>
                            <p className="text-xs text-muted-foreground">{file.size}</p>
                          </div>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => restoreFile(file)}
                          disabled={isRestoring === file.id}
                        >
                          {isRestoring === file.id ? (
                            <RefreshCw className="h-4 w-4 animate-spin mr-2" />
                          ) : (
                            <ArrowDownToLine className="h-4 w-4 mr-2" />
                          )}
                          {isRestoring === file.id ? "Restoring..." : "Restore"}
                        </Button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center h-64 text-center text-muted-foreground">
                    <Hammer className="h-10 w-10 mb-4 text-icebox-400" />
                    <p>No refrigerated files yet</p>
                    <p className="text-sm">Use the compression bay to refrigerate your files</p>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </AppLayout>
  );
};

export default Refrigerated;
