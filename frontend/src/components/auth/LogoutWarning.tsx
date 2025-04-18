
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

interface LogoutWarningProps {
  onCancel: () => void;
}

export function LogoutWarning({ onCancel }: LogoutWarningProps) {
  const [countdown, setCountdown] = useState(5);
  
  // Function that will be called before logout
  const performLogoutTasks = () => {
    // Add any cleanup or state saving tasks here
    console.log("Performing pre-logout tasks...");
    
    // For now, just show a toast message
    toast.info("Logging out...");
    
    // This will be replaced with actual logout logic
    window.location.reload();
  };
  
  useEffect(() => {
    // Start countdown
    const timer = setInterval(() => {
      setCountdown((prevCount) => {
        if (prevCount <= 1) {
          clearInterval(timer);
          performLogoutTasks();
          return 0;
        }
        return prevCount - 1;
      });
    }, 1000);
    
    // Clean up timer on unmount
    return () => clearInterval(timer);
  }, []);
  
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Semi-transparent overlay */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onCancel}></div>
      
      {/* Warning message */}
      <div className="relative bg-background p-6 rounded-lg shadow-lg border-2 border-destructive max-w-md w-full mx-4">
        <h2 className="text-xl font-bold mb-4">Automatic Logout</h2>
        <p className="mb-6">
          You left the seat and the system interface will log out of the session automatically in <span className="text-destructive font-bold">{countdown}</span> seconds.
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={performLogoutTasks}>
            Logout Now
          </Button>
        </div>
      </div>
    </div>
  );
}
