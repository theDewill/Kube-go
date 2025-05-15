import { useState, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Camera, Key, LockKeyhole, Mail, UserPlus } from "lucide-react";
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

export function LoginForm({ onLogin, facialSystem }) {
  const [isLoading, setIsLoading] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [registerEmail, setRegisterEmail] = useState("");
  const [cameraActive, setCameraActive] = useState(false);
  const videoRef = useRef(null);
  const streamRef = useRef(null);

  const handleLogin = (e) => {
    e.preventDefault();
    setIsLoading(true);

    // Simulate login for credential-based login
    setTimeout(() => {
      setIsLoading(false);
      if (email && password) {
        toast.success("Login successful");
        onLogin();
      } else {
        toast.error("Please enter both email and password");
      }
    }, 1500);
  };

  const startCamera = async () => {
    try {
      // Stop any previous stream
      if (streamRef.current) {
        streamRef.current.getTracks().forEach((track) => track.stop());
      }

      const stream = await navigator.mediaDevices.getUserMedia({
        video: true,
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
    setIsLoading(true);

    try {
      // Call the Go method for facial login
      const userEmail = await window.go.security.FacialSystem.LoginUser();

      setIsLoading(false);
      toast.success(`Welcome back, ${userEmail}`);
      stopCamera();
      onLogin();
    } catch (error) {
      setIsLoading(false);
      toast.error(`Authentication failed: ${error.message || "Face not recognized"}`);
    }
  };

  const handleRegister = async () => {
    if (!registerEmail) {
      toast.error("Please enter an email address for registration");
      return;
    }

    setIsLoading(true);

    try {
      // Call the Go method for user registration with facial training
      const userId = await window.go.security.FacialSystem.TrainNewUser(registerEmail);

      setIsLoading(false);
      toast.success("Registration successful! You can now login with facial recognition.");
      stopCamera();
      // Switch to facial login tab
      document.querySelector('[value="facial"]').click();
    } catch (error) {
      setIsLoading(false);
      toast.error(`Registration failed: ${error.message || "Could not register face"}`);
    }
  };

  // Handle tab changes to start/stop camera
  const handleTabChange = (value) => {
    if (value === "facial" || value === "register") {
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

  return (
    <Card className="w-full max-w-md mx-auto">
      <CardHeader className="space-y-1 text-center">
        <CardTitle className="text-2xl font-bold">Kube Login</CardTitle>
        <CardDescription>On-premise storage solution</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="credentials" onValueChange={handleTabChange}>
          <TabsList className="grid w-full grid-cols-3 mb-4">
            <TabsTrigger value="credentials">
              <Key className="h-4 w-4 mr-2" />
              Credentials
            </TabsTrigger>
            <TabsTrigger value="facial">
              <Camera className="h-4 w-4 mr-2" />
              Facial Login
            </TabsTrigger>
            <TabsTrigger value="register">
              <UserPlus className="h-4 w-4 mr-2" />
              Register Face
            </TabsTrigger>
          </TabsList>

          <TabsContent value="credentials">
            <form onSubmit={handleLogin} className="space-y-4">
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
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <div className="relative">
                  <LockKeyhole className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                  <Input
                    id="password"
                    type="password"
                    className="pl-10"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                </div>
              </div>
              <Button type="submit" className="w-full" disabled={isLoading}>
                {isLoading ? "Logging in..." : "Login"}
              </Button>
            </form>
          </TabsContent>

          <TabsContent value="facial">
            <div className="space-y-4 text-center">
              <div className="w-36 h-36 mx-auto rounded-full border-2 border-dashed border-muted-foreground overflow-hidden flex items-center justify-center">
                {cameraActive ? (
                  <video
                    ref={videoRef}
                    autoPlay
                    playsInline
                    muted
                    className="min-w-full min-h-full object-cover"
                  />
                ) : (
                  <div className="flex items-center justify-center h-full w-full">
                    <Camera className="h-12 w-12 text-muted-foreground" />
                  </div>
                )}
              </div>
              <p className="text-sm text-muted-foreground">
                Position your face in front of the camera to login
              </p>
              <Button
                onClick={handleFacialLogin}
                className="w-full bg-green-500 hover:bg-green-600"
                disabled={isLoading}
              >
                {isLoading ? "Scanning..." : "Login with Face"}
              </Button>
            </div>
          </TabsContent>

          <TabsContent value="register">
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="registerEmail">Email</Label>
                <div className="relative">
                  <Mail className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                  <Input
                    id="registerEmail"
                    type="email"
                    placeholder="name@example.com"
                    className="pl-10"
                    value={registerEmail}
                    onChange={(e) => setRegisterEmail(e.target.value)}
                  />
                </div>
              </div>

              <div className="w-36 h-36 mx-auto rounded-full border-2 border-dashed border-muted-foreground overflow-hidden flex items-center justify-center">
                {cameraActive ? (
                  <video
                    ref={videoRef}
                    autoPlay
                    playsInline
                    muted
                    className="min-w-full min-h-full object-cover"
                  />
                ) : (
                  <div className="flex items-center justify-center h-full w-full">
                    <Camera className="h-12 w-12 text-muted-foreground" />
                  </div>
                )}
              </div>
              <p className="text-sm text-muted-foreground text-center">
                We'll take multiple samples of your face to train the recognition system
              </p>
              <Button
                onClick={handleRegister}
                className="w-full bg-blue-500 hover:bg-blue-600"
                disabled={isLoading}
              >
                {isLoading ? "Registering..." : "Register Face"}
              </Button>
            </div>
          </TabsContent>
        </Tabs>
      </CardContent>
      <CardFooter className="flex flex-col space-y-4">
        <div className="text-xs text-muted-foreground text-center">
          Your face data is processed locally and never leaves your device
        </div>
      </CardFooter>
    </Card>
  );
}
