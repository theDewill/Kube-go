import { useEffect } from "react";
import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AuthProvider, useAuth } from "@/contexts/AuthContext";
import Index from "./pages/Index";
import Nodes from "./pages/Nodes";
import Refrigerated from "./pages/Refrigerated";
import NotFound from "./pages/NotFound";
import Finder from "./pages/Finder";
import Settings from "./pages/Settings";
import { LoginForm } from "@/components/auth/LoginForm";
import UserManagement from "./pages/UserManager";

const queryClient = new QueryClient();

// Inner component that has access to auth context
const AppRouter = () => {
  const { user, sessionId, isLoading, isAuthenticated } = useAuth();

  useEffect(() => {
    console.log("Auth state changed:", {
      user: !!user,
      sessionId: !!sessionId,
      isLoading,
      isAuthenticated,
    });
  }, [user, sessionId, isLoading, isAuthenticated]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary">
          Loading...
        </div>
      </div>
    );
  }

  // If authenticated, show protected routes
  if (isAuthenticated) {
    return (
      <Routes>
        <Route path="/" element={<Index />} />
        <Route path="/files" element={<Index />} />
        <Route path="/nodes" element={<Nodes />} />
        <Route path="/refrigerated" element={<Refrigerated />} />
        <Route path="/finder" element={<Finder />} />
        <Route path="/users" element={<UserManagement />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/login" element={<Index />} />
        <Route path="*" element={<NotFound />} />
      </Routes>
    );
  }

  // If not authenticated, show login
  return (
    <Routes>
      <Route path="*" element={<LoginForm />} />
    </Routes>
  );
};

const App = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <TooltipProvider>
          <Toaster />
          <Sonner />
          <BrowserRouter>
            <AppRouter />
          </BrowserRouter>
        </TooltipProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
};

export default App;
