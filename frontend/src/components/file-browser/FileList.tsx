import { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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
  ChevronDown,
  ChevronUp,
  Cloud,
} from "lucide-react";
import { File as FileType, Folder as FolderType } from "@/types/file-types";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "sonner";
import { fileIconMap } from "@/lib/file-icons";
import { format } from "date-fns";
import { Badge } from "@/components/ui/badge";

interface FileListProps {
  files: FileType[];
  folders: FolderType[];
  onFolderClick: (path: string) => void;
  onContextAction: (action: string, item: FileType | FolderType) => void;
}

type SortField = "name" | "size" | "lastModified" | "type";
type SortDirection = "asc" | "desc";

export function FileList({ files, folders, onFolderClick, onContextAction }: FileListProps) {
  const [selectedItems, setSelectedItems] = useState<string[]>([]);
  const [sortField, setSortField] = useState<SortField>("name");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDirection("asc");
    }
  };

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

  const toggleSelect = (item: FileType | FolderType) => {
    setSelectedItems((prev) =>
      prev.includes(item.id) ? prev.filter((id) => id !== item.id) : [...prev, item.id],
    );
  };

  const folderRows = folders.map((folder) => (
    <ContextMenu key={folder.id}>
      <ContextMenuTrigger>
        <TableRow
          className={`cursor-pointer ${selectedItems.includes(folder.id) ? "bg-muted" : ""}`}
          onClick={(e) => handleItemClick(folder, e)}
        >
          <TableCell className="w-10">
            <Checkbox
              checked={selectedItems.includes(folder.id)}
              onCheckedChange={() => toggleSelect(folder)}
              onClick={(e) => e.stopPropagation()}
            />
          </TableCell>
          <TableCell>
            <div className="flex items-center space-x-2">
              <Folder className="h-5 w-5 text-icebox-600" />
              <span>{folder.name}</span>
            </div>
          </TableCell>
          <TableCell>Folder</TableCell>
          <TableCell>{folder.itemCount} items</TableCell>
          <TableCell>{folder.lastModified}</TableCell>
          <TableCell>{folder.owner}</TableCell>
        </TableRow>
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuItem onClick={() => onContextAction("Open", folder)}>Open</ContextMenuItem>
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
  ));

  const fileRows = files.map((file) => {
    const FileIcon = fileIconMap[file.extension] || File;
    const isRefrigerated = file.isRefrigerated;
    const isDistributed = file.isDistributed;

    return (
      <ContextMenu key={file.id}>
        <ContextMenuTrigger>
          <TableRow
            className={`cursor-pointer ${selectedItems.includes(file.id) ? "bg-muted" : ""}`}
            onClick={(e) => handleItemClick(file, e)}
          >
            <TableCell className="w-10">
              <Checkbox
                checked={selectedItems.includes(file.id)}
                onCheckedChange={() => toggleSelect(file)}
                onClick={(e) => e.stopPropagation()}
              />
            </TableCell>
            <TableCell>
              <div className="flex items-center space-x-2">
                <FileIcon
                  className={`h-5 w-5 ${isRefrigerated ? "text-icebox-700" : "text-muted-foreground"}`}
                />
                <span className="flex items-center gap-2">
                  {file.name}
                  <div className="flex items-center gap-1">
                    {isRefrigerated && (
                      <Badge variant="outline" className="text-xs">
                        <Snowflake className="h-3 w-3 mr-1" />
                        Refrigerated
                      </Badge>
                    )}
                    {isDistributed && (
                      <Badge variant="secondary" className="text-xs">
                        <Cloud className="h-3 w-3 mr-1" />
                        Distributed
                      </Badge>
                    )}
                  </div>
                </span>
              </div>
            </TableCell>
            <TableCell>{file.extension.toUpperCase()}</TableCell>
            <TableCell>{file.size}</TableCell>
            <TableCell>{file.lastModified}</TableCell>
            <TableCell>{file.owner}</TableCell>
          </TableRow>
        </ContextMenuTrigger>
        <ContextMenuContent>
          <ContextMenuItem onClick={() => onContextAction("Open", file)}>Open</ContextMenuItem>
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
  });

  return (
    <div className="w-full overflow-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-10">
              <Checkbox />
            </TableHead>
            <TableHead className="cursor-pointer" onClick={() => handleSort("name")}>
              <div className="flex items-center space-x-1">
                <span>Name</span>
                {sortField === "name" &&
                  (sortDirection === "asc" ? (
                    <ChevronUp className="h-4 w-4" />
                  ) : (
                    <ChevronDown className="h-4 w-4" />
                  ))}
              </div>
            </TableHead>
            <TableHead className="cursor-pointer" onClick={() => handleSort("type")}>
              <div className="flex items-center space-x-1">
                <span>Type</span>
                {sortField === "type" &&
                  (sortDirection === "asc" ? (
                    <ChevronUp className="h-4 w-4" />
                  ) : (
                    <ChevronDown className="h-4 w-4" />
                  ))}
              </div>
            </TableHead>
            <TableHead className="cursor-pointer" onClick={() => handleSort("size")}>
              <div className="flex items-center space-x-1">
                <span>Size</span>
                {sortField === "size" &&
                  (sortDirection === "asc" ? (
                    <ChevronUp className="h-4 w-4" />
                  ) : (
                    <ChevronDown className="h-4 w-4" />
                  ))}
              </div>
            </TableHead>
            <TableHead className="cursor-pointer" onClick={() => handleSort("lastModified")}>
              <div className="flex items-center space-x-1">
                <span>Modified</span>
                {sortField === "lastModified" &&
                  (sortDirection === "asc" ? (
                    <ChevronUp className="h-4 w-4" />
                  ) : (
                    <ChevronDown className="h-4 w-4" />
                  ))}
              </div>
            </TableHead>
            <TableHead>Owner</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {folderRows}
          {fileRows}
        </TableBody>
      </Table>
    </div>
  );
}
