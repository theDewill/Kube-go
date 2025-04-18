
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Upload } from "lucide-react";

interface FileUploadProps {
  onUpload: () => void;
  isUploading: boolean;
}

export function FileUpload({ onUpload, isUploading }: FileUploadProps) {
  const [progress, setProgress] = useState(0);
  
  const handleUpload = () => {
    setProgress(0);
    onUpload();
    
    const interval = setInterval(() => {
      setProgress(prev => {
        if (prev >= 100) {
          clearInterval(interval);
          return 100;
        }
        return prev + 10;
      });
    }, 200);
    
    return () => clearInterval(interval);
  };
  
  return (
    <div className="flex items-center gap-2">
      <Button onClick={handleUpload} variant="outline" size="sm" disabled={isUploading}>
        <Upload className="h-4 w-4 mr-2" />
        Upload File
      </Button>
      
      {isUploading && (
        <div className="w-32">
          <Progress value={progress} className="h-2" />
        </div>
      )}
    </div>
  );
}
