import React, { createContext, useContext, useState, useEffect, ReactNode } from "react";
import {
  GetCurrentUser,
  LoginWithCredentials,
  LoginWithFacialAuth,
  Logout,
  RegisterUser,
} from "@/../wailsjs/go/security/UserManager";
interface User {
  id: number;
  email: string;
  facial_data_id?: string;
  created_at: string;
  last_login_at?: string;
  is_active: boolean;
  has_facial_auth: boolean;
}

interface AuthResponse {
  success: boolean;
  user?: User;
  token?: string;
  message?: string;
  session_id?: string;
}

interface LoginRequest {
  email: string;
  password: string;
}

interface RegistrationRequest {
  email: string;
  password: string;
  enable_facial_auth: boolean;
}

interface AuthContextType {
  user: User | null;
  sessionId: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<boolean>;
  loginWithFace: () => Promise<boolean>;
  register: (email: string, password: string, enableFacialAuth: boolean) => Promise<boolean>;
  logout: () => Promise<void>;
  validateSession: () => Promise<boolean>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const isAuthenticated = !!user && !!sessionId;

  // Check for existing session on mount
  useEffect(() => {
    const savedSessionId = localStorage.getItem("session_id");
    if (savedSessionId) {
      setSessionId(savedSessionId);
      validateSession();
    } else {
      setIsLoading(false);
    }
  }, []);

  // Validate current session
  const validateSession = async (): Promise<boolean> => {
    const currentSessionId = sessionId || localStorage.getItem("session_id");
    if (!currentSessionId) {
      setIsLoading(false);
      return false;
    }

    try {
      const userData = await GetCurrentUser(currentSessionId);
      setUser(userData);
      setSessionId(currentSessionId);
      localStorage.setItem("session_id", currentSessionId);
      setIsLoading(false);
      return true;
    } catch (error) {
      console.error("Session validation failed:", error);
      // Clear invalid session
      setUser(null);
      setSessionId(null);
      localStorage.removeItem("session_id");
      setIsLoading(false);
      return false;
    }
  };

  // Login with email and password
  const login = async (email: string, password: string): Promise<boolean> => {
    setIsLoading(true);
    try {
      const response: AuthResponse = await LoginWithCredentials(email, password);

      if (response.success && response.user && response.session_id) {
        setUser(response.user);
        setSessionId(response.session_id);
        localStorage.setItem("session_id", response.session_id);
        setIsLoading(false);
        return true;
      } else {
        setIsLoading(false);
        throw new Error(response.message || "Login failed");
      }
    } catch (error) {
      setIsLoading(false);
      console.error("Login failed:", error);
      throw error;
    }
  };

  // Login with facial recognition
  const loginWithFace = async (): Promise<boolean> => {
    setIsLoading(true);
    try {
      const response: AuthResponse = await LoginWithFacialAuth();

      if (response.success && response.user && response.session_id) {
        setUser(response.user);
        setSessionId(response.session_id);
        localStorage.setItem("session_id", response.session_id);
        setIsLoading(false);
        return true;
      } else {
        setIsLoading(false);
        throw new Error(response.message || "Facial login failed");
      }
    } catch (error) {
      setIsLoading(false);
      console.error("Facial login failed:", error);
      throw error;
    }
  };

  // Register new user
  const register = async (
    email: string,
    password: string,
    enableFacialAuth: boolean,
  ): Promise<boolean> => {
    setIsLoading(true);
    try {
      const registrationData: RegistrationRequest = {
        email,
        password,
        enable_facial_auth: enableFacialAuth,
      };

      const response: AuthResponse = await RegisterUser(registrationData);

      if (response.success && response.user && response.session_id) {
        setUser(response.user);
        setSessionId(response.session_id);
        localStorage.setItem("session_id", response.session_id);
        setIsLoading(false);
        return true;
      } else {
        setIsLoading(false);
        throw new Error(response.message || "Registration failed");
      }
    } catch (error) {
      setIsLoading(false);
      console.error("Registration failed:", error);
      throw error;
    }
  };

  // Logout
  const logout = async (): Promise<void> => {
    try {
      if (sessionId) {
        await Logout(sessionId);
      }
    } catch (error) {
      console.error("Logout error:", error);
    } finally {
      setUser(null);
      setSessionId(null);
      localStorage.removeItem("session_id");
    }
  };

  const value: AuthContextType = {
    user,
    sessionId,
    isAuthenticated,
    isLoading,
    login,
    loginWithFace,
    register,
    logout,
    validateSession,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};

// Protected Route Component
interface ProtectedRouteProps {
  children: ReactNode;
  fallback?: ReactNode;
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children, fallback }) => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};
