import { Form } from "@/components/ui/form"
import { logger } from "@/lib/helpers/debug"
import { ANILIST_OAUTH_URL, COOKIE_OPTIONS } from "@/lib/server/config"
import axios from "axios"
import { deleteCookie, getCookie, setCookie } from "cookies-next"
import Link from "next/link"
import { useEffect, useState } from "react"
import { useLogin } from "@/api/hooks/auth.hooks"
import { isPlatformMirrorMode } from "@/lib/server/config.client"
import { useAtomValue } from "jotai/react"
import { useRouter } from "next/navigation"
import React from "react"
import { toast } from "sonner"
import { LoadingOverlay } from "@/components/ui/loading-spinner"
import { websocketConnectedAtom } from "@/app/websocket-provider"

type CallbackPageProps = {}

/**
 * @description
 * - Logs the user in using the AniList token present in the URL hash
 */
export function CallbackPage(props: CallbackPageProps) {
    const router = useRouter()
    const {} = props

    const websocketConnected = useAtomValue(websocketConnectedAtom)

    const { mutate: login } = useLogin()

    const called = React.useRef(false)

    const [error, setError] = React.useState<string | null>(null)
    const [retrying, setRetrying] = React.useState(false)

    // Debug function to show more information about what's happening
    const debugLogin = (message: string, data?: any) => {
        console.log(`[Auth Debug] ${message}`, data || '');
    }

    React.useEffect(() => {
        if (typeof window === "undefined") {
            debugLogin('Window is undefined');
            return;
        }
        
        // Don't wait for websocket connection - proceed anyway
        if (!websocketConnected) {
            debugLogin('Websocket not connected, but proceeding anyway');
        }

        debugLogin('Processing login, URL hash:', window.location.hash);
        
        // Check if we already have the session cookie
        const getCookieValue = (name: string) => {
            const value = `; ${document.cookie}`
            const parts = value.split(`; ${name}=`)
            if (parts.length === 2) return parts.pop()?.split(";").shift() || ""
            return ""
        }
        
        const sessionCookie = getCookieValue("Seanime-Anilist-Session");
        if (sessionCookie) {
            debugLogin('Session cookie already exists, redirecting to home');
            // We already have a session cookie, just redirect to home
            setTimeout(() => {
                window.location.href = '/';
            }, 500);
            return;
        }
        
        // Try to get token from URL hash or localStorage
        let _token = '';
        
        // First check URL hash
        if (window.location.hash && window.location.hash.includes('access_token')) {
            _token = window.location.hash.replace("#access_token=", "").replace(/&.*/, "");
            debugLogin('Token extracted from URL:', _token ? 'Found token' : 'No token found');
        }
        
        // If no token in URL, check localStorage as fallback
        if (!_token) {
            const storedToken = localStorage.getItem('anilist_token');
            if (storedToken) {
                _token = storedToken;
                debugLogin('Using token from localStorage');
            }
        }
        
        if (_token && !called.current) {
            debugLogin('Attempting login with token');
            setError(null);
            called.current = true; // Mark as called to prevent duplicate logins
            
            // Store token in localStorage for persistence
            localStorage.setItem('anilist_token', _token);
            
            // Clean up the URL by removing the hash
            if (window.history && window.history.replaceState) {
                window.history.replaceState({}, document.title, window.location.pathname);
            }
            
            // Check if we're in mirror mode (connecting to a different host)
            const isMirrorMode = process.env.NEXT_PUBLIC_PLATFORM === 'mirror';
            debugLogin('Mirror mode:', isMirrorMode ? 'true' : 'false');
            
            if (isMirrorMode) {
                // In mirror mode, we need to call the API to properly authenticate
                debugLogin('Using API login in mirror mode with token length:', _token.length);
                
                // Show detailed token information (truncated for security)
                console.log('Token first 10 chars:', _token.substring(0, 10) + '...');
                
                // Try to clean the token in case there are any formatting issues
                const cleanedToken = _token.trim().replace(/[\n\r\t]/g, '');
                
                // Make sure we have the correct Content-Type for the API call
                document.cookie = `Seanime-Anilist-Session=${cleanedToken}; path=/; max-age=31536000`;
                
                // Make the login API call
                login({
                    token: cleanedToken,
                }, {
                    onSuccess: (data) => {
                        debugLogin('Login API call successful', data);
                        // Redirect to home page
                        setTimeout(() => {
                            window.location.href = '/';
                        }, 1000);
                    },
                    onError: (error) => {
                        debugLogin('Login API call failed', error);
                        // Provide more detailed error information
                        const errorMsg = error?.message || 'Unknown error';
                        const statusCode = error?.response?.status;
                        console.error(`Login error (${statusCode}):`, errorMsg, error);
                        
                        setError(`Failed to log in with AniList token (${statusCode}): ${errorMsg}`);
                        setRetrying(false);
                    }
                });
            } else {
                // In standard mode, set the cookie directly
                debugLogin('Using direct cookie in standard mode');
                document.cookie = `Seanime-Anilist-Session=${_token}; path=/; max-age=31536000`;
                
                // Redirect to home page
                setTimeout(() => {
                    window.location.href = '/';
                }, 1000);
            }
        } else if (!_token && !called.current) {
            debugLogin('No token found');
            setError("Invalid or missing AniList token. Please try logging in again.");
        }
    }, [websocketConnected, retrying])

    return (
        <div>
            <LoadingOverlay className="fixed w-full h-full z-[80]">
                {error ? (
                    <div className="flex flex-col items-center justify-center min-h-[40vh] space-y-4">
                        <h3 className="text-xl font-semibold text-red-500">{error}</h3>
                        <button
                            className="px-4 py-2 rounded bg-indigo-600 text-white hover:bg-indigo-700"
                            onClick={() => {
                                setRetrying(r => !r)
                                setError(null)
                                window.location.href = "/auth" // Go back to login
                            }}
                        >
                            Retry Login
                        </button>
                    </div>
                ) : (
                    <h3 className="mt-2">{websocketConnected ? "Authenticating..." : "Connecting..."}</h3>
                )}
            </LoadingOverlay>
        </div>
    )
}
