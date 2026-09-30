import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  useRef,
  useCallback,
  type ReactNode,
} from "react";
import { apiUrl } from "../lib/urls";
import { apiJSON, ApiError } from "../lib/api";
import type { User } from "../lib/userAccess";

interface UserContextType {
  user: User | null;
  isLoading: boolean;
  authError: boolean;
  login: (userData: User) => void;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;
}

const UserContext = createContext<UserContextType | undefined>(undefined);

export const UserProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [authError, setAuthError] = useState(false);
  const revision = useRef(0);

  const login = (userData: User) => {
    revision.current++;
    setAuthError(false);
    setIsLoading(false);
    setUser(userData);
  };

  const logout = async () => {
    const current = ++revision.current;
    await apiJSON(apiUrl("/api/logout"), { method: "POST" });
    if (current !== revision.current) return;
    setUser(null);
    setAuthError(false);
    setIsLoading(false);
  };

  const checkAuth = useCallback(async () => {
    const current = ++revision.current;
    setIsLoading(true);
    try {
      const data = await apiJSON<User>(apiUrl("/api/me"));
      if (current !== revision.current) return;
      setUser(data);
      setAuthError(false);
    } catch (error) {
      if (current !== revision.current) return;
      if (error instanceof ApiError && error.status === 401) {
        setUser(null);
        setAuthError(false);
      } else setAuthError(true);
    } finally {
      if (current === revision.current) setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    const state = revision;
    void checkAuth();
    return () => {
      state.current++;
    };
  }, [checkAuth]);

  return (
    <UserContext.Provider
      value={{ user, isLoading, authError, login, logout, checkAuth }}
    >
      {children}
    </UserContext.Provider>
  );
};

export const useUser = () => {
  const context = useContext(UserContext);
  if (context === undefined) {
    throw new Error("useUser must be used within a UserProvider");
  }
  return context;
};
