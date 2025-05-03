import { useEffect, useRef, useState } from "react";
import { Server, User, Lock, AlertCircle } from "lucide-react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

import { GetNodesForFrontend } from "@/../wailsjs/go/kubenet/NodeRegistry";

// Define the Node type based on your Golang struct
interface Node {
  id: string;
  ip: string;
  hostname: string;
  lastSeen: string;
  version: number;
  broadcasted: boolean;
  status: string;
  isLocal: boolean;
}

// Connections will be calculated based on the network topology
interface Connection {
  source: string;
  target: string;
  isActive: boolean;
}

interface NodeNetworkProps {
  onNodeClick: (nodeId: string) => void;
}

export function NodeNetwork({ onNodeClick }: NodeNetworkProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [nodes, setNodes] = useState<Array<any>>([]);
  const [connections, setConnections] = useState<Array<Connection>>([]);
  const [containerSize, setContainerSize] = useState({ width: 1000, height: 600 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Function to fetch nodes from the backend
  const fetchNodes = async () => {
    try {
      setLoading(true);
      // Call the Go function exposed by Wails
      const nodesData = await GetNodesForFrontend();

      // Process nodes for visualization
      processNodes(nodesData);
      setLoading(false);
    } catch (err) {
      console.error("Failed to fetch nodes:", err);
      setError("Failed to fetch network nodes. Please try again.");
      setLoading(false);
    }
  };

  // Process nodes and create visualization data
  const processNodes = (nodesData: Node[]) => {
    // Transform backend nodes to visual nodes
    const center = {
      x: containerSize.width / 2,
      y: containerSize.height / 2,
    };

    // Find local node
    const localNode = nodesData.find((node) => node.isLocal);

    // Position nodes in a network layout
    // Place local node at center
    const positionedNodes = nodesData.map((node, index) => {
      // Default to user type, we'll determine by hostname pattern if it's a server
      const nodeType = node.hostname.toLowerCase().includes("server") ? "server" : "user";

      let x, y;

      if (node.isLocal) {
        // Local node goes at the center
        x = center.x;
        y = center.y;
      } else {
        // Calculate positions in a circle around the local node
        const totalNodes = nodesData.length - 1; // Excluding local node
        const nodeIndex = nodesData.indexOf(node) - (node.id > localNode?.id ? 1 : 0);
        const angle = nodeIndex * ((2 * Math.PI) / totalNodes) - Math.PI / 2;
        const radius = 250;

        x = center.x + radius * Math.cos(angle);
        y = center.y + radius * Math.sin(angle);
      }

      // Map status from backend to what the UI expects
      let uiStatus = "offline";
      switch (node.status) {
        case "online":
          uiStatus = "online";
          break;
        case "locked":
          uiStatus = "away";
          break;
        case "offline":
        default:
          uiStatus = "offline";
          break;
      }

      return {
        ...node,
        x,
        y,
        type: nodeType,
        name: node.hostname,
        status: uiStatus,
        isClickable: uiStatus !== "offline",
      };
    });

    setNodes(positionedNodes);

    // Generate connections
    // For a simple network, connect local node to all others
    const newConnections: Connection[] = [];

    if (localNode) {
      positionedNodes.forEach((node) => {
        if (!node.isLocal) {
          newConnections.push({
            source: localNode.id,
            target: node.id,
            isActive: node.status !== "offline",
          });
        }
      });
    }

    // Also add some random connections between other nodes for visualization appeal
    positionedNodes.forEach((source, i) => {
      positionedNodes.forEach((target, j) => {
        if (i !== j && !source.isLocal && !target.isLocal && Math.random() > 0.75) {
          newConnections.push({
            source: source.id,
            target: target.id,
            isActive: source.status !== "offline" && target.status !== "offline",
          });
        }
      });
    });

    setConnections(newConnections);
  };

  // Initialize and set up polling
  useEffect(() => {
    // Initial fetch
    fetchNodes();

    // Set up polling interval (every 5 seconds)
    const interval = setInterval(fetchNodes, 5000);

    // Handle window resize
    const updateSize = () => {
      if (containerRef.current) {
        setContainerSize({
          width: containerRef.current.offsetWidth,
          height: containerRef.current.offsetHeight,
        });
      }
    };

    updateSize();
    window.addEventListener("resize", updateSize);

    // Cleanup
    return () => {
      clearInterval(interval);
      window.removeEventListener("resize", updateSize);
    };
  }, []);

  // Update layout when container size changes
  useEffect(() => {
    if (nodes.length > 0) {
      fetchNodes(); // Re-process nodes with new container size
    }
  }, [containerSize.width, containerSize.height]);

  // Draw a straight line between two nodes
  const drawLine = (sourceNode: any, targetNode: any) => {
    const sourceX = sourceNode.x;
    const sourceY = sourceNode.y;
    const targetX = targetNode.x;
    const targetY = targetNode.y;

    return {
      x1: sourceX,
      y1: sourceY,
      x2: targetX,
      y2: targetY,
    };
  };

  if (loading && nodes.length === 0) {
    return (
      <div className="w-full h-full flex items-center justify-center">
        <div className="flex flex-col items-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
          <div className="mt-4 text-lg font-medium">Discovering network nodes...</div>
        </div>
      </div>
    );
  }

  if (error && nodes.length === 0) {
    return (
      <div className="w-full h-full flex items-center justify-center">
        <div className="flex flex-col items-center text-destructive">
          <AlertCircle className="h-12 w-12" />
          <div className="mt-4 text-lg font-medium">{error}</div>
          <button
            className="mt-4 px-4 py-2 bg-primary text-primary-foreground rounded-md"
            onClick={fetchNodes}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full h-full relative overflow-hidden" ref={containerRef}>
      <div className="absolute inset-0">
        <svg width="100%" height="100%">
          {/* Render connections as straight lines */}
          {connections.map((connection, index) => {
            const sourceNode = nodes.find((node) => node.id === connection.source);
            const targetNode = nodes.find((node) => node.id === connection.target);

            if (!sourceNode || !targetNode) return null;

            const line = drawLine(sourceNode, targetNode);
            const isActive = connection.isActive;

            return (
              <line
                key={`connection-${index}`}
                x1={line.x1}
                y1={line.y1}
                x2={line.x2}
                y2={line.y2}
                stroke={isActive ? "#1EAEDB" : "#8E9196"}
                strokeWidth={isActive ? 2 : 1}
                strokeDasharray={!isActive ? "4 4" : "none"}
              />
            );
          })}

          {/* Add arrowheads at the end of lines */}
          {connections.map((connection, index) => {
            const sourceNode = nodes.find((node) => node.id === connection.source);
            const targetNode = nodes.find((node) => node.id === connection.target);

            if (!sourceNode || !targetNode) return null;

            const line = drawLine(sourceNode, targetNode);
            const isActive = connection.isActive;

            // Calculate the angle of the line
            const angle = (Math.atan2(line.y2 - line.y1, line.x2 - line.x1) * 180) / Math.PI;

            // Calculate the position of the arrowhead
            const arrowLength = 10;
            const dx = line.x2 - line.x1;
            const dy = line.y2 - line.y1;
            const length = Math.sqrt(dx * dx + dy * dy);
            const unitDx = dx / length;
            const unitDy = dy / length;

            const arrowX = line.x2 - unitDx * 20; // 20px from target
            const arrowY = line.y2 - unitDy * 20;

            return (
              <polygon
                key={`arrow-${index}`}
                points={`0,-3 0,3 6,0`}
                fill={isActive ? "#1EAEDB" : "#8E9196"}
                transform={`translate(${arrowX},${arrowY}) rotate(${angle})`}
              />
            );
          })}
        </svg>

        {/* Render nodes */}
        {nodes.map((node) => {
          const isClickable = node.status !== "offline" && node.isClickable;
          const isServer = node.type === "server";
          const isLocked = node.status === "away";

          // Choose the appropriate icon
          let IconComponent;
          if (isServer) {
            IconComponent = Server;
          } else if (isLocked) {
            IconComponent = Lock;
          } else {
            IconComponent = User;
          }

          // Special styling for local node
          const nodeColors = node.isLocal
            ? {
                bg: "bg-green-100 dark:bg-green-900",
                text: "text-green-600 dark:text-green-400",
                border: "border-4 border-green-500",
              }
            : isServer
              ? {
                  bg: "bg-purple-100 dark:bg-purple-900",
                  text: "text-purple-600 dark:text-purple-400",
                  border: "",
                }
              : {
                  bg: "bg-blue-100 dark:bg-blue-900",
                  text: "text-blue-600 dark:text-blue-400",
                  border: "",
                };

          return (
            <TooltipProvider key={node.id}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <div
                    className={`absolute transform -translate-x-1/2 -translate-y-1/2 rounded-full flex items-center justify-center transition-transform ${
                      isClickable ? "cursor-pointer hover:scale-110" : "opacity-60"
                    }`}
                    style={{
                      left: node.x,
                      top: node.y,
                      filter: !isClickable ? "blur(1px)" : "none",
                    }}
                    onClick={() => isClickable && onNodeClick(node.id)}
                  >
                    <div className={`${nodeColors.bg} p-4 rounded-full ${nodeColors.border}`}>
                      <IconComponent className={`h-8 w-8 ${nodeColors.text}`} />
                    </div>
                    {node.status === "offline" && (
                      <div className="absolute -top-1 -right-1 bg-red-500 p-1 rounded-full">
                        <AlertCircle className="h-3 w-3 text-white" />
                      </div>
                    )}
                    {node.status === "away" && (
                      <div className="absolute -top-1 -right-1 bg-yellow-500 p-1 rounded-full">
                        <AlertCircle className="h-3 w-3 text-white" />
                      </div>
                    )}
                    <div className="absolute mt-10 text-center font-medium text-sm whitespace-nowrap">
                      {node.hostname}
                      {node.isLocal ? " (You)" : ""}
                    </div>
                  </div>
                </TooltipTrigger>
                <TooltipContent>
                  <div className="flex flex-col">
                    <span className="font-bold">
                      {node.hostname}
                      {node.isLocal ? " (You)" : ""}
                    </span>
                    <span className="text-xs">IP: {node.ip}</span>
                    <span className="text-xs capitalize">Type: {node.type}</span>
                    <span
                      className={`text-xs ${
                        node.status === "online"
                          ? "text-green-500"
                          : node.status === "away"
                            ? "text-yellow-500"
                            : "text-red-500"
                      }`}
                    >
                      Status: {node.status}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      Last seen: {new Date(node.lastSeen).toLocaleTimeString()}
                    </span>
                    {isClickable ? (
                      <span className="text-xs text-muted-foreground">Click to connect</span>
                    ) : (
                      <span className="text-xs text-muted-foreground">Not available</span>
                    )}
                  </div>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          );
        })}
      </div>
    </div>
  );
}
