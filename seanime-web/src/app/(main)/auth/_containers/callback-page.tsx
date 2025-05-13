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
            /**
             * Get the AniList token from the URL hash
             */
            const _token = window?.location?.hash?.replace("#access_token=", "")?.replace(/&.*/, "")
            if (!!_token && !called.current) {
                // Store the token in localStorage for session-based access
                localStorage.setItem("anilist_access_token", _token)
                // Fetch AniList user profile and store ID in sessionStorage
                fetch('https://graphql.anilist.co', {
                  method: 'POST',
                  headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${_token}`,
                  },
                  body: JSON.stringify({
                    query: `query { Viewer { id } }`,
                  }),
                })
                  .then(res => res.json())
                  .then(data => {
                    const typed = data as { data?: { Viewer?: { id?: number } } }
                    const id = typed?.data?.Viewer?.id
                    if (id) {
                      sessionStorage.setItem('anilist_id', String(id))
                      toast.success('AniList authentication successful!')
                      // Notify opener window if present
                      if (window.opener && window.opener !== window) {
                        window.opener.postMessage('ANILIST_AUTH_SUCCESS', window.location.origin);
                      }
                    } else {
                      toast.error('Failed to retrieve AniList ID')
                    }
                    called.current = true
                    router.push('/')
                  })
                  .catch(() => {
                    toast.error('Failed to fetch AniList profile')
                    router.push('/')
                  })
            } else {
                toast.error("Invalid token")
                router.push("/")
            }
        }
    }, [websocketConnected])

    return (
        <div>
            <LoadingOverlay className="fixed w-full h-full z-[80]">
                <h3 className="mt-2">Authenticating...</h3>
            </LoadingOverlay>
        </div>
    )
}
