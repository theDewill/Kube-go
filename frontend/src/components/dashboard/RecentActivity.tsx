
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { FileEdit, FileUp, FolderOpen, Share2, Snowflake, UserPlus } from "lucide-react";

const activities = [
  {
    id: 1,
    user: {
      name: "Mark Wilson",
      avatar: "/placeholder.svg",
      initials: "MW",
    },
    action: "uploaded",
    target: "quarterly-report.docx",
    time: "2 minutes ago",
    icon: FileUp,
  },
  {
    id: 2,
    user: {
      name: "Sarah Chen",
      avatar: "/placeholder.svg",
      initials: "SC",
    },
    action: "refrigerated",
    target: "marketing-videos folder",
    time: "15 minutes ago",
    icon: Snowflake,
  },
  {
    id: 3,
    user: {
      name: "Alex Johnson",
      avatar: "/placeholder.svg",
      initials: "AJ",
    },
    action: "shared",
    target: "project-budget.xlsx",
    time: "1 hour ago",
    icon: Share2,
  },
  {
    id: 4,
    user: {
      name: "Jamie Lee",
      avatar: "/placeholder.svg",
      initials: "JL",
    },
    action: "edited",
    target: "meeting-notes.md",
    time: "3 hours ago",
    icon: FileEdit,
  },
  {
    id: 5,
    user: {
      name: "System",
      avatar: "/placeholder.svg",
      initials: "SY",
    },
    action: "added",
    target: "Chris Evans as a new user",
    time: "5 hours ago",
    icon: UserPlus,
  },
  {
    id: 6,
    user: {
      name: "Taylor Swift",
      avatar: "/placeholder.svg",
      initials: "TS",
    },
    action: "opened",
    target: "archived-projects folder",
    time: "1 day ago",
    icon: FolderOpen,
  },
];

export function RecentActivity() {
  return (
    <Card className="col-span-3 md:col-span-1">
      <CardHeader>
        <CardTitle className="text-base font-medium">Recent Activity</CardTitle>
        <CardDescription>
          Latest actions across the system
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[400px]">
          <div className="space-y-4">
            {activities.map((activity) => {
              const Icon = activity.icon;
              return (
                <div key={activity.id} className="flex items-start space-x-4">
                  <Avatar className="h-8 w-8">
                    <AvatarImage src={activity.user.avatar} alt={activity.user.name} />
                    <AvatarFallback>{activity.user.initials}</AvatarFallback>
                  </Avatar>
                  <div className="space-y-1">
                    <p className="text-sm">
                      <span className="font-medium">{activity.user.name}</span>{" "}
                      <span className="text-muted-foreground">{activity.action}</span>{" "}
                      <span className="font-medium">{activity.target}</span>
                    </p>
                    <div className="flex items-center text-xs text-muted-foreground">
                      <Icon className="mr-1 h-3 w-3" />
                      <span>{activity.time}</span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
