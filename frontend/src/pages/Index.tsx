import { useState } from "react";
import { AppLayout } from "@/components/layout/AppLayout";
import { FileBrowser } from "@/components/file-browser/FileBrowser";
import { StorageOverview } from "@/components/dashboard/StorageOverview";
import { RecentActivity } from "@/components/dashboard/RecentActivity";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { FolderOpen, LayoutDashboard, UserCircle } from "lucide-react";

const Index = () => {
  // const [isAuthenticated, setIsAuthenticated] = useState(false);

  // if (!isAuthenticated) {
  //   return (
  //     <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-950 p-4">
  //       <LoginForm onLogin={() => setIsAuthenticated(true)} />
  //     </div>
  //   );
  // }

  return (
    <AppLayout>
      <Tabs defaultValue="files" className="h-full flex flex-col">
        <div className="flex justify-between items-center mb-4">
          <TabsList>
            <TabsTrigger value="files" className="flex items-center">
              <FolderOpen className="h-4 w-4 mr-2" />
              Files
            </TabsTrigger>
            <TabsTrigger value="dashboard" className="flex items-center">
              <LayoutDashboard className="h-4 w-4 mr-2" />
              Dashboard
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="files" className="flex-1 h-0">
          <FileBrowser />
        </TabsContent>

        <TabsContent value="dashboard" className="space-y-6 flex-1 h-0 overflow-auto">
          <h2 className="text-2xl font-bold">Dashboard</h2>
          <StorageOverview />

          <div className="grid gap-4 md:grid-cols-3">
            <RecentActivity />
          </div>
        </TabsContent>
      </Tabs>
    </AppLayout>
  );
};

export default Index;
