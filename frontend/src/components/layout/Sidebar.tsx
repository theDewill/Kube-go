
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import {
  ChevronLeft,
  ChevronRight,
  Home,
  FolderOpen,
  Share2,
  Star,
  Clock,
  Trash2,
  Settings,
  HardDrive,
  Server,
  Users,
  PieChart,
  Snowflake
} from "lucide-react";
import { toast } from "sonner";
import { useNavigate, useLocation } from "react-router-dom";

interface SidebarProps {
  isOpen: boolean;
  toggleSidebar: () => void;
}

export function Sidebar({ isOpen, toggleSidebar }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();
  
  const handleNavigation = (path: string) => {
    if (path === "Home") {
      navigate("/");
    } else if (path === "Nodes") {
      navigate("/nodes");
    } else {
      toast.info(`Navigating to ${path} (coming soon)`);
    }
  };

  const isActive = (path: string) => {
    if (path === "Home" && location.pathname === "/") return true;
    if (path === "Nodes" && location.pathname === "/nodes") return true;
    return false;
  };

  return (
    <aside
      className={cn(
        "bg-sidebar text-sidebar-foreground flex flex-col border-r transition-all duration-300",
        isOpen ? "w-64" : "w-16"
      )}
    >
      <div className="flex h-16 items-center px-4 justify-between">
        <div className={cn("flex items-center gap-2", !isOpen && "hidden")}>
          <Snowflake className="h-6 w-6 text-icebox-600" />
          <span className="font-semibold text-lg">Drive Space</span>
        </div>
        {!isOpen && (
          <Snowflake className="h-6 w-6 text-icebox-600 mx-auto" />
        )}
        <Button 
          variant="ghost" 
          size="icon" 
          onClick={toggleSidebar}
          className={cn("absolute right-2", !isOpen && "right-auto left-2")}
        >
          {isOpen ? (
            <ChevronLeft className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
          <span className="sr-only">
            {isOpen ? "Close sidebar" : "Open sidebar"}
          </span>
        </Button>
      </div>
      <ScrollArea className="flex-1 pt-4">
        <nav className="grid gap-1 px-2">
          <SidebarItem
            icon={Home}
            label="Home"
            isOpen={isOpen}
            isActive={isActive("Home")}
            onClick={() => handleNavigation("Home")}
          />
          <SidebarItem
            icon={FolderOpen}
            label="My Files"
            isOpen={isOpen}
            isActive={isActive("My Files")}
            onClick={() => handleNavigation("My Files")}
          />
          <SidebarItem
            icon={Share2}
            label="Shared"
            isOpen={isOpen}
            isActive={isActive("Shared")}
            onClick={() => handleNavigation("Shared")}
          />
          <SidebarItem
            icon={Star}
            label="Starred"
            isOpen={isOpen}
            isActive={isActive("Starred")}
            onClick={() => handleNavigation("Starred")}
          />
          <SidebarItem
            icon={Clock}
            label="Recent"
            isOpen={isOpen}
            isActive={isActive("Recent")}
            onClick={() => handleNavigation("Recent")}
          />
          <SidebarItem
            icon={Snowflake}
            label="Refrigerated"
            isOpen={isOpen}
            isActive={isActive("Refrigerated")}
            onClick={() => handleNavigation("Refrigerated")}
          />
          <SidebarItem
            icon={Trash2}
            label="Trash"
            isOpen={isOpen}
            isActive={isActive("Trash")}
            onClick={() => handleNavigation("Trash")}
          />
          
          <Separator className="my-4" />
          
          <div className={cn("px-4 py-2", !isOpen && "hidden")}>
            <h3 className="text-xs font-medium text-muted-foreground">
              Administration
            </h3>
          </div>
          
          <SidebarItem
            icon={Users}
            label="Users"
            isOpen={isOpen}
            isActive={isActive("Users")}
            onClick={() => handleNavigation("Users")}
          />
          <SidebarItem
            icon={Server}
            label="Nodes"
            isOpen={isOpen}
            isActive={isActive("Nodes")}
            onClick={() => handleNavigation("Nodes")}
          />
          <SidebarItem
            icon={HardDrive}
            label="Storage"
            isOpen={isOpen}
            isActive={isActive("Storage")}
            onClick={() => handleNavigation("Storage")}
          />
          <SidebarItem
            icon={PieChart}
            label="Analytics"
            isOpen={isOpen}
            isActive={isActive("Analytics")}
            onClick={() => handleNavigation("Analytics")}
          />
          <SidebarItem
            icon={Settings}
            label="Settings"
            isOpen={isOpen}
            isActive={isActive("Settings")}
            onClick={() => handleNavigation("Settings")}
          />
        </nav>
      </ScrollArea>
      
      <div className={cn(
        "border-t p-4",
        !isOpen && "p-2"
      )}>
        <div className="flex items-center gap-3">
          <div className="relative">
            <HardDrive className="h-5 w-5 text-muted-foreground" />
            <div className="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full bg-green-500 ring-1 ring-white" />
          </div>
          {isOpen && (
            <div className="space-y-1">
              <p className="text-xs font-medium leading-none">Storage</p>
              <p className="text-xs text-muted-foreground">45% used (450 GB / 1 TB)</p>
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}

interface SidebarItemProps {
  icon: React.ElementType;
  label: string;
  isOpen: boolean;
  isActive?: boolean;
  onClick: () => void;
}

function SidebarItem({ icon: Icon, label, isOpen, isActive = false, onClick }: SidebarItemProps) {
  return (
    <Button
      variant={isActive ? "secondary" : "ghost"}
      className={cn(
        "w-full justify-start",
        !isOpen && "justify-center px-0"
      )}
      onClick={onClick}
    >
      <Icon className={cn("h-5 w-5", isOpen && "mr-2")} />
      {isOpen && <span>{label}</span>}
      {!isOpen && <span className="sr-only">{label}</span>}
    </Button>
  );
}
