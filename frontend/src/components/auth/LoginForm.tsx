import { useState, useRef, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Camera,
  Key,
  LockKeyhole,
  Mail,
  UserPlus,
  Eye,
  EyeOff,
  AlertCircle,
  RefreshCw,
} from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { toast } from "sonner";
import { useAuth } from "@/contexts/AuthContext";

export function LoginForm() {
  const { login, loginWithFace, isLoading } = useAuth();

  // Form states
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  // Camera states
  const [cameraStatus, setCameraStatus] = useState<"inactive" | "requesting" | "active" | "error">(
    "inactive",
  );
  const [cameraError, setCameraError] = useState<string>("");
  const [permissionStatus, setPermissionStatus] = useState<"unknown" | "granted" | "denied">(
    "unknown",
  );

  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const isRequestingRef = useRef(false);

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

  const stopCamera = useCallback(() => {
    console.log("🛑 Stopping camera");

    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => {
        track.stop();
        console.log(`Stopped ${track.kind} track`);
      });
      streamRef.current = null;
    }

    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }

    setCameraStatus("inactive");
    setCameraError("");
    isRequestingRef.current = false;
  }, []);

  const activateCamera = useCallback(async () => {
    if (isRequestingRef.current) {
      console.log("🚫 Camera request already in progress");
      return;
    }

    isRequestingRef.current = true;
    setCameraStatus("requesting");
    setCameraError("");
    setPermissionStatus("unknown");

    console.log("🎥 Requesting camera activation");

    try {
      // Check if getUserMedia is supported
      if (!navigator.mediaDevices?.getUserMedia) {
        throw new Error("Camera not supported in this browser");
      }

      // Check for existing permissions first
      if (navigator.permissions) {
        try {
          const permissionResult = await navigator.permissions.query({
            name: "camera" as PermissionName,
          });
          console.log("📋 Current camera permission:", permissionResult.state);

          if (permissionResult.state === "denied") {
            setPermissionStatus("denied");
            throw new Error(
              "Camera permission was previously denied. Please enable it in your browser settings.",
            );
          }
        } catch (permError) {
          console.log("⚠️ Could not check permissions:", permError);
        }
      }

      console.log("📷 Requesting camera access...");

      // Simple, reliable constraints
      const stream = await navigator.mediaDevices.getUserMedia({
        video: {
          facingMode: "user",
          width: { ideal: 640 },
          height: { ideal: 480 },
        },
        audio: false,
      });

      console.log("✅ Camera stream obtained");
      setPermissionStatus("granted");
      streamRef.current = stream;

      // Wait for video element to be available
      if (!videoRef.current) {
        await new Promise((resolve) => setTimeout(resolve, 100));
      }

      if (!videoRef.current) {
        throw new Error("Video element not available");
      }

      // Set up video element
      const video = videoRef.current;
      console.log("📺 Setting up video element");

      // Create a promise to handle video setup
      await new Promise<void>((resolve, reject) => {
        const handleSuccess = () => {
          console.log("🎬 Video setup completed successfully");
          setCameraStatus("active");
          toast.success("Camera activated successfully!");
          resolve();
        };

        const handleError = (error: any) => {
          console.error("❌ Video setup failed:", error);
          reject(new Error(`Video setup failed: ${error.message}`));
        };

        try {
          // Set video properties
          video.muted = true;
          video.playsInline = true;
          video.autoplay = true;
          video.controls = false;

          // Set up event handlers
          const onCanPlay = () => {
            console.log("▶️ Video can play");
            video
              .play()
              .then(() => {
                console.log("🎵 Video play successful");
                handleSuccess();
              })
              .catch((playError) => {
                console.log("⚠️ Video play failed, but continuing:", playError);
                handleSuccess();
              });
          };

          const onError = (event: any) => {
            handleError(new Error("Video element error"));
          };

          // Set up event listeners
          video.addEventListener("canplay", onCanPlay, { once: true });
          video.addEventListener("error", onError, { once: true });

          // Set the stream
          video.srcObject = stream;

          // Timeout fallback
          setTimeout(() => {
            if (video.readyState >= 2) {
              console.log("⏰ Video setup timeout, but video seems ready");
              video.removeEventListener("canplay", onCanPlay);
              video.removeEventListener("error", onError);
              handleSuccess();
            } else {
              handleError(new Error("Video setup timeout"));
            }
          }, 5000);
        } catch (setupError) {
          handleError(setupError);
        }
      });
    } catch (error: any) {
      console.error("❌ Camera activation error:", error);

      let errorMessage = "Failed to activate camera";

      if (error.name === "NotAllowedError") {
        errorMessage = "Camera permission denied. Please allow camera access and try again.";
        setPermissionStatus("denied");
      } else if (error.name === "NotFoundError") {
        errorMessage = "No camera found on this device.";
      } else if (error.name === "NotReadableError") {
        errorMessage = "Camera is being used by another application.";
      } else if (error.name === "OverconstrainedError") {
        errorMessage = "Camera doesn't support the required settings.";
      } else if (error.name === "SecurityError") {
        errorMessage = "Camera access blocked for security reasons.";
      } else {
        errorMessage = error.message || "Unknown camera error";
      }

      setCameraStatus("error");
      setCameraError(errorMessage);
      toast.error(errorMessage);
    } finally {
      isRequestingRef.current = false;
    }
  }, []);

  const captureFrame = useCallback(() => {
    if (!videoRef.current || cameraStatus !== "active") {
      return null;
    }

    const video = videoRef.current;
    const canvas = document.createElement("canvas");
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;

    const ctx = canvas.getContext("2d");
    if (!ctx) return null;

    ctx.drawImage(video, 0, 0);

    // Convert to base64
    const dataURL = canvas.toDataURL("image/jpeg", 0.8);
    return dataURL.split(",")[1]; // Remove data:image/jpeg;base64, prefix
  }, [cameraStatus]);

  const handleFacialLogin = async () => {
    if (cameraStatus !== "active") {
      toast.error("Please activate the camera first");
      return;
    }

    try {
      // Capture frame from video
      const frameData = captureFrame();
      if (!frameData) {
        toast.error("Failed to capture frame from camera");
        return;
      }

      // Visual feedback - flash effect
      if (videoRef.current) {
        videoRef.current.style.filter = "brightness(1.5)";
        setTimeout(() => {
          if (videoRef.current) {
            videoRef.current.style.filter = "none";
          }
        }, 200);
      }

      // Call the updated loginWithFace with frame data
      await loginWithFace(frameData);
      toast.success("Welcome back!");
      stopCamera();
    } catch (error: any) {
      toast.error(error.message || "Face not recognized");
    }
  };

  // Handle tab changes to manage camera
  const handleTabChange = (value: string) => {
    setActiveTab(value);
    if (value !== "facial") {
      stopCamera();
    }
  };

  // Clean up camera on unmount
  useEffect(() => {
    return () => {
      stopCamera();
    };
  }, [stopCamera]);

  const renderCameraSection = () => {
    return (
      <div className="space-y-4">
        {/* Camera Preview Area */}
        <div className="mx-auto w-64 h-48 rounded-lg border-2 border-dashed border-muted-foreground overflow-hidden bg-black relative">
          {/* Always render the video element, but control its visibility */}
          <video
            ref={videoRef}
            autoPlay
            playsInline
            muted
            controls={false}
            className={`w-full h-full object-cover ${
              cameraStatus === "active" ? "block" : "hidden"
            }`}
            style={{ transform: "scaleX(-1)" }}
          />

          {/* Overlay content based on status */}
          {cameraStatus === "requesting" && (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <RefreshCw className="h-8 w-8 animate-spin text-blue-400 mx-auto mb-2" />
                <p className="text-sm text-gray-400">Requesting camera access...</p>
                <p className="text-xs text-gray-500 mt-1">Please allow camera permission</p>
              </div>
            </div>
          )}

          {cameraStatus === "error" && (
            <div className="absolute inset-0 flex items-center justify-center p-4">
              <div className="text-center">
                <AlertCircle className="h-8 w-8 text-red-400 mx-auto mb-2" />
                <p className="text-sm text-red-400">Camera Error</p>
              </div>
            </div>
          )}

          {cameraStatus === "inactive" && (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <Camera className="h-12 w-12 text-gray-400 mx-auto mb-2" />
                <p className="text-sm text-gray-400">Camera Inactive</p>
                <p className="text-xs text-gray-500 mt-1">Click button below to activate</p>
              </div>
            </div>
          )}

          {/* Status indicator */}
          <div className="absolute top-2 right-2">
            <div
              className={`w-3 h-3 rounded-full ${
                cameraStatus === "active"
                  ? "bg-green-500"
                  : cameraStatus === "requesting"
                    ? "bg-blue-500 animate-pulse"
                    : cameraStatus === "error"
                      ? "bg-red-500"
                      : "bg-gray-500"
              }`}
            />
          </div>
        </div>

        {/* Camera Status and Controls */}
        <div className="text-center space-y-3">
          {cameraStatus === "error" && (
            <Alert className="mb-3">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{cameraError}</AlertDescription>
            </Alert>
          )}

          {cameraStatus === "inactive" && (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">
                Position your face in the frame for facial login
              </p>
              <Button onClick={activateCamera} className="w-full" variant="outline">
                <Camera className="h-4 w-4 mr-2" />
                Enable Camera
              </Button>
            </div>
          )}

          {cameraStatus === "requesting" && (
            <div className="space-y-2">
              <p className="text-sm text-blue-600 font-medium">Requesting camera access...</p>
              <p className="text-xs text-muted-foreground">
                {permissionStatus === "denied"
                  ? "Camera permission was denied. Please check your browser settings."
                  : "Please allow camera access when prompted by your browser"}
              </p>
            </div>
          )}

          {cameraStatus === "active" && (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">
                Position your face in the frame and click to login
              </p>
              <div className="flex gap-2 justify-center">
                <Button onClick={handleFacialLogin} className="flex-1" disabled={isLoading}>
                  {isLoading ? "Scanning..." : "Login with Face"}
                </Button>
                <Button
                  onClick={() => {
                    stopCamera();
                    setTimeout(activateCamera, 300);
                  }}
                  variant="outline"
                  size="sm"
                  disabled={isLoading}
                >
                  <RefreshCw className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}

          {cameraStatus === "error" && (
            <div className="space-y-2">
              <Button onClick={activateCamera} className="w-full" variant="outline">
                <Camera className="h-4 w-4 mr-2" />
                Try Again
              </Button>
              {permissionStatus === "denied" && (
                <p className="text-xs text-muted-foreground">
                  Please enable camera access in your browser settings and refresh the page.
                </p>
              )}
            </div>
          )}
        </div>
      </div>
    );
  };

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
              {renderCameraSection()}
            </TabsContent>
          </Tabs>
        </CardContent>

        <CardFooter>
          <p className="text-xs text-muted-foreground text-center w-full">
            {activeTab === "facial"
              ? "Your face data is processed locally and securely"
              : "Your credentials are securely encrypted and stored"}
          </p>
        </CardFooter>
      </Card>
    </div>
  );
}
