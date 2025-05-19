import { useState } from "react";
import { Sidebar } from "./Sidebar";
import { Header } from "./Header";
import { UserHeader } from "./UserHeader";
import { toast } from "sonner";

interface AppLayoutProps {
  children: React.ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const [sidebarOpen, setSidebarOpen] = useState(true);

  const toggleSidebar = () => {
    setSidebarOpen(!sidebarOpen);
    toast.success(sidebarOpen ? "Sidebar collapsed" : "Sidebar expanded");
  };

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar isOpen={sidebarOpen} toggleSidebar={toggleSidebar} />
      <div className="flex flex-col flex-1 overflow-hidden">
        <UserHeader toggleSidebar={toggleSidebar} />

        <main className="flex-1 overflow-auto p-4 md:p-6">{children}</main>
      </div>
    </div>
  );
}
