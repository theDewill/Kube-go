import { useState, useRef, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Camera,
  Mail,
  LockKeyhole,
  UserPlus,
  Eye,
  EyeOff,
  AlertCircle,
  RefreshCw,
  Video,
} from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { toast } from "sonner";
import { useAuth } from "@/contexts/AuthContext";

export default function UserRegistration() {
  const { register, isLoading } = useAuth();

  // Form states
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [enableFacialAuth, setEnableFacialAuth] = useState(false); // Start with false
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  // Camera states
  const [cameraStatus, setCameraStatus] = useState<"inactive" | "requesting" | "active" | "error">(
    "inactive",
  );
  const [cameraError, setCameraError] = useState<string>("");
  const [permissionStatus, setPermissionStatus] = useState<"unknown" | "granted" | "denied">(
    "unknown",
  );

  // Refs
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const isRequestingRef = useRef(false);

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
          // Continue anyway, the getUserMedia call will handle it
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

      // Wait for video element to be available with retries
      let videoElement = videoRef.current;
      let attempts = 0;
      const maxAttempts = 10;

      while (!videoElement && attempts < maxAttempts) {
        console.log(`⏳ Waiting for video element... attempt ${attempts + 1}/${maxAttempts}`);
        await new Promise((resolve) => setTimeout(resolve, 200));
        videoElement = videoRef.current;
        attempts++;
      }

      if (!videoElement) {
        throw new Error("Video element not available after multiple attempts");
      }

      // Set up video element
      console.log("📺 Setting up video element");

      // Create a promise to handle video setup
      await new Promise<void>((resolve, reject) => {
        const video = videoElement!;

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
          const onLoadedMetadata = () => {
            console.log("📹 Video metadata loaded");
          };

          const onCanPlay = () => {
            console.log("▶️ Video can play");
            // Try to play the video
            video
              .play()
              .then(() => {
                console.log("🎵 Video play successful");
                handleSuccess();
              })
              .catch((playError) => {
                console.log("⚠️ Video play failed, but continuing:", playError);
                // Even if autoplay fails, we can still consider it successful
                handleSuccess();
              });
          };

          const onError = (event: any) => {
            handleError(new Error("Video element error"));
          };

          // Set up event listeners
          video.addEventListener("loadedmetadata", onLoadedMetadata, { once: true });
          video.addEventListener("canplay", onCanPlay, { once: true });
          video.addEventListener("error", onError, { once: true });

          // Set the stream
          video.srcObject = stream;

          // Timeout fallback
          setTimeout(() => {
            if (video.readyState >= 2) {
              console.log("⏰ Video setup timeout, but video seems ready");
              video.removeEventListener("loadedmetadata", onLoadedMetadata);
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

  // Add capturing state
  const [isCapturing, setIsCapturing] = useState(false);
  const [captureCount, setCaptureCount] = useState(0);

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

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!email) {
      toast.error("Email is required");
      return;
    }

    if (!password) {
      toast.error("Password is required");
      return;
    }

    if (password !== confirmPassword) {
      toast.error("Passwords do not match");
      return;
    }

    if (password.length < 8) {
      toast.error("Password must be at least 8 characters long");
      return;
    }

    if (enableFacialAuth && cameraStatus !== "active") {
      toast.error("Please activate camera for facial authentication");
      return;
    }

    try {
      // If facial auth is enabled, capture frames for training
      let faceFrames: string[] = [];
      if (enableFacialAuth && cameraStatus === "active") {
        setIsCapturing(true);
        setCaptureCount(0);

        toast.info("Starting facial data capture...");

        // Capture 5 frames with delays
        for (let i = 0; i < 5; i++) {
          setCaptureCount(i + 1);
          toast.info(`Capturing frame ${i + 1}/5 - Please look at the camera`);

          // Wait a moment for user to adjust
          await new Promise((resolve) => setTimeout(resolve, 1000));

          const frame = captureFrame();
          if (frame) {
            faceFrames.push(frame);
            // Visual feedback - flash effect
            if (videoRef.current) {
              videoRef.current.style.filter = "brightness(1.5)";
              setTimeout(() => {
                if (videoRef.current) {
                  videoRef.current.style.filter = "none";
                }
              }, 200);
            }
          }

          // Delay between captures
          await new Promise((resolve) => setTimeout(resolve, 500));
        }

        setIsCapturing(false);
        setCaptureCount(0);

        if (faceFrames.length === 0) {
          toast.error("Failed to capture facial data");
          return;
        }

        toast.success(`Captured ${faceFrames.length} frames for training`);
      }

      await register(email, password, enableFacialAuth, faceFrames);
      toast.success("Registration successful!");

      // Reset form
      setEmail("");
      setPassword("");
      setConfirmPassword("");
      stopCamera();
    } catch (error: any) {
      toast.error(error.message || "Registration failed");
    }
  };

  const handleFacialAuthToggle = (enabled: boolean) => {
    console.log("🔄 Facial auth toggled:", enabled);
    setEnableFacialAuth(enabled);

    if (!enabled) {
      stopCamera();
    }
    // Don't auto-start camera when enabled - let user click the button
  };

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      stopCamera();
    };
  }, [stopCamera]);

  const renderCameraSection = () => {
    if (!enableFacialAuth) return null;

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
            onLoadStart={() => console.log("📺 Video load started")}
            onCanPlay={() => console.log("▶️ Video can play event")}
            onPlaying={() => console.log("🎬 Video playing event")}
            onError={(e) => console.error("❌ Video error event:", e)}
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
                <Video className="h-12 w-12 text-gray-400 mx-auto mb-2" />
                <p className="text-sm text-gray-400">Camera Not Active</p>
                <p className="text-xs text-gray-500 mt-1">Click button below to activate</p>
              </div>
            </div>
          )}

          {/* Status indicator with more detailed states */}
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
              title={
                cameraStatus === "active"
                  ? "Camera Active"
                  : cameraStatus === "requesting"
                    ? "Requesting Access"
                    : cameraStatus === "error"
                      ? "Camera Error"
                      : "Camera Inactive"
              }
            />
          </div>
        </div>

        {/* Camera Controls */}
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
                Camera is required for facial authentication
              </p>
              <Button
                type="button"
                onClick={activateCamera}
                disabled={isLoading}
                className="bg-blue-600 hover:bg-blue-700"
              >
                <Camera className="h-4 w-4 mr-2" />
                Activate Camera
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
              {isCapturing ? (
                <div className="space-y-2">
                  <p className="text-sm text-blue-600 font-medium">
                    📸 Capturing facial data... {captureCount}/5
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Please look directly at the camera and stay still
                  </p>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                      style={{ width: `${(captureCount / 5) * 100}%` }}
                    />
                  </div>
                </div>
              ) : (
                <>
                  <p className="text-sm text-green-600 font-medium">✓ Camera is active and ready</p>
                  <div className="flex gap-2 justify-center">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        stopCamera();
                        setTimeout(activateCamera, 300);
                      }}
                      disabled={isLoading}
                    >
                      <RefreshCw className="h-4 w-4 mr-2" />
                      Restart Camera
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={stopCamera}
                      disabled={isLoading}
                    >
                      <Camera className="h-4 w-4 mr-2" />
                      Stop Camera
                    </Button>
                  </div>
                </>
              )}
            </div>
          )}

          {cameraStatus === "error" && (
            <div className="space-y-2">
              <Button type="button" onClick={activateCamera} disabled={isLoading} variant="outline">
                <Camera className="h-4 w-4 mr-2" />
                Try Again
              </Button>
              {permissionStatus === "denied" && (
                <p className="text-xs text-muted-foreground">
                  If permission was denied, you may need to enable camera access in your browser
                  settings and refresh the page.
                </p>
              )}
            </div>
          )}
        </div>
      </div>
    );
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center space-x-2">
          <UserPlus className="h-5 w-5 text-blue-500" />
          <span>Register New User</span>
        </CardTitle>
        <CardDescription>
          Create a new user account with optional facial authentication
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleRegister} className="space-y-4">
          {/* Email Field */}
          <div className="space-y-2">
            <Label htmlFor="register-email">Email</Label>
            <div className="relative">
              <Mail className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
              <Input
                id="register-email"
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

          {/* Password Field */}
          <div className="space-y-2">
            <Label htmlFor="register-password">Password</Label>
            <div className="relative">
              <LockKeyhole className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
              <Input
                id="register-password"
                type={showPassword ? "text" : "password"}
                className="pl-10 pr-10"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={isLoading}
                required
                minLength={8}
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

          {/* Confirm Password Field */}
          <div className="space-y-2">
            <Label htmlFor="confirm-password">Confirm Password</Label>
            <div className="relative">
              <LockKeyhole className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
              <Input
                id="confirm-password"
                type={showConfirmPassword ? "text" : "password"}
                className="pl-10 pr-10"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                disabled={isLoading}
                required
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                disabled={isLoading}
              >
                {showConfirmPassword ? (
                  <EyeOff className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <Eye className="h-4 w-4 text-muted-foreground" />
                )}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              Password must be at least 8 characters long
            </p>
          </div>

          {/* Facial Authentication Toggle */}
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="facial-auth">Enable Facial Authentication</Label>
                <p className="text-sm text-muted-foreground">
                  Allow login using facial recognition
                </p>
              </div>
              <Switch
                id="facial-auth"
                checked={enableFacialAuth}
                onCheckedChange={handleFacialAuthToggle}
                disabled={isLoading}
              />
            </div>

            {/* Camera Section */}
            {renderCameraSection()}
          </div>

          {/* Submit Button */}
          <Button
            type="submit"
            className="w-full"
            disabled={isLoading || (enableFacialAuth && cameraStatus !== "active")}
          >
            {isLoading ? "Creating Account..." : "Create Account"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
