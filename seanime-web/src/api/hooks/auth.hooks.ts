import { useServerMutation, useServerQuery } from "@/api/client/requests"
import { Login_Variables } from "@/api/generated/endpoint.types"
import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { Status } from "@/api/generated/types"
import { useSetServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { useRouter } from "next/navigation"
import { toast } from "sonner"

export function useLogin() {
    const router = useRouter()
    const setServerStatus = useSetServerStatus()

    // Debug function to show more information about what's happening
    const debugLogin = (message: string, data?: any) => {
        console.log(`[Login Hook Debug] ${message}`, data || '');
    }

    return useServerMutation<Status, Login_Variables>({
        endpoint: API_ENDPOINTS.AUTH.Login.endpoint,
        method: API_ENDPOINTS.AUTH.Login.methods[0],
        mutationKey: [API_ENDPOINTS.AUTH.Login.key],
        onMutate: (variables) => {
            debugLogin('Login attempt with token', variables?.token ? 'Token provided' : 'No token');
        },
        onSuccess: async (data: any) => {
            debugLogin('Login response', data);
            if (data && typeof data === 'object' && data.status === "success") {
                toast.success("Successfully logged in")
                debugLogin('Setting server status and redirecting');
                // Set server status with the response data
                setServerStatus(data)
                
                // Redirect to home page after successful login
                setTimeout(() => {
                    router.push("/")
                }, 1000)
            } else {
                const errorMsg = data && typeof data === 'object' && 'message' in data ? 
                    String(data.message) : 'Unknown error';
                debugLogin('Login failed', errorMsg);
                toast.error(`Failed to login: ${errorMsg}`)
            }
        },
        onError: (error: any) => {
            const errorMsg = error && typeof error === 'object' && 'message' in error ? 
                String(error.message) : 'Unknown error';
            debugLogin('Login error', errorMsg);
            toast.error(`Failed to login: ${errorMsg}`)
        },
    })
}

export function useLogout() {
    const router = useRouter()
    
    return useServerMutation<Status>({
        endpoint: API_ENDPOINTS.AUTH.Logout.endpoint,
        method: API_ENDPOINTS.AUTH.Logout.methods[0],
        mutationKey: [API_ENDPOINTS.AUTH.Logout.key],
        onSuccess: async () => {
            toast.success("Successfully logged out")
            
            // Clear all auth-related cookies
            document.cookie = 'Seanime-Anilist-Session=; Max-Age=0; path=/; domain=' + window.location.hostname;
            localStorage.removeItem('anilist_token');
            
            // Force a page refresh and redirect to auth page
            setTimeout(() => {
                window.location.href = '/auth';
            }, 500);
        },
        onError: () => {
            toast.error("Failed to log out")
        },
    })
}

// Define the session account interface
export interface SessionAccount {
    id: number
    username: string
    sessionId: string
    lastActive: number
    isActive: boolean
    createdAt: string
    updatedAt: string
}

// Hook to list all active sessions
export function useListSessions() {
    return useServerQuery<SessionAccount[]>({
        endpoint: API_ENDPOINTS.AUTH.ListSessions.endpoint,
        method: API_ENDPOINTS.AUTH.ListSessions.methods[0],
        queryKey: [API_ENDPOINTS.AUTH.ListSessions.key],
    })
}
