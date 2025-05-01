import { useEffect, useRef, useState } from "react";
import { Network, Server, User, Link, AlertCircle } from "lucide-react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

// Mock data for nodes in the network
const mockNodes = [
  { id: "node-1", name: "Main Server", type: "server", status: "online", isClickable: true },
  { id: "node-2", name: "Development Server", type: "server", status: "online", isClickable: true },
  { id: "node-3", name: "Backup Server", type: "server", status: "offline", isClickable: false },
  { id: "user-1", name: "John Doe", type: "user", status: "online", isClickable: true },
  { id: "user-2", name: "Jane Smith", type: "user", status: "online", isClickable: true },
  { id: "user-3", name: "Mark Johnson", type: "user", status: "online", isClickable: true },
  { id: "user-4", name: "Sarah Williams", type: "user", status: "away", isClickable: false },
  { id: "user-5", name: "Robert Brown", type: "user", status: "offline", isClickable: false },
];

// Mock connections between nodes
const mockConnections = [
  { source: "node-1", target: "node-2" },
  { source: "node-1", target: "node-3" },
  { source: "node-1", target: "user-1" },
  { source: "node-1", target: "user-2" },
  { source: "node-2", target: "user-3" },
  { source: "node-2", target: "user-4" },
  { source: "node-3", target: "user-5" },
];

interface NodeNetworkProps {
  onNodeClick: (nodeId: string) => void;
}

export function NodeNetwork({ onNodeClick }: NodeNetworkProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [nodes, setNodes] = useState<Array<any>>([]);
  const [connections, setConnections] = useState<Array<any>>([]);
  const [containerSize, setContainerSize] = useState({ width: 1000, height: 600 });

  useEffect(() => {
    // Position nodes in a force-directed layout simulation
    // For simplicity, we'll use fixed positions in this example
    const center = {
      x: containerSize.width / 2,
      y: containerSize.height / 2,
    };

    // Position server nodes in a triangle in the center
    const serverNodes = mockNodes.filter((node) => node.type === "server");
    const serverPositions = serverNodes.map((node, index) => {
      const angle = index * ((2 * Math.PI) / serverNodes.length) - Math.PI / 2;
      const radius = 100;
      return {
        ...node,
        x: center.x + radius * Math.cos(angle),
        y: center.y + radius * Math.sin(angle),
      };
    });

    // Position user nodes in a circle around the servers
    const userNodes = mockNodes.filter((node) => node.type === "user");
    const userPositions = userNodes.map((node, index) => {
      const angle = index * ((2 * Math.PI) / userNodes.length) - Math.PI / 2;
      const radius = 250;
      return {
        ...node,
        x: center.x + radius * Math.cos(angle),
        y: center.y + radius * Math.sin(angle),
      };
    });

    setNodes([...serverPositions, ...userPositions]);
    setConnections(mockConnections);

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
    return () => window.removeEventListener("resize", updateSize);
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
            const isActive = sourceNode.status === "online" && targetNode.status === "online";

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
            const isActive = sourceNode.status === "online" && targetNode.status === "online";

            // Calculate the angle of the line
            const angle = (Math.atan2(line.y2 - line.y1, line.x2 - line.x1) * 180) / Math.PI;

            // Calculate the position of the arrowhead (10px before the target)
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
          const isClickable = node.status === "online" && node.isClickable;
          const isServer = node.type === "server";
          const IconComponent = isServer ? Server : User;

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
                    <div
                      className={`${
                        isServer
                          ? "bg-purple-100 dark:bg-purple-900"
                          : "bg-blue-100 dark:bg-blue-900"
                      } p-4 rounded-full`}
                    >
                      <IconComponent
                        className={`h-8 w-8 ${
                          isServer
                            ? "text-purple-600 dark:text-purple-400"
                            : "text-blue-600 dark:text-blue-400"
                        }`}
                      />
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
                      {node.name}
                    </div>
                  </div>
                </TooltipTrigger>
                <TooltipContent>
                  <div className="flex flex-col">
                    <span className="font-bold">{node.name}</span>
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
