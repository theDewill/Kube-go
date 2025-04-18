
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { 
  LayoutGrid, 
  List, 
  Menu, 
  Bell, 
  Search as SearchIcon, 
  Moon, 
  Sun, 
  X,
  PowerOff
} from "lucide-react";
import { useIsMobile } from "@/hooks/use-mobile";
import { 
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import { toast } from "sonner";
import { LogoutWarning } from "@/components/auth/LogoutWarning";

interface HeaderProps {
  toggleSidebar: () => void;
}

export function Header({ toggleSidebar }: HeaderProps) {
  const isMobile = useIsMobile();
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [showLogoutWarning, setShowLogoutWarning] = useState(false);

  const toggleViewMode = () => {
    const newMode = viewMode === "grid" ? "list" : "grid";
    setViewMode(newMode);
    toast.success(`Switched to ${newMode} view`);
  };

  const toggleDarkMode = () => {
    setIsDarkMode(!isDarkMode);
    document.documentElement.classList.toggle("dark");
    toast.success(`Switched to ${!isDarkMode ? "dark" : "light"} mode`);
  };

  const handleNotificationsClick = () => {
    toast.info("Notification center coming soon");
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    const formData = new FormData(e.target as HTMLFormElement);
    const query = formData.get("search") as string;
    if (query) {
      toast.info(`Searching for "${query}"`);
    }
  };

  return (
    <>
      <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="flex h-16 items-center px-4 md:px-6">
          <Button variant="ghost" size="icon" onClick={toggleSidebar} className="mr-2">
            <Menu className="h-5 w-5" />
            <span className="sr-only">Toggle sidebar</span>
          </Button>
          
          {isMobile && !isSearchOpen ? (
            <div className="flex items-center space-x-2">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setIsSearchOpen(true)}
              >
                <SearchIcon className="h-5 w-5" />
                <span className="sr-only">Search</span>
              </Button>
            </div>
          ) : isMobile && isSearchOpen ? (
            <form onSubmit={handleSearch} className="flex-1 flex items-center">
              <Input
                name="search"
                placeholder="Search files and folders..."
                className="h-9 md:w-[300px] lg:w-[400px]"
                autoFocus
              />
              <Button
                variant="ghost"
                size="icon"
                type="button"
                onClick={() => setIsSearchOpen(false)}
                className="ml-2"
              >
                <X className="h-5 w-5" />
                <span className="sr-only">Close search</span>
              </Button>
            </form>
          ) : (
            <form onSubmit={handleSearch} className="ml-auto flex-1 md:ml-0 md:flex-initial">
              <div className="relative">
                <SearchIcon className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  name="search"
                  placeholder="Search files and folders..."
                  className="pl-8 h-9 md:w-[300px] lg:w-[400px]"
                />
              </div>
            </form>
          )}
          
          <div className="ml-auto flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleViewMode}
              className="hidden md:flex"
            >
              {viewMode === "grid" ? (
                <List className="h-5 w-5" />
              ) : (
                <LayoutGrid className="h-5 w-5" />
              )}
              <span className="sr-only">
                Switch to {viewMode === "grid" ? "list" : "grid"} view
              </span>
            </Button>
            
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleDarkMode}
              className="hidden md:flex"
            >
              {isDarkMode ? (
                <Sun className="h-5 w-5" />
              ) : (
                <Moon className="h-5 w-5" />
              )}
              <span className="sr-only">
                Toggle {isDarkMode ? "light" : "dark"} mode
              </span>
            </Button>
            
            <Button
              variant="ghost"
              size="icon"
              onClick={handleNotificationsClick}
            >
              <Bell className="h-5 w-5" />
              <span className="sr-only">Notifications</span>
            </Button>
            
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setShowLogoutWarning(true)}
              className="text-destructive hover:bg-destructive/10"
            >
              <PowerOff className="h-5 w-5" />
              <span className="sr-only">Log out</span>
            </Button>
            
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="rounded-full">
                  <Avatar className="h-8 w-8">
                    <AvatarImage src="/placeholder.svg" alt="User" />
                    <AvatarFallback>JP</AvatarFallback>
                  </Avatar>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>My Account</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => toast.info("Profile settings coming soon")}>
                  Profile
                </DropdownMenuItem>
                <DropdownMenuItem onClick={toggleDarkMode} className="md:hidden">
                  {isDarkMode ? "Light mode" : "Dark mode"}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => toast.info("Settings coming soon")}>
                  Settings
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setShowLogoutWarning(true)}>
                  Log out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>
      
      {showLogoutWarning && (
        <LogoutWarning onCancel={() => setShowLogoutWarning(false)} />
      )}
    </>
  );
}
