import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AuthProvider, ProtectedRoute } from "@/contexts/AuthContext";
import Index from "./pages/Index";
import Nodes from "./pages/Nodes";
import Refrigerated from "./pages/Refrigerated";
import NotFound from "./pages/NotFound";
import Finder from "./pages/Finder";
import Settings from "./pages/Settings";
import { LoginForm } from "@/components/auth/LoginForm";

const queryClient = new QueryClient();

const App = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <TooltipProvider>
          <Toaster />
          <Sonner />
          <BrowserRouter>
            <Routes>
              {/* Protected Routes */}
              <Route
                path="/"
                element={
                  <ProtectedRoute fallback={<LoginForm />}>
                    <Index />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/files"
                element={
                  <ProtectedRoute fallback={<LoginForm />}>
                    <Index />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/nodes"
                element={
                  <ProtectedRoute fallback={<LoginForm />}>
                    <Nodes />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/refrigerated"
                element={
                  <ProtectedRoute fallback={<LoginForm />}>
                    <Refrigerated />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/finder"
                element={
                  <ProtectedRoute fallback={<LoginForm />}>
                    <Finder />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/settings"
                element={
                  <ProtectedRoute fallback={<LoginForm />}>
                    <Settings />
                  </ProtectedRoute>
                }
              />

              {/* Public Routes */}
              <Route path="/login" element={<LoginForm />} />

              {/* Catch-all route - must be last */}
              <Route
                path="*"
                element={
                  <ProtectedRoute fallback={<LoginForm />}>
                    <NotFound />
                  </ProtectedRoute>
                }
              />
            </Routes>
          </BrowserRouter>
        </TooltipProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
};

export default App;
