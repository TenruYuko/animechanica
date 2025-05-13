import { useEffect, useState } from "react";

/**
 * useSessionAnilistId
 * React hook to get/set AniList ID in sessionStorage
 */
export function useSessionAnilistId(): [string | null, (id: string) => void] {
  const [anilistId, setAnilistIdState] = useState<string | null>(null);

  useEffect(() => {
    if (typeof window !== "undefined") {
      setAnilistIdState(sessionStorage.getItem("anilist_id"));
    }
  }, []);

  const setAnilistId = (id: string) => {
    sessionStorage.setItem("anilist_id", id);
    setAnilistIdState(id);
  };

  return [anilistId, setAnilistId];
}
