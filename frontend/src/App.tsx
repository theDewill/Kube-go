import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import Index from "./pages/Index";
import Nodes from "./pages/Nodes";
import Refrigerated from "./pages/Refrigerated";
import NotFound from "./pages/NotFound";
import Finder from "./pages/Finder";
import Settings from "./pages/Settings";
import { LoginForm } from "@/components/auth/LoginForm";
import { useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
const queryClient = new QueryClient();

const App = () => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  //const navigate = useNavigate();
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <Toaster />
        <Sonner />
        <BrowserRouter>
          <Routes>
            <Route
              path="/"
              element={
                <LoginForm
                  onLogin={() => {
                    setIsAuthenticated(true);
                    //navigate("/files");
                  }}
                />
              }
            />
            <Route path="/files" element={<Index />} />
            <Route path="/nodes" element={<Nodes />} />
            <Route path="/refrigerated" element={<Refrigerated />} />
            <Route path="/finder" element={<Finder />} />
            <Route path="/settings" element={<Settings />} />

            {/* ADD ALL CUSTOM ROUTES ABOVE THE CATCH-ALL "*" ROUTE */}
            <Route path="*" element={<NotFound />} />
          </Routes>
        </BrowserRouter>
      </TooltipProvider>
    </QueryClientProvider>
  );
};

export default App;
