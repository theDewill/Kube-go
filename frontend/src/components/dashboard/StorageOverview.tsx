
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from "recharts";
import { HardDrive, Server, Snowflake } from "lucide-react";

const storageData = [
  { name: "Documents", value: 35, color: "#3684ff" },
  { name: "Images", value: 25, color: "#10b981" },
  { name: "Videos", value: 20, color: "#f97316" },
  { name: "Other", value: 20, color: "#8b5cf6" },
];

const compressionData = [
  { name: "Regular Files", value: 65, color: "#64748b" },
  { name: "Refrigerated", value: 35, color: "#3684ff" },
];

export function StorageOverview() {
  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base font-medium">Storage Usage</CardTitle>
          <CardDescription>450GB of 1TB used</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-4">
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">45% used</span>
                <span className="font-medium">450GB/1TB</span>
              </div>
              <Progress value={45} className="h-2" />
            </div>
            <div className="h-[180px]">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={storageData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={80}
                    paddingAngle={2}
                    dataKey="value"
                  >
                    {storageData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value: number) => [`${value}%`, "Percentage"]}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </div>
        </CardContent>
        <CardFooter className="border-t pt-4 text-xs text-muted-foreground">
          <div className="grid grid-cols-2 gap-2 w-full">
            {storageData.map((item, index) => (
              <div key={index} className="flex items-center gap-1">
                <div
                  className="h-3 w-3 rounded-full"
                  style={{ backgroundColor: item.color }}
                />
                <span>{item.name}: {item.value}%</span>
              </div>
            ))}
          </div>
        </CardFooter>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base font-medium">Refrigeration Status</CardTitle>
          <CardDescription>35% of files are refrigerated</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-4">
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Space saved</span>
                <span className="font-medium">120GB (26%)</span>
              </div>
              <Progress value={26} className="h-2" />
            </div>
            <div className="h-[180px]">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={compressionData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={80}
                    paddingAngle={2}
                    dataKey="value"
                    label={({ name, percent }) => `${name}: ${(percent * 100).toFixed(0)}%`}
                  >
                    {compressionData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value: number) => [`${value}%`, "Percentage"]}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </div>
        </CardContent>
        <CardFooter className="border-t pt-4 flex justify-between text-sm">
          <div className="flex items-center">
            <HardDrive className="h-4 w-4 mr-2 text-gray-500" />
            <span>Regular: 480GB</span>
          </div>
          <div className="flex items-center">
            <Snowflake className="h-4 w-4 mr-2 text-icebox-500" />
            <span>Refrigerated: 256GB</span>
          </div>
        </CardFooter>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base font-medium">Node Status</CardTitle>
          <CardDescription>3 nodes active in network</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <NodeStatus name="Main Server" status="online" load={72} storage={65} />
            <NodeStatus name="Backup Node 1" status="online" load={45} storage={38} />
            <NodeStatus name="Backup Node 2" status="online" load={22} storage={54} />
          </div>
        </CardContent>
        <CardFooter className="border-t pt-4 text-sm text-muted-foreground">
          All nodes are running normally
        </CardFooter>
      </Card>
    </div>
  );
}

interface NodeStatusProps {
  name: string;
  status: "online" | "offline" | "warning";
  load: number;
  storage: number;
}

function NodeStatus({ name, status, load, storage }: NodeStatusProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center">
          <Server className="h-4 w-4 mr-2 text-muted-foreground" />
          <span className="font-medium">{name}</span>
        </div>
        <div className="flex items-center">
          <div
            className={`h-2 w-2 rounded-full mr-2 ${
              status === "online" ? "bg-green-500" : status === "warning" ? "bg-yellow-500" : "bg-red-500"
            }`}
          />
          <span className="text-xs capitalize">{status}</span>
        </div>
      </div>
      <div className="space-y-1">
        <div className="flex items-center justify-between text-xs">
          <span>CPU Load</span>
          <span>{load}%</span>
        </div>
        <Progress value={load} className="h-1" />
      </div>
      <div className="space-y-1">
        <div className="flex items-center justify-between text-xs">
          <span>Storage</span>
          <span>{storage}%</span>
        </div>
        <Progress value={storage} className="h-1" />
      </div>
    </div>
  );
}
