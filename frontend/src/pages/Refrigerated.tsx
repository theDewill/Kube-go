import { useState } from "react";
import { AppLayout } from "@/components/layout/AppLayout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Upload, Hammer } from "lucide-react";
import { toast } from "sonner";
import { mockFiles } from "@/data/mock-files";

const Refrigerated = () => {
  const [isCompressing, setIsCompressing] = useState(false);
  const [compressedFiles, setCompressedFiles] = useState(
    mockFiles.filter((file) => file.isRefrigerated),
  );
  const [dragActive, setDragActive] = useState(false);

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
      handleFileUpload(e.dataTransfer.files);
    }
  };

  const handleFileUpload = (files: FileList) => {
    toast.info(`${files.length} file(s) added to compression bay`);
  };

  const compressFiles = () => {
    setIsCompressing(true);
    toast.info("Starting refrigeration process");

    // Simulate compression process
    setTimeout(() => {
      toast.success("Files have been refrigerated successfully");
      setIsCompressing(false);

      // Add a mock compressed file
      const newCompressedFile = {
        ...mockFiles[0],
        id: `compressed-${Date.now()}`,
        name: `Compressed-${Date.now()}.zip`,
        isRefrigerated: true,
        compressionRatio: Math.floor(Math.random() * 80) + 10, // Random compression ratio between 10-90%
      };

      setCompressedFiles((prev) => [newCompressedFile, ...prev]);
    }, 2000);
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
                    input.multiple = true;
                    input.onchange = (e) => {
                      const target = e.target as HTMLInputElement;
                      if (target.files) handleFileUpload(target.files);
                    };
                    input.click();
                  }}
                >
                  Select Files
                </Button>
              </div>

              <div className="flex justify-center mt-4">
                <Button
                  className="relative overflow-hidden group bg-red-400 hover:bg-red-500"
                  onClick={compressFiles}
                  disabled={isCompressing}
                  variant="default"
                >
                  <span className="flex items-center">
                    <Hammer className="h-5 w-5 mr-2 text-red-900 group-hover:animate-pulse" />
                    {isCompressing ? "Refrigerating..." : "Refrigerate"}
                  </span>
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Compressed Files Index */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Hammer className="mr-2 h-5 w-5 text-red-500" />
                Recent Refrigerations
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="overflow-auto max-h-80">
                {compressedFiles.length > 0 ? (
                  <div className="space-y-2">
                    {compressedFiles.map((file) => (
                      <div
                        key={file.id}
                        className="flex items-center justify-between p-3 bg-sidebar/5 rounded-md hover:bg-sidebar/10"
                      >
                        <div className="flex items-center space-x-3">
                          <div className="p-2 bg-primary/10 rounded-md">
                            <Hammer className="h-4 w-4 text-red-500" />
                          </div>
                          <div>
                            <p className="font-medium text-sm">{file.name}</p>
                            <p className="text-xs text-muted-foreground">{file.size}</p>
                          </div>
                        </div>
                        <Button variant="ghost" size="sm">
                          Restore
                        </Button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center h-64 text-center text-muted-foreground">
                    <Hammer className="h-10 w-10 mb-4" />
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
