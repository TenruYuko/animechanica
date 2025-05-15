import React, { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { formatDistanceToNow } from "date-fns"
import { Badge } from "@/components/ui/badge"
import { SessionAccount, useListSessions, useLogin, useLogout } from "@/api/hooks/auth.hooks"
import { useRouter } from "next/navigation"
import { toast } from "sonner"

export function SessionManager() {
  const router = useRouter()
  const { mutate: login } = useLogin()
  const { mutate: logout } = useLogout()
  const [currentSessionId, setCurrentSessionId] = useState<string>("")

  // Fetch the list of active sessions
  const { data: sessions, isLoading, refetch } = useListSessions()

  // Get the current session ID from the cookie
  useEffect(() => {
    const getCookieValue = (name: string) => {
      const value = `; ${document.cookie}`
      const parts = value.split(`; ${name}=`)
      if (parts.length === 2) return parts.pop()?.split(";").shift() || ""
      return ""
    }
    
    // Use the correct cookie name "Seanime-Anilist-Session"
    const sessionId = getCookieValue("Seanime-Anilist-Session")
    setCurrentSessionId(sessionId)
  }, [])

  // Handle logout
  const handleLogout = () => {
    logout(undefined, {
      onError: () => {
        toast.error("Failed to log out")
      },
    })
  }

  // Format the last active time
  const formatLastActive = (timestamp: number) => {
    const date = new Date(timestamp * 1000)
    return formatDistanceToNow(date, { addSuffix: true })
  }

  // Check if this is the current session
  const isCurrentSession = (sessionId: string) => {
    return sessionId === currentSessionId
  }

  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle>Active Sessions</CardTitle>
        <CardDescription>
          Manage your active sessions across different browsers and devices
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex justify-center p-4">Loading sessions...</div>
        ) : sessions && sessions.length > 0 ? (
          <div className="space-y-4">
            {sessions.map((session: SessionAccount) => (
              <div 
                key={session.sessionId} 
                className={`p-4 border rounded-lg ${isCurrentSession(session.sessionId) ? 'border-primary bg-primary/5' : 'border-border'}`}
              >
                <div className="flex justify-between items-start">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-medium">{session.username}</h3>
                      {isCurrentSession(session.sessionId) && (
                        <Badge className="bg-primary/10 border border-primary/30">Current Session</Badge>
                      )}
                    </div>
                    <div className="text-sm text-muted-foreground mt-1">
                      Last active: {formatLastActive(session.lastActive)}
                    </div>
                    <div className="text-xs text-muted-foreground mt-1">
                      Session ID: {session.sessionId.substring(0, 8)}...
                    </div>
                  </div>
                  {isCurrentSession(session.sessionId) && (
                    <Button 
                      className="bg-red-600 hover:bg-red-700 text-white"
                      size="sm"
                      onClick={handleLogout}
                    >
                      Log Out
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center p-4 text-muted-foreground">
            No active sessions found
          </div>
        )}
      </CardContent>
      <CardFooter className="flex justify-between">
        <Button className="border border-primary/30 bg-transparent" onClick={() => refetch()}>
          Refresh
        </Button>
        <Button className="bg-red-600 hover:bg-red-700 text-white" onClick={handleLogout}>
          Log Out Current Session
        </Button>
      </CardFooter>
    </Card>
  )
}
