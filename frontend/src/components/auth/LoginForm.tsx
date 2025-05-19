import { useState, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Camera, Key, LockKeyhole, Mail, UserPlus, Eye, EyeOff } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { toast } from "sonner";
import { useAuth } from "@/contexts/AuthContext";

export function LoginForm() {
  const { login, loginWithFace, isLoading } = useAuth();

  // Form states
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  // Camera states
  const [cameraActive, setCameraActive] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);

  // Tab state
  const [activeTab, setActiveTab] = useState("credentials");

  const handleCredentialLogin = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!email || !password) {
      toast.error("Please enter both email and password");
      return;
    }

    try {
      await login(email, password);
      toast.success("Login successful!");
    } catch (error: any) {
      toast.error(error.message || "Login failed");
    }
  };

  const startCamera = async () => {
    try {
      // Stop any previous stream
      if (streamRef.current) {
        streamRef.current.getTracks().forEach((track) => track.stop());
      }

      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "user" },
        audio: false,
      });

      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        streamRef.current = stream;
        setCameraActive(true);
      }
    } catch (error) {
      console.error("Error accessing camera:", error);
      toast.error("Could not access camera. Please check permissions.");
      setCameraActive(false);
    }
  };

  const stopCamera = () => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
      setCameraActive(false);
      if (videoRef.current) {
        videoRef.current.srcObject = null;
      }
    }
  };

  const handleFacialLogin = async () => {
    if (!cameraActive) {
      toast.error("Please enable the camera first");
      return;
    }

    try {
      await loginWithFace();
      toast.success("Welcome back!");
      stopCamera();
    } catch (error: any) {
      toast.error(error.message || "Face not recognized");
    }
  };

  // Handle tab changes to manage camera
  const handleTabChange = (value: string) => {
    setActiveTab(value);
    if (value === "facial") {
      startCamera();
    } else {
      stopCamera();
    }
  };

  // Clean up camera on unmount
  useEffect(() => {
    return () => {
      stopCamera();
    };
  }, []);

  // Auto-start camera if facial tab is selected on mount
  useEffect(() => {
    if (activeTab === "facial") {
      startCamera();
    }
  }, []);

  return (
    <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-gray-900 dark:to-gray-800">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-1 text-center">
          <CardTitle className="text-2xl font-bold">Kube Login</CardTitle>
          <CardDescription>Secure access to your distributed storage</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs value={activeTab} onValueChange={handleTabChange} className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="credentials" className="flex items-center gap-2">
                <Key className="h-4 w-4" />
                Credentials
              </TabsTrigger>
              <TabsTrigger value="facial" className="flex items-center gap-2">
                <Camera className="h-4 w-4" />
                Facial Login
              </TabsTrigger>
            </TabsList>

            <TabsContent value="credentials" className="space-y-4 mt-4">
              <form onSubmit={handleCredentialLogin} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <div className="relative">
                    <Mail className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                    <Input
                      id="email"
                      type="email"
                      placeholder="name@example.com"
                      className="pl-10"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      disabled={isLoading}
                      required
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="password">Password</Label>
                  <div className="relative">
                    <LockKeyhole className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                    <Input
                      id="password"
                      type={showPassword ? "text" : "password"}
                      className="pl-10 pr-10"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      disabled={isLoading}
                      required
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                      onClick={() => setShowPassword(!showPassword)}
                      disabled={isLoading}
                    >
                      {showPassword ? (
                        <EyeOff className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <Eye className="h-4 w-4 text-muted-foreground" />
                      )}
                    </Button>
                  </div>
                </div>

                <Button type="submit" className="w-full" disabled={isLoading}>
                  {isLoading ? "Signing in..." : "Sign In"}
                </Button>
              </form>
            </TabsContent>

            <TabsContent value="facial" className="space-y-4 mt-4">
              <div className="space-y-4">
                <div className="mx-auto w-64 h-48 rounded-lg border-2 border-dashed border-muted-foreground overflow-hidden">
                  {cameraActive ? (
                    <video
                      ref={videoRef}
                      autoPlay
                      playsInline
                      muted
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <div className="flex items-center justify-center h-full">
                      <Camera className="h-12 w-12 text-muted-foreground" />
                    </div>
                  )}
                </div>

                <p className="text-sm text-muted-foreground text-center">
                  {cameraActive
                    ? "Position your face in the frame and click to login"
                    : "Click the button below to enable your camera"}
                </p>

                {!cameraActive ? (
                  <Button onClick={startCamera} className="w-full" variant="outline">
                    <Camera className="h-4 w-4 mr-2" />
                    Enable Camera
                  </Button>
                ) : (
                  <Button onClick={handleFacialLogin} className="w-full" disabled={isLoading}>
                    {isLoading ? "Scanning..." : "Login with Face"}
                  </Button>
                )}
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>

        <CardFooter>
          <p className="text-xs text-muted-foreground text-center w-full">
            {activeTab === "facial"
              ? "Your face data is processed locally and never leaves your device"
              : "Your credentials are securely encrypted and stored"}
          </p>
        </CardFooter>
      </Card>
    </div>
  );
}
