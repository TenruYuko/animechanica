"use client"
import { useRouter } from "next/navigation";
import React from "react";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [checked, setChecked] = React.useState(false);

  React.useEffect(() => {
    // Only run on the client
    const token = localStorage.getItem("anilist_access_token");
    if (!token) {
      router.replace("/auth/login");
    } else {
      setChecked(true);
    }
  }, [router]);

  // Prevent rendering children until we've checked auth
  if (!checked) return null;
  return <>{children}</>;
}
