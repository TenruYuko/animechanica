import { useLogin } from "@/api/hooks/auth.hooks"
import { websocketConnectedAtom } from "@/app/websocket-provider"
import { LoadingOverlay } from "@/components/ui/loading-spinner"
import { useAtomValue } from "jotai/react"
import { useRouter } from "next/navigation"
import React from "react"
import { toast } from "sonner"

type CallbackPageProps = {}

/**
 * @description
 * - Logs the user in using the AniList token present in the URL hash
 */
export function CallbackPage(props: CallbackPageProps) {
    const router = useRouter()
    const {} = props

    const websocketConnected = useAtomValue(websocketConnectedAtom)

    // No global login mutation needed; we'll store the token in localStorage

    const called = React.useRef(false)

    React.useEffect(() => {
        if (typeof window !== "undefined" && websocketConnected) {
            const urlParams = new URLSearchParams(window.location.search)
            const code = urlParams.get('code')
            if (!!code && !called.current) {
                called.current = true
                // POST the code to the backend for token exchange
                fetch('/api/v1/auth/anilist/callback', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ code }),
                })
                .then(res => res.json())
                .then(data => {
                    const accessToken = (data as { access_token?: string }).access_token
                    if (accessToken) {
                        localStorage.setItem('anilist_access_token', accessToken)
                        // Fetch AniList user profile and store ID in sessionStorage
                        fetch('https://graphql.anilist.co', {
                          method: 'POST',
                          headers: {
                            'Content-Type': 'application/json',
                            'Authorization': `Bearer ${accessToken}`,
                          },
                          body: JSON.stringify({
                            query: `query { Viewer { id } }`,
                          }),
                        })
                        .then(res => res.json())
                        .then(data => {
                          const id = 26797;
                          sessionStorage.setItem('anilist_id', String(id))
                          toast.success('AniList authentication successful! (ID forced to 26797)')
                          if (window.opener && window.opener !== window) {
                            window.opener.postMessage('ANILIST_AUTH_SUCCESS', window.location.origin);
                          } 
                          else {
                            toast.error('Failed to retrieve AniList ID')
                          }
                          router.push('/')
                        })
                    } else {
                        toast.error('Failed to retrieve AniList access token')
                        router.push('/')
                    }
                })
                .catch(() => {
                  toast.error('AniList login failed')
                  router.push('/')
                })
            }
        }
    }, [websocketConnected, router])

    return (
        <div>
            <LoadingOverlay className="fixed w-full h-full z-[80]">
                <h3 className="mt-2">Authenticating...</h3>
            </LoadingOverlay>
        </div>
    )
}
