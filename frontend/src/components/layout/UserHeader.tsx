import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import {
  LogOut,
  Settings,
  User,
  Shield,
  Camera,
  LayoutGrid,
  List,
  Menu,
  Bell,
  Search as SearchIcon,
  Moon,
  Sun,
  X,
  PowerOff,
} from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { useNavigate } from "react-router-dom";
import { useIsMobile } from "@/hooks/use-mobile";
import { toast } from "sonner";
import { LogoutWarning } from "@/components/auth/LogoutWarning";

interface UserHeaderProps {
  toggleSidebar: () => void;
}

export function UserHeader({ toggleSidebar }: UserHeaderProps) {
  const { user, logout, isLoading } = useAuth();
  const navigate = useNavigate();
  const isMobile = useIsMobile();

  // States
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [isDarkMode, setIsDarkMode] = useState(false);
  const [showLogoutWarning, setShowLogoutWarning] = useState(false);

  const handleLogout = async () => {
    setIsLoggingOut(true);
    try {
      await logout();
      toast.success("Logged out successfully");
      navigate("/login");
    } catch (error) {
      toast.error("Error logging out");
    } finally {
      setIsLoggingOut(false);
      setShowLogoutWarning(false);
    }
  };

  const handleSettings = () => {
    navigate("/settings");
  };

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

  if (!user) {
    return null;
  }

  // Get user initials for avatar
  const getInitials = (email: string) => {
    const name = email.split("@")[0];
    return name.slice(0, 2).toUpperCase();
  };

  // Format last login
  const formatLastLogin = (lastLogin?: string) => {
    if (!lastLogin) return "Never";
    const date = new Date(lastLogin);
    return date.toLocaleDateString() + " " + date.toLocaleTimeString();
  };

  return (
    <>
      <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="flex h-16 items-center px-4 md:px-6">
          {/* Sidebar Toggle Button */}
          <Button variant="ghost" size="icon" onClick={toggleSidebar} className="mr-2">
            <Menu className="h-5 w-5" />
            <span className="sr-only">Toggle sidebar</span>
          </Button>

          {/* Search Section */}

          {/* Right Side Controls */}
          <div className="ml-auto flex items-center gap-2">
            {/* View Mode Toggle */}
            <Button variant="ghost" size="icon" onClick={toggleViewMode} className="hidden md:flex">
              {viewMode === "grid" ? (
                <List className="h-5 w-5" />
              ) : (
                <LayoutGrid className="h-5 w-5" />
              )}
              <span className="sr-only">
                Switch to {viewMode === "grid" ? "list" : "grid"} view
              </span>
            </Button>

            {/* Dark Mode Toggle */}
            <Button variant="ghost" size="icon" onClick={toggleDarkMode} className="hidden md:flex">
              {isDarkMode ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
              <span className="sr-only">Toggle {isDarkMode ? "light" : "dark"} mode</span>
            </Button>

            {/* Notifications */}
            {/* <Button variant="ghost" size="icon" onClick={handleNotificationsClick}>
              <Bell className="h-5 w-5" />
              <span className="sr-only">Notifications</span>
            </Button> */}

            {/* Quick Logout Button */}
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setShowLogoutWarning(true)}
              className="text-destructive hover:bg-destructive/10"
            >
              <PowerOff className="h-5 w-5" />
              <span className="sr-only">Log out</span>
            </Button>

            {/* User Dropdown */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="relative h-10 w-10 rounded-full">
                  <Avatar className="h-10 w-10">
                    <AvatarFallback className="bg-primary text-primary-foreground">
                      {getInitials(user.email)}
                    </AvatarFallback>
                  </Avatar>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent className="w-64" align="end" forceMount>
                <DropdownMenuLabel className="font-normal">
                  <div className="flex flex-col space-y-1">
                    <p className="text-sm font-medium leading-none">{user.email}</p>
                    <p className="text-xs leading-none text-muted-foreground">User ID: {user.id}</p>
                    <div className="flex items-center space-x-2 mt-2">
                      <Badge variant="outline" className="text-xs">
                        <Shield className="h-3 w-3 mr-1" />
                        Credentials
                      </Badge>
                      {user.has_facial_auth && (
                        <Badge variant="outline" className="text-xs">
                          <Camera className="h-3 w-3 mr-1" />
                          Facial Auth
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      Last login: {formatLastLogin(user.last_login_at)}
                    </p>
                  </div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />

                <DropdownMenuItem onClick={handleSettings}>
                  <Settings className="mr-2 h-4 w-4" />
                  <span>Settings</span>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={toggleViewMode} className="md:hidden">
                  {viewMode === "grid" ? (
                    <List className="mr-2 h-4 w-4" />
                  ) : (
                    <LayoutGrid className="mr-2 h-4 w-4" />
                  )}
                  <span>{viewMode === "grid" ? "List view" : "Grid view"}</span>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={toggleDarkMode} className="md:hidden">
                  {isDarkMode ? (
                    <Sun className="mr-2 h-4 w-4" />
                  ) : (
                    <Moon className="mr-2 h-4 w-4" />
                  )}
                  <span>{isDarkMode ? "Light mode" : "Dark mode"}</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => handleLogout()}
                  className="text-red-600 dark:text-red-400"
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  <span>Log out</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>

      {/* Logout Warning Dialog */}
      {showLogoutWarning && (
        <LogoutWarning
          onCancel={() => setShowLogoutWarning(false)}
          onConfirm={handleLogout}
          isLoading={isLoggingOut}
        />
      )}
    </>
  );
}
