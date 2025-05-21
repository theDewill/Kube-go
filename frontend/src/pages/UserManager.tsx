import { useState, useEffect } from "react";
import { AppLayout } from "@/components/layout/AppLayout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import {
  RefreshCw,
  UserPlus,
  Trash2,
  User,
  Shield,
  Clock,
  Camera,
  MoreHorizontal,
  Check,
  X,
} from "lucide-react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

// Import the Go bindings for user management
import { ListUsers, DeleteUser } from "@/../wailsjs/go/security/UserManager";

// Define the user interface type
interface User {
  id: number;
  email: string;
  facial_data_id?: string;
  created_at: string;
  last_login_at?: string;
  is_active: boolean;
  has_facial_auth: boolean;
  is_admin: boolean;
}

const UserManagement = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [userToDelete, setUserToDelete] = useState<User | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [currentUserID, setCurrentUserID] = useState<number | null>(null);

  // Load users on component mount
  useEffect(() => {
    loadUsers();

    // Try to get current user ID from localStorage
    const sessionID = localStorage.getItem("session_id");
    if (sessionID) {
      // We don't have a direct way to get user ID from session ID in the frontend
      // You might need to add a GetCurrentUserID binding or store user ID in localStorage
      // For now, we'll use a placeholder
      const storedUser = localStorage.getItem("user");
      if (storedUser) {
        try {
          const user = JSON.parse(storedUser);
          setCurrentUserID(user.id);
        } catch (e) {
          console.error("Failed to parse stored user:", e);
        }
      }
    }
  }, []);

  const loadUsers = async () => {
    try {
      setIsLoading(true);
      setError(null);

      const userList = await ListUsers();
      setUsers(userList);
      setIsLoading(false);
    } catch (err) {
      console.error("Error loading users:", err);
      setError("Failed to load users. Please try again.");
      setIsLoading(false);
    }
  };

  const handleDeleteUser = async (user: User) => {
    setUserToDelete(user);
    setDeleteDialogOpen(true);
  };

  const confirmDeleteUser = async () => {
    if (!userToDelete) return;

    try {
      setIsDeleting(true);

      // Check if trying to delete themselves
      if (userToDelete.id === currentUserID) {
        toast.error("You cannot delete your own account");
        setIsDeleting(false);
        setDeleteDialogOpen(false);
        return;
      }

      // Check if trying to delete the last admin
      if (userToDelete.is_admin) {
        const adminCount = users.filter((u) => u.is_admin).length;
        if (adminCount <= 1) {
          toast.error("Cannot delete the last administrator account");
          setIsDeleting(false);
          setDeleteDialogOpen(false);
          return;
        }
      }

      await DeleteUser(userToDelete.id);
      toast.success(`User ${userToDelete.email} deleted successfully`);

      // Refresh user list
      await loadUsers();
    } catch (err) {
      console.error("Failed to delete user:", err);
      toast.error(`Failed to delete user: ${err}`);
    } finally {
      setIsDeleting(false);
      setDeleteDialogOpen(false);
      setUserToDelete(null);
    }
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return "Never";

    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const getTimeSince = (dateString?: string) => {
    if (!dateString) return "Never logged in";

    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();

    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    if (diffDays > 0) {
      return `${diffDays} ${diffDays === 1 ? "day" : "days"} ago`;
    }

    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    if (diffHours > 0) {
      return `${diffHours} ${diffHours === 1 ? "hour" : "hours"} ago`;
    }

    const diffMinutes = Math.floor(diffMs / (1000 * 60));
    if (diffMinutes > 0) {
      return `${diffMinutes} ${diffMinutes === 1 ? "minute" : "minutes"} ago`;
    }

    return "Just now";
  };

  return (
    <AppLayout>
      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold">Kube User Manager</h1>
            <p className="text-muted-foreground">
              Manage user accounts integrated to this kube node and has operations access
            </p>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              className="bg-green-500"
              size="sm"
              onClick={loadUsers}
              disabled={isLoading}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
          </div>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <User className="mr-2 h-5 w-5" />
              Node Users
            </CardTitle>
            <CardDescription className="text-blue-600">
              {users.length} active {users.length === 1 ? "user" : "users"} has access
            </CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="flex items-center justify-center py-8">
                <RefreshCw className="h-8 w-8 animate-spin text-primary" />
                <span className="ml-2">Loading users...</span>
              </div>
            ) : error ? (
              <div className="flex flex-col items-center justify-center py-8">
                <p className="text-destructive">{error}</p>
                <Button variant="outline" className="mt-4" onClick={loadUsers} disabled={isLoading}>
                  Try Again
                </Button>
              </div>
            ) : (
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Email</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Authentication Methods</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead>Last Login</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {users.map((user) => (
                      <TableRow key={user.id}>
                        <TableCell className="font-medium">
                          <div className="flex items-center">
                            {user.email}
                            {user.is_admin && (
                              <TooltipProvider>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Badge variant="outline" className="ml-2">
                                      <Shield className="h-3 w-3 mr-1 text-primary" />
                                      Admin
                                    </Badge>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    <p>Administrator account</p>
                                  </TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            )}
                            {user.id === currentUserID && (
                              <Badge variant="secondary" className="ml-2">
                                You
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          {user.is_active ? (
                            <Badge variant="success" className="flex items-center">
                              <Check className="h-3 w-3 mr-1" />
                              Active
                            </Badge>
                          ) : (
                            <Badge variant="destructive" className="flex items-center">
                              <X className="h-3 w-3 mr-1" />
                              Disabled
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell>
                          <div className="flex space-x-1">
                            {user.has_facial_auth && (
                              <Badge variant="outline" className="flex items-center">
                                <Camera className="h-3 w-3 mr-1" />
                                Facial
                              </Badge>
                            )}
                            {user.email.includes("@") && (
                              <Badge variant="outline" className="flex items-center">
                                <User className="h-3 w-3 mr-1" />
                                Password
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger className="flex items-center text-muted-foreground">
                                <Clock className="h-3 w-3 mr-1" />
                                {new Date(user.created_at).toLocaleDateString()}
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>{formatDate(user.created_at)}</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </TableCell>
                        <TableCell>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger className="text-muted-foreground">
                                {user.last_login_at ? getTimeSince(user.last_login_at) : "Never"}
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>{formatDate(user.last_login_at)}</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="outline"
                            className="text-destructive focus:text-destructive"
                            onClick={() => handleDeleteUser(user)}
                            disabled={user.id === currentUserID}
                          >
                            Delete
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                    {users.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center py-6 text-muted-foreground">
                          No users found. Create a new user to get started.
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Delete User Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete User</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete user <strong>{userToDelete?.email}</strong>?
              <br />
              {userToDelete?.has_facial_auth && (
                <span className="block mt-2 text-destructive">
                  This will also delete all associated facial authentication data.
                </span>
              )}
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDeleteUser}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? (
                <>
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                  Deleting...
                </>
              ) : (
                <>
                  <Trash2 className="h-4 w-4 mr-2" />
                  Delete
                </>
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </AppLayout>
  );
};

export default UserManagement;
