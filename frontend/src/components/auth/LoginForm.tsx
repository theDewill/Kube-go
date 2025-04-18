
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Camera, Key, LockKeyhole, Mail, UserCircle2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { toast } from "sonner";

export function LoginForm({ onLogin }: { onLogin: () => void }) {
  const [isLoading, setIsLoading] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    
    // Simulate login
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

  const handleFacialLogin = () => {
    setIsLoading(true);
    
    // Simulate facial recognition
    setTimeout(() => {
      setIsLoading(false);
      toast.success("Facial recognition successful");
      onLogin();
    }, 2000);
  };

  return (
    <Card className="w-full max-w-md mx-auto">
      <CardHeader className="space-y-1 text-center">
        <CardTitle className="text-2xl font-bold">Drive Space</CardTitle>
        <CardDescription>
          On-premise storage solution
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="credentials">
          <TabsList className="grid w-full grid-cols-2 mb-4">
            <TabsTrigger value="credentials">
              <Key className="h-4 w-4 mr-2" />
              Credentials
            </TabsTrigger>
            <TabsTrigger value="facial">
              <Camera className="h-4 w-4 mr-2" />
              Facial Recognition
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
              <div className="w-32 h-32 mx-auto rounded-full border-2 border-dashed border-muted-foreground flex items-center justify-center">
                <UserCircle2 className="h-16 w-16 text-muted-foreground" />
              </div>
              <p className="text-sm text-muted-foreground">
                Position your face in front of the camera to login
              </p>
              <Button onClick={handleFacialLogin} className="w-full" disabled={isLoading}>
                {isLoading ? "Scanning..." : "Start Scan"}
              </Button>
            </div>
          </TabsContent>
        </Tabs>
      </CardContent>
      <CardFooter className="flex flex-col space-y-4">
        <div className="text-xs text-muted-foreground text-center">
          This is a demo application. Any credentials will work.
        </div>
      </CardFooter>
    </Card>
  );
}
