
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
  MoreHorizontal
} from "lucide-react";
import { File as FileType, Folder as FolderType } from "@/types/file-types";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { fileIconMap } from "@/lib/file-icons";
import { 
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface FileGridProps {
  files: FileType[];
  folders: FolderType[];
  onFolderClick: (path: string) => void;
}

export function FileGrid({ files, folders, onFolderClick }: FileGridProps) {
  const [selectedItems, setSelectedItems] = useState<string[]>([]);

  const handleContextAction = (action: string, item: FileType | FolderType) => {
    toast.info(`${action} ${item.name} (coming soon)`);
  };

  const handleItemClick = (item: FileType | FolderType, event: React.MouseEvent) => {
    if (event.ctrlKey || event.metaKey) {
      // Toggle selection
      setSelectedItems(prev => 
        prev.includes(item.id) 
          ? prev.filter(id => id !== item.id)
          : [...prev, item.id]
      );
    } else if (item.type === "folder") {
      onFolderClick(item.path);
    } else {
      toast.info(`Opening ${item.name} (coming soon)`);
    }
  };

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
      {folders.map((folder) => (
        <ContextMenu key={folder.id}>
          <ContextMenuTrigger>
            <div 
              className={`folder-card cursor-pointer ${selectedItems.includes(folder.id) ? 'ring-2 ring-primary' : ''}`}
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
            <ContextMenuItem onClick={() => handleContextAction("Open", folder)}>
              Open
            </ContextMenuItem>
            <ContextMenuSeparator />
            <ContextMenuItem onClick={() => handleContextAction("Rename", folder)}>
              <Edit className="h-4 w-4 mr-2" /> Rename
            </ContextMenuItem>
            <ContextMenuItem onClick={() => handleContextAction("Copy", folder)}>
              <Copy className="h-4 w-4 mr-2" /> Copy
            </ContextMenuItem>
            <ContextMenuItem onClick={() => handleContextAction("Move", folder)}>
              <Move className="h-4 w-4 mr-2" /> Move
            </ContextMenuItem>
            <ContextMenuSeparator />
            <ContextMenuItem onClick={() => handleContextAction("Share", folder)}>
              <Share2 className="h-4 w-4 mr-2" /> Share
            </ContextMenuItem>
            <ContextMenuSeparator />
            <ContextMenuItem 
              className="text-destructive focus:text-destructive" 
              onClick={() => handleContextAction("Delete", folder)}
            >
              <Trash2 className="h-4 w-4 mr-2" /> Delete
            </ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      ))}
      
      {files.map((file) => {
        const FileIcon = fileIconMap[file.extension] || File;
        const isRefrigerated = file.isRefrigerated;
        
        return (
          <ContextMenu key={file.id}>
            <ContextMenuTrigger>
              <div 
                className={`file-card cursor-pointer ${isRefrigerated ? 'refrigerated' : ''} ${selectedItems.includes(file.id) ? 'ring-2 ring-primary' : ''}`}
                onClick={(e) => handleItemClick(file, e)}
              >
                <div className="flex flex-col items-center justify-center h-full relative">
                  <div className="mb-2">
                    <FileIcon className={`h-12 w-12 ${isRefrigerated ? 'text-icebox-700' : 'text-muted-foreground'}`} />
                    {isRefrigerated && (
                      <div className="absolute top-0 right-0">
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
                  </div>
                  <div className="text-center">
                    <p className="text-sm font-medium truncate max-w-full">{file.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {file.size} • {file.lastModified}
                    </p>
                  </div>
                </div>
              </div>
            </ContextMenuTrigger>
            <ContextMenuContent>
              <ContextMenuItem onClick={() => handleContextAction("Open", file)}>
                Open
              </ContextMenuItem>
              <ContextMenuItem onClick={() => handleContextAction("Download", file)}>
                <Download className="h-4 w-4 mr-2" /> Download
              </ContextMenuItem>
              <ContextMenuSeparator />
              <ContextMenuItem onClick={() => handleContextAction("Rename", file)}>
                <Edit className="h-4 w-4 mr-2" /> Rename
              </ContextMenuItem>
              <ContextMenuItem onClick={() => handleContextAction("Copy", file)}>
                <Copy className="h-4 w-4 mr-2" /> Copy
              </ContextMenuItem>
              <ContextMenuItem onClick={() => handleContextAction("Move", file)}>
                <Move className="h-4 w-4 mr-2" /> Move
              </ContextMenuItem>
              <ContextMenuSeparator />
              <ContextMenuItem onClick={() => handleContextAction("Share", file)}>
                <Share2 className="h-4 w-4 mr-2" /> Share
              </ContextMenuItem>
              {isRefrigerated ? (
                <ContextMenuItem onClick={() => handleContextAction("Unrefrigerate", file)}>
                  <Snowflake className="h-4 w-4 mr-2" /> Unrefrigerate
                </ContextMenuItem>
              ) : (
                <ContextMenuItem onClick={() => handleContextAction("Refrigerate", file)}>
                  <Snowflake className="h-4 w-4 mr-2" /> Refrigerate
                </ContextMenuItem>
              )}
              <ContextMenuSeparator />
              <ContextMenuItem 
                className="text-destructive focus:text-destructive" 
                onClick={() => handleContextAction("Delete", file)}
              >
                <Trash2 className="h-4 w-4 mr-2" /> Delete
              </ContextMenuItem>
            </ContextMenuContent>
          </ContextMenu>
        );
      })}
    </div>
  );
}
