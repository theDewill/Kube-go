
import { useState } from "react";
import { AppLayout } from "@/components/layout/AppLayout";
import { NodeNetwork } from "@/components/nodes/NodeNetwork";
import { FileBrowser } from "@/components/file-browser/FileBrowser";
import { Button } from "@/components/ui/button";
import { LayoutGrid, Network, RefreshCw } from "lucide-react";
import { toast } from "sonner";

const Nodes = () => {
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  
  const handleNodeClick = (nodeId: string) => {
    setSelectedNode(nodeId);
    toast.info(`Connected to node: ${nodeId}`);
  };
  
  const refreshNodes = () => {
    setIsRefreshing(true);
    setTimeout(() => {
      setIsRefreshing(false);
      toast.success("Node network refreshed");
    }, 1000);
  };
  
  const resetSelection = () => {
    setSelectedNode(null);
  };

  return (
    <AppLayout>
      <div className="h-full flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold flex items-center">
            <Network className="h-6 w-6 mr-2 text-icebox-600" />
            Network Nodes
          </h1>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={refreshNodes}
              disabled={isRefreshing}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? "animate-spin" : ""}`} />
              Refresh Nodes
            </Button>
            {selectedNode && (
              <Button variant="outline" size="sm" onClick={resetSelection}>
                <LayoutGrid className="h-4 w-4 mr-2" />
                Back to Network
              </Button>
            )}
          </div>
        </div>
        
        {selectedNode ? (
          <div className="flex-1 overflow-hidden">
            <div className="mb-2 text-sm text-muted-foreground">
              Browsing files on node: <span className="font-medium">{selectedNode}</span>
            </div>
            <FileBrowser />
          </div>
        ) : (
          <div className="flex-1 overflow-hidden bg-sidebar/5 rounded-lg">
            <NodeNetwork onNodeClick={handleNodeClick} />
          </div>
        )}
      </div>
    </AppLayout>
  );
};

export default Nodes;
