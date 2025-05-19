import { useState } from "react";
import { Button } from "@/components/ui/button";
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
import { LogOut, Settings, User, Shield, Camera } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";

export function UserHeader() {
  const { user, logout, isLoading } = useAuth();
  const navigate = useNavigate();
  const [isLoggingOut, setIsLoggingOut] = useState(false);

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
    }
  };

  const handleSettings = () => {
    navigate("/settings");
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
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={handleLogout}
          disabled={isLoggingOut}
          className="text-red-600 dark:text-red-400"
        >
          <LogOut className="mr-2 h-4 w-4" />
          <span>{isLoggingOut ? "Logging out..." : "Log out"}</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
