import { useState, useEffect, useRef } from "react";
import { AppLayout } from "@/components/layout/AppLayout";
import {
  Settings as SettingsIcon,
  Save,
  RefreshCw,
  Eye,
  EyeOff,
  CheckCircle,
  AlertCircle,
  Info,
  ExternalLink,
  Zap,
  Globe,
  Mail,
  Camera,
  Users2,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { toast } from "sonner";
import { FileBrowserAPI } from "@/lib/file-api";
import { TrainNewUser } from "@/../wailsjs/go/security/FacialSystem";

interface AppSettings {
  gemini_api_key: string;
  ollama_url: string;
  ollama_model: string;
}

interface ConnectionStatus {
  ollama: boolean;
  gemini: boolean;
  checking: boolean;
}

export default function Settings() {
  const [settings, setSettings] = useState<AppSettings>({
    gemini_api_key: "",
    ollama_url: "http://localhost:11434",
    ollama_model: "phi3:mini",
  });
  const [originalSettings, setOriginalSettings] = useState<AppSettings>({
    gemini_api_key: "",
    ollama_url: "http://localhost:11434",
    ollama_model: "phi3:mini",
  });
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [showApiKey, setShowApiKey] = useState(false);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [registerEmail, setRegisterEmail] = useState("");
  const [cameraActive, setCameraActive] = useState(false);
  const videoRef = useRef(null);
  const streamRef = useRef(null);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>({
    ollama: false,
    gemini: false,
    checking: false,
  });

  // Load settings from file
  useEffect(() => {
    loadSettings();
  }, []);

  // Check for unsaved changes
  useEffect(() => {
    const hasChanges = JSON.stringify(settings) !== JSON.stringify(originalSettings);
    setHasUnsavedChanges(hasChanges);
  }, [settings, originalSettings]);

  // Replace the loadSettings function:
  const loadSettings = async () => {
    setIsLoading(true);
    try {
      // Use the dedicated settings file method
      const blob = await FileBrowserAPI.downloadSettingsFile();
      const text = await blob.text();
      const loadedSettings = JSON.parse(text);

      setSettings(loadedSettings);
      setOriginalSettings(loadedSettings);

      // Check connection status after loading
      checkConnections(loadedSettings);
    } catch (error) {
      console.error("Failed to load settings:", error);
      toast.error("Failed to load settings. Using default values.");
      // If file doesn't exist, we'll create it with default values when saving
    } finally {
      setIsLoading(false);
    }
  };

  // Replace the saveSettings function:
  const saveSettings = async () => {
    setIsSaving(true);
    try {
      // Convert settings to JSON
      const jsonString = JSON.stringify(settings, null, 2);
      const buffer = new TextEncoder().encode(jsonString);

      // Use the dedicated settings file method to overwrite
      await FileBrowserAPI.uploadSettingsFile(buffer);

      setOriginalSettings(settings);
      toast.success("Settings saved successfully");

      // Recheck connections after saving
      checkConnections(settings);
    } catch (error) {
      console.error("Failed to save settings:", error);
      toast.error("Failed to save settings. Please try again.");
    } finally {
      setIsSaving(false);
    }
  };

  const resetSettings = () => {
    setSettings(originalSettings);
    toast.info("Settings reset to last saved values");
  };

  const checkConnections = async (settingsToCheck: AppSettings) => {
    setConnectionStatus((prev) => ({ ...prev, checking: true }));

    try {
      // Check Ollama connection
      const ollamaStatus = await checkOllamaConnection(settingsToCheck.ollama_url);

      // Check Gemini connection (only if API key is provided)
      const geminiStatus = settingsToCheck.gemini_api_key
        ? await checkGeminiConnection(settingsToCheck.gemini_api_key)
        : false;

      setConnectionStatus({
        ollama: ollamaStatus,
        gemini: geminiStatus,
        checking: false,
      });
    } catch (error) {
      console.error("Error checking connections:", error);
      setConnectionStatus({
        ollama: false,
        gemini: false,
        checking: false,
      });
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

  const handleRegister = async () => {
    if (!registerEmail) {
      toast.error("Please enter an email address for registration");
      return;
    }

    setIsLoading(true);

    try {
      // Call the Go method for user registration with facial training
      const userId = await TrainNewUser(registerEmail);

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

  const checkOllamaConnection = async (url: string): Promise<boolean> => {
    try {
      const response = await fetch(`${url}/api/tags`, {
        method: "GET",
        signal: AbortSignal.timeout(5000), // 5 second timeout
      });
      return response.ok;
    } catch {
      return false;
    }
  };

  const checkGeminiConnection = async (apiKey: string): Promise<boolean> => {
    try {
      const response = await fetch(
        `https://generativelanguage.googleapis.com/v1beta/models?key=${apiKey}`,
        {
          method: "GET",
          signal: AbortSignal.timeout(5000), // 5 second timeout
        },
      );
      return response.ok;
    } catch {
      return false;
    }
  };

  const handleInputChange = (field: keyof AppSettings, value: string) => {
    setSettings((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const getConnectionStatusBadge = (status: boolean, checking: boolean, label: string) => {
    if (checking) {
      return (
        <Badge variant="outline" className="flex items-center space-x-1">
          <RefreshCw className="h-3 w-3 animate-spin" />
          <span>Checking {label}...</span>
        </Badge>
      );
    }

    return (
      <Badge
        variant={status ? "default" : "secondary"}
        className={
          status ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" : ""
        }
      >
        <div className="flex items-center space-x-1">
          {status ? <CheckCircle className="h-3 w-3" /> : <AlertCircle className="h-3 w-3" />}
          <span>{status ? "Connected" : "Disconnected"}</span>
        </div>
      </Badge>
    );
  };

  if (isLoading) {
    return (
      <div className="flex flex-col h-full items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        <span className="mt-4 text-muted-foreground">Loading settings...</span>
      </div>
    );
  }

  return (
    <AppLayout>
      <div className="flex flex-col h-full p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <SettingsIcon className="h-6 w-6 text-primary" />
            <h1 className="text-2xl font-bold">Kube Settings</h1>
          </div>

          <div className="flex items-center space-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => checkConnections(settings)}
              disabled={connectionStatus.checking}
            >
              <RefreshCw
                className={`h-4 w-4 mr-2 ${connectionStatus.checking ? "animate-spin" : ""}`}
              />
              Test Connections
            </Button>
            {hasUnsavedChanges && (
              <Badge variant="outline" className="text-orange-600">
                <AlertCircle className="h-3 w-3 mr-1" />
                Unsaved Changes
              </Badge>
            )}
          </div>
        </div>

        {/* Action Buttons */}
        {hasUnsavedChanges && (
          <Alert>
            <Info className="h-4 w-4" />
            <AlertDescription className="flex items-center justify-between">
              <span>You have unsaved changes to your settings.</span>
              <div className="flex space-x-2">
                <Button size="sm" onClick={saveSettings} disabled={isSaving}>
                  {isSaving ? (
                    <>
                      <RefreshCw className="h-3 w-3 mr-1 animate-spin" />
                      Saving...
                    </>
                  ) : (
                    <>
                      <Save className="h-3 w-3 mr-1" />
                      Save
                    </>
                  )}
                </Button>
                <Button size="sm" variant="outline" onClick={resetSettings}>
                  Reset
                </Button>
              </div>
            </AlertDescription>
          </Alert>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 flex-1">
          {/* Ollama Settings */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center space-x-2">
                  <Zap className="h-5 w-5 text-blue-500" />
                  <span>Local Settings (LAN)</span>
                </CardTitle>
                {getConnectionStatusBadge(
                  connectionStatus.ollama,
                  connectionStatus.checking,
                  "Ollama",
                )}
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="ollama-url">Ollama Server URL</Label>
                <Input
                  id="ollama-url"
                  value={settings.ollama_url}
                  onChange={(e) => handleInputChange("ollama_url", e.target.value)}
                  placeholder="http://localhost:11434"
                />
                <p className="text-sm text-muted-foreground">
                  The URL where your Ollama server is running. Default is localhost:11434.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="ollama-model">Ollama Model</Label>
                <Input
                  id="ollama-model"
                  value={settings.ollama_model}
                  onChange={(e) => handleInputChange("ollama_model", e.target.value)}
                  placeholder="phi3:mini"
                />
                <p className="text-sm text-muted-foreground">
                  The Ollama model to use for content analysis. Popular options: phi3:mini,
                  llama3.1:8b
                </p>
              </div>

              <div className="pt-4 border-t">
                <div className="flex items-center space-x-2 text-sm text-muted-foreground">
                  <Info className="h-4 w-4" />
                  <span>Local AI processing for maximum privacy and offline capability.</span>
                </div>
                <Button
                  variant="link"
                  size="sm"
                  className="p-0 h-auto"
                  onClick={() => window.open("https://ollama.ai/download", "_blank")}
                >
                  <ExternalLink className="h-3 w-3 mr-1" />
                  Download Ollama
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Gemini Settings */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center space-x-2">
                  <Globe className="h-5 w-5 text-green-500" />
                  <span>Google Gemini Settings</span>
                </CardTitle>
                {getConnectionStatusBadge(
                  connectionStatus.gemini,
                  connectionStatus.checking,
                  "Gemini",
                )}
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="gemini-key">Gemini API Key</Label>
                <div className="relative">
                  <Input
                    id="gemini-key"
                    type={showApiKey ? "text" : "password"}
                    value={settings.gemini_api_key}
                    onChange={(e) => handleInputChange("gemini_api_key", e.target.value)}
                    placeholder="Enter your Gemini API key"
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    className="absolute right-0 top-0 h-full px-3"
                    onClick={() => setShowApiKey(!showApiKey)}
                  >
                    {showApiKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </Button>
                </div>
                <p className="text-sm text-muted-foreground">
                  Your Google Gemini API key for cloud-based AI analysis.
                </p>
              </div>

              <div className="pt-4 border-t">
                <div className="flex items-center space-x-2 text-sm text-muted-foreground">
                  <Info className="h-4 w-4" />
                  <span>Advanced cloud AI with superior content understanding.</span>
                </div>
                <Button
                  variant="link"
                  size="sm"
                  className="p-0 h-auto"
                  onClick={() => window.open("https://aistudio.google.com/app/apikey", "_blank")}
                >
                  <ExternalLink className="h-3 w-3 mr-1" />
                  Get Gemini API Key
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>

        <div>
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex flex-col">
                  <div className="flex items-center space-x-2">
                    <Users2 className="h-5 w-5 text-yellow-500" />
                    <p className="text-2xl">Kube Transfer</p>
                  </div>
                  <p className="text-sm mt-2 text-neutral-500">
                    Transfer the kube space of this node instance to a new user
                  </p>
                </CardTitle>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
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
            </CardContent>
          </Card>
        </div>

        {/* Information Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-6">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center space-x-2 mb-2">
                <CheckCircle className="h-5 w-5 text-green-500" />
                <h3 className="font-medium">Privacy</h3>
              </div>
              <p className="text-sm text-muted-foreground">
                Ollama processes files locally on your machine, ensuring complete data privacy.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-4">
              <div className="flex items-center space-x-2 mb-2">
                <Zap className="h-5 w-5 text-blue-500" />
                <h3 className="font-medium">Performance</h3>
              </div>
              <p className="text-sm text-muted-foreground">
                Both AI providers offer different performance characteristics. Test to find your
                preference.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-4">
              <div className="flex items-center space-x-2 mb-2">
                <Globe className="h-5 w-5 text-green-500" />
                <h3 className="font-medium">Quality</h3>
              </div>
              <p className="text-sm text-muted-foreground">
                Gemini typically provides more sophisticated content analysis and understanding.
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Save Button (Fixed at bottom) */}
        <div className="flex justify-end space-x-2 pt-4 border-t">
          <Button
            variant="outline"
            onClick={resetSettings}
            disabled={!hasUnsavedChanges || isSaving}
          >
            Reset
          </Button>
          <Button onClick={saveSettings} disabled={!hasUnsavedChanges || isSaving}>
            {isSaving ? (
              <>
                <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Save className="h-4 w-4 mr-2" />
                Save Settings
              </>
            )}
          </Button>
        </div>
      </div>
    </AppLayout>
  );
}
