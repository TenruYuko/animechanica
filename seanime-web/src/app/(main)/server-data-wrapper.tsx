import { useGetStatus } from "@/api/hooks/status.hooks"
import { GettingStartedPage } from "@/app/(main)/_features/getting-started/getting-started-page"
import { useServerStatus, useSetServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { LoadingOverlayWithLogo } from "@/components/shared/loading-overlay-with-logo"
import { LuffyError } from "@/components/shared/luffy-error"
import { AppLayoutStack } from "@/components/ui/app-layout"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { defineSchema, Field, Form } from "@/components/ui/form"
import { logger } from "@/lib/helpers/debug"
import { ANILIST_OAUTH_URL, ANILIST_PIN_URL } from "@/lib/server/config"
import { WSEvents } from "@/lib/server/ws-events"
import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import React, { useState, useEffect } from "react"
import { useWebsocketMessageListener } from "./_hooks/handle-websockets"

type ServerDataWrapperProps = {
    host: string
    children?: React.ReactNode
}

export function ServerDataWrapper(props: ServerDataWrapperProps) {

    const {
        host,
        children,
        ...rest
    } = props

    const pathname = usePathname()
    const router = useRouter()
    const serverStatus = useServerStatus()
    const setServerStatus = useSetServerStatus()
    const { data: _serverStatus, isLoading, refetch } = useGetStatus()
    
    // Check for session token and redirect to /auth page if not present
    const [hasSessionToken, setHasSessionToken] = useState<boolean>(false)
    
    useEffect(() => {
        // Skip token check for auth-related paths
        if (pathname.startsWith("/auth")) {
            return
        }
        
        const getCookieValue = (name: string) => {
            const value = `; ${document.cookie}`
            const parts = value.split(`; ${name}=`)
            if (parts.length === 2) return parts.pop()?.split(";").shift() || ""
            return ""
        }
        
        // Check for the correct cookie name "Seanime-Anilist-Session"
        const sessionId = getCookieValue("Seanime-Anilist-Session")
        setHasSessionToken(!!sessionId)
        
        // Redirect to auth page if no session token is found
        if (!sessionId && !pathname.startsWith("/auth")) {
            router.push("/auth")
        }
    }, [pathname, router])

    React.useEffect(() => {
        if (_serverStatus) {
            // logger("SERVER").info("Server status", _serverStatus)
            setServerStatus(_serverStatus)
        }
    }, [_serverStatus])

    useWebsocketMessageListener({
        type: WSEvents.ANILIST_DATA_LOADED,
        onMessage: () => {
            logger("Data Wrapper").info("Anilist data loaded, refetching server status")
            refetch()
        },
    })

    // Refetch the server status every 2 seconds if serverReady is false
    // This is a fallback to the websocket
    const intervalId = React.useRef<NodeJS.Timeout | null>(null)
    React.useEffect(() => {
        if (!serverStatus?.serverReady) {
            intervalId.current = setInterval(() => {
                logger("Data Wrapper").info("Refetching server status")
                refetch()
            }, 2000)
        }
        return () => {
            logger("Data Wrapper").info("Clearing interval")
            if (intervalId.current) {
                clearInterval(intervalId.current)
                intervalId.current = null
            }
        }
    }, [serverStatus?.serverReady])

    // Check if we have the session cookie directly
    const hasCookie = React.useMemo(() => {
        if (typeof window === "undefined") return false;
        
        const getCookieValue = (name: string) => {
            const value = `; ${document.cookie}`
            const parts = value.split(`; ${name}=`)
            if (parts.length === 2) return parts.pop()?.split(";").shift() || ""
            return ""
        }
        
        return !!getCookieValue("Seanime-Anilist-Session");
    }, []);
    
    /**
     * If the server status is loading or doesn't exist, show the loading overlay
     * But if we have the cookie, we'll proceed anyway after a timeout
     */
    const [bypassLoading, setBypassLoading] = React.useState(false);
    
    React.useEffect(() => {
        if ((isLoading || !serverStatus) && hasCookie) {
            // If we have the cookie but server status is loading, wait a bit then bypass
            const timer = setTimeout(() => {
                setBypassLoading(true);
            }, 5000); // Wait 5 seconds then bypass
            
            return () => clearTimeout(timer);
        }
    }, [isLoading, serverStatus, hasCookie]);
    
    if ((isLoading || !serverStatus) && !bypassLoading) return <LoadingOverlayWithLogo />
    if (!serverStatus?.serverReady && !bypassLoading) return <LoadingOverlayWithLogo title="L o a d i n g" />

    /**
     * If the pathname is /auth/callback, show the callback page
     */
    if (pathname.startsWith("/auth/callback")) return children

    /**
     * If the server status doesn't have settings, show the getting started page
     */
    if (!serverStatus?.settings) {
        // Only show getting started page if we have serverStatus
        // If serverStatus is undefined but we're bypassing loading, render children
        if (serverStatus) {
            return <GettingStartedPage status={serverStatus} />
        } else if (bypassLoading) {
            // If we're bypassing loading and have no serverStatus, just render children
            return children;
        }
    }

    /**
     * If the app is updating, show a different screen
     */
    if (serverStatus?.updating) {
        return <div className="container max-w-3xl py-10">
            <div className="mb-4 flex justify-center w-full">
                <img src="/logo_2.png" alt="logo" className="w-36 h-auto" />
            </div>
            <p className="text-center text-lg">
                Seanime is currently updating. Refresh the page once the update is complete and the connection has been reestablished.
            </p>
        </div>
    }

    /**
     * Check feature flag routes
     */

    if (!serverStatus?.mediastreamSettings?.transcodeEnabled && pathname.startsWith("/mediastream")) {
        return <LuffyError title="Transcoding not enabled" />
    }

    if (!serverStatus?.user && host === "127.0.0.1:43211" && process.env.NEXT_PUBLIC_PLATFORM !== "desktop") {
        return <div className="container max-w-3xl py-10">
            <Card className="md:py-10">
                <AppLayoutStack>
                    <div className="text-center space-y-4">
                        <div className="mb-4 flex justify-center w-full">
                            <img src="/logo.png" alt="logo" className="w-24 h-auto" />
                        </div>
                        <h3>Welcome!</h3>
                        <Button
                            onClick={() => {
                                const url = serverStatus?.anilistClientId
                                    ? `https://anilist.co/api/v2/oauth/authorize?client_id=${serverStatus?.anilistClientId}&response_type=token`
                                    : ANILIST_OAUTH_URL
                                window.open(url, "_self")
                            }}
                            leftIcon={<svg
                                xmlns="http://www.w3.org/2000/svg" fill="currentColor" width="24" height="24"
                                viewBox="0 0 24 24" role="img"
                            >
                                <path
                                    d="M6.361 2.943 0 21.056h4.942l1.077-3.133H11.4l1.052 3.133H22.9c.71 0 1.1-.392 1.1-1.101V17.53c0-.71-.39-1.101-1.1-1.101h-6.483V4.045c0-.71-.392-1.102-1.101-1.102h-2.422c-.71 0-1.101.392-1.101 1.102v1.064l-.758-2.166zm2.324 5.948 1.688 5.018H7.144z"
                                />
                            </svg>}
                            intent="primary"
                            size="xl"
                        >Log in with AniList</Button>
                    </div>
                </AppLayoutStack>
            </Card>
        </div>
    } else if (!serverStatus?.user) {
        // Redirect to the auth page for AniList token input
        useEffect(() => {
            router.push('/auth');
        }, []);
        
        // Show loading while redirecting
        return <LoadingOverlayWithLogo title="Redirecting to login..." />
    }

    return children
}
