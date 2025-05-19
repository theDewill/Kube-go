import { useState } from "react";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
  Folder,
  File,
  Download,
  Trash2,
  Edit,
  Copy,
  Move,
  Share2,
  Snowflake,
  Cloud,
  CloudOff,
} from "lucide-react";
import { File as FileType, Folder as FolderType } from "@/types/file-types";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { fileIconMap } from "@/lib/file-icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

interface FileGridProps {
  files: FileType[];
  folders: FolderType[];
  onFolderClick: (path: string) => void;
  onContextAction: (action: string, item: FileType | FolderType) => void;
}

export function FileGrid({ files, folders, onFolderClick, onContextAction }: FileGridProps) {
  const [selectedItems, setSelectedItems] = useState<string[]>([]);

  const handleItemClick = (item: FileType | FolderType, event: React.MouseEvent) => {
    if (event.ctrlKey || event.metaKey) {
      // Toggle selection
      setSelectedItems((prev) =>
        prev.includes(item.id) ? prev.filter((id) => id !== item.id) : [...prev, item.id],
      );
    } else if (item.type === "folder") {
      onFolderClick(item.path);
    } else {
      toast.info(`Opening ${item.name} (coming soon)`);
    }
  };

  return (
    <div className="w-full h-full">
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 w-full h-fit">
        {folders.map((folder) => (
          <ContextMenu key={folder.id}>
            <ContextMenuTrigger>
              <div
                className={`folder-card cursor-pointer ${selectedItems.includes(folder.id) ? "ring-2 ring-primary" : ""}`}
                onClick={(e) => handleItemClick(folder, e)}
              >
                <div className="flex flex-col items-center justify-center h-full">
                  <div className="mb-2">
                    <Folder className="h-12 w-12 text-icebox-600" />
                  </div>
                  <div className="text-center">
                    <p className="text-sm font-medium truncate max-w-full">{folder.name}</p>
                    <p className="text-xs text-muted-foreground">{folder.itemCount} items</p>
                  </div>
                </div>
              </div>
            </ContextMenuTrigger>
            <ContextMenuContent>
              <ContextMenuItem onClick={() => onContextAction("Open", folder)}>
                Open
              </ContextMenuItem>
              <ContextMenuSeparator />
              <ContextMenuItem onClick={() => onContextAction("Rename", folder)}>
                <Edit className="h-4 w-4 mr-2" /> Rename
              </ContextMenuItem>
              <ContextMenuItem onClick={() => onContextAction("Copy", folder)}>
                <Copy className="h-4 w-4 mr-2" /> Copy
              </ContextMenuItem>
              <ContextMenuItem onClick={() => onContextAction("Move", folder)}>
                <Move className="h-4 w-4 mr-2" /> Move
              </ContextMenuItem>
              <ContextMenuSeparator />
              <ContextMenuItem onClick={() => onContextAction("Share", folder)}>
                <Share2 className="h-4 w-4 mr-2" /> Share
              </ContextMenuItem>
              <ContextMenuSeparator />
              <ContextMenuItem
                className="text-destructive focus:text-destructive"
                onClick={() => onContextAction("Delete", folder)}
              >
                <Trash2 className="h-4 w-4 mr-2" /> Delete
              </ContextMenuItem>
            </ContextMenuContent>
          </ContextMenu>
        ))}

        {files.map((file) => {
          const FileIcon = fileIconMap[file.extension] || File;
          const isRefrigerated = file.isRefrigerated;
          const isDistributed = file.isDistributed;

          return (
            <ContextMenu key={file.id}>
              <ContextMenuTrigger>
                <div
                  className={`file-card cursor-pointer ${isRefrigerated ? "refrigerated" : ""} ${selectedItems.includes(file.id) ? "ring-2 ring-primary" : ""}`}
                  onClick={(e) => handleItemClick(file, e)}
                >
                  <div className="flex flex-col items-center justify-center h-full relative">
                    <div className="mb-2 relative">
                      <FileIcon
                        className={`h-12 w-12 ${isRefrigerated ? "text-icebox-700" : "text-muted-foreground"}`}
                      />

                      {/* Refrigerated indicator */}
                      {isRefrigerated && (
                        <div className="absolute -top-1 -right-1">
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <div className="bg-icebox-100 dark:bg-icebox-900 p-0.5 rounded-full">
                                  <Snowflake className="h-4 w-4 text-icebox-600" />
                                </div>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>Refrigerated (compressed)</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </div>
                      )}

                      {/* Distributed indicator */}
                      {isDistributed && (
                        <div className="absolute -bottom-1 -right-1">
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <div className="bg-primary/10 dark:bg-primary/20 p-0.5 rounded-full">
                                  <Cloud className="h-4 w-4 text-primary" />
                                </div>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>Distributed across network</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </div>
                      )}
                    </div>
                    <div className="text-center">
                      <p className="text-sm font-medium truncate max-w-full">{file.name}</p>
                      <div className="flex items-center justify-center gap-1 text-xs text-muted-foreground">
                        <span>{file.size}</span>
                        <span>•</span>
                        <span>{file.lastModified}</span>
                        {isDistributed && (
                          <>
                            <span>•</span>
                            <Cloud className="h-3 w-3" />
                          </>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </ContextMenuTrigger>
              <ContextMenuContent>
                <ContextMenuItem onClick={() => onContextAction("Open", file)}>
                  Open
                </ContextMenuItem>
                <ContextMenuItem onClick={() => onContextAction("Download", file)}>
                  <Download className="h-4 w-4 mr-2" /> Download
                </ContextMenuItem>
                <ContextMenuSeparator />
                <ContextMenuItem onClick={() => onContextAction("Rename", file)}>
                  <Edit className="h-4 w-4 mr-2" /> Rename
                </ContextMenuItem>
                <ContextMenuItem onClick={() => onContextAction("Copy", file)}>
                  <Copy className="h-4 w-4 mr-2" /> Copy
                </ContextMenuItem>
                <ContextMenuItem onClick={() => onContextAction("Move", file)}>
                  <Move className="h-4 w-4 mr-2" /> Move
                </ContextMenuItem>
                <ContextMenuSeparator />
                <ContextMenuItem onClick={() => onContextAction("Share", file)}>
                  <Share2 className="h-4 w-4 mr-2" /> Share
                </ContextMenuItem>
                {isRefrigerated ? (
                  <ContextMenuItem onClick={() => onContextAction("Unrefrigerate", file)}>
                    <Snowflake className="h-4 w-4 mr-2" /> Unrefrigerate
                  </ContextMenuItem>
                ) : (
                  <ContextMenuItem onClick={() => onContextAction("Refrigerate", file)}>
                    <Snowflake className="h-4 w-4 mr-2" /> Refrigerate
                  </ContextMenuItem>
                )}
                <ContextMenuSeparator />
                <ContextMenuItem
                  className="text-destructive focus:text-destructive"
                  onClick={() => onContextAction("Delete", file)}
                >
                  <Trash2 className="h-4 w-4 mr-2" /> Delete
                </ContextMenuItem>
              </ContextMenuContent>
            </ContextMenu>
          );
        })}
      </div>
    </div>
  );
}
