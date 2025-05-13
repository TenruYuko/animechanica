"use client"
import React from "react"

const ANILIST_CLIENT_ID = "26797";

export default function AnilistLoginPage() {

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
    <div className="flex flex-col items-center justify-center min-h-screen bg-gradient-to-br from-gray-900 to-gray-800">
      <div className="bg-white dark:bg-gray-900 p-10 rounded-2xl shadow-xl w-full max-w-md flex flex-col items-center">
        {/* Seanime logo placeholder */}
        <div className="mb-8">
          <div className="w-16 h-16 rounded-full bg-gradient-to-br from-indigo-500 to-blue-500 flex items-center justify-center">
            <span className="text-2xl font-black text-white select-none">S</span>
          </div>
        </div>
        <h1 className="text-3xl font-extrabold mb-4 text-center tracking-tight">Sign in to Seanime</h1>
        <p className="text-gray-500 mb-8 text-center">Connect your AniList account to continue</p>
        <button
          className="w-full px-4 py-3 bg-gradient-to-r from-indigo-500 to-blue-600 text-white text-lg rounded-lg font-semibold shadow hover:from-indigo-600 hover:to-blue-700 transition"
          onClick={handleLogin}
        >
          Login with AniList
        </button>
        <p className="text-xs text-gray-400 mt-6 text-center">Get your Client ID at <a href="https://anilist.co/settings/developer" target="_blank" rel="noopener noreferrer" className="underline">AniList Developer Settings</a>.</p>
      </div>
    </div>
  );
}
