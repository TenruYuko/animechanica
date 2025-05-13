"use client"
import React from "react"

const ANILIST_CLIENT_ID_KEY = "anilist_client_id"

export default function AnilistLoginPage() {
  // Optionally, you could allow the user to enter their own client ID here
  // For now, prompt for it if missing
  const ANILIST_CLIENT_ID = "26797";

  const handleLogin = () => {
    const redirectUri = `${window.location.origin}/auth/callback`;
    const url = `https://anilist.co/api/v2/oauth/authorize?client_id=${encodeURIComponent(ANILIST_CLIENT_ID)}&response_type=code&redirect_uri=${encodeURIComponent(redirectUri)}`;
    console.log('AniList OAuth URL:', url);
    const popup = window.open(url, "anilist_oauth", "width=500,height=700");
    if (popup) {
      const listener = (event: MessageEvent) => {
        if (event.origin === window.location.origin && event.data === "ANILIST_AUTH_SUCCESS") {
          window.removeEventListener("message", listener);
          popup.close();
          window.location.reload(); // or update UI as needed
        }
      };
      window.addEventListener("message", listener);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center min-h-screen">
      <div className="bg-white dark:bg-gray-900 p-8 rounded shadow-md w-full max-w-md">
        <h2 className="text-2xl font-bold mb-6 text-center">Login with AniList</h2>
        <button
          className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          onClick={handleLogin}
        >
          Login with AniList
        </button>
        <p className="text-xs text-gray-500 mt-4">You can get your Client ID by registering an app at <a href="https://anilist.co/settings/developer" target="_blank" rel="noopener noreferrer" className="underline">AniList Developer Settings</a>. Please make sure to store your Client ID in local storage before logging in.</p>
        <p className="text-xs text-gray-500 mt-4">You can get your Client ID by registering an app at <a href="https://anilist.co/settings/developer" target="_blank" rel="noopener noreferrer" className="underline">AniList Developer Settings</a>.</p>
      </div>
    </div>
  );
}
