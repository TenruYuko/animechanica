import React, { createContext, useContext, useEffect, useState } from "react";


const AnilistAuthContext = createContext<{ token: string | null }>({ token: null });

export function useAnilistAuth() {
  return useContext(AnilistAuthContext);
}

export function AnilistAuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(null);

  React.useEffect(() => {
    const stored = localStorage.getItem("anilist_token");
    if (stored) {
      setToken(stored);
    } else {
      // No token, redirect to login page
      if (typeof window !== "undefined" && window.location.pathname !== "/auth/login") {
        window.location.href = "/auth/login";
      }
    }
  }, []);

  return (
    <AnilistAuthContext.Provider value={{ token }}>
      {children}
    </AnilistAuthContext.Provider>
  );
}
