"use client"
import React, { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { AppLayoutStack } from "@/components/ui/app-layout"
import { defineSchema, Field, Form } from "@/components/ui/form"
import { ANILIST_PIN_URL } from "@/lib/server/config"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { useLogin } from '@/api/hooks/auth.hooks';

export default function AuthPage() {
  const router = useRouter()
  const serverStatus = useServerStatus()
  const [showDirectInput, setShowDirectInput] = useState(false);
  const [token, setToken] = useState('');
  const { mutate: login } = useLogin();
  const [isLoading, setIsLoading] = useState(false);

  // Debug function to show more information about what's happening
  const debugAuth = (message: string, data?: any) => {
    console.log(`[Auth Page Debug] ${message}`, data || '');
  }

  // Clear any existing tokens when the auth page loads
  // This ensures a clean state after logout
  useEffect(() => {
    debugAuth('Auth page loaded');
    
    // Clear any stored tokens or session data only if we're not in the middle of a login flow
    if (!window.location.hash || !window.location.hash.includes('access_token')) {
      debugAuth('Clearing existing tokens');
      document.cookie = 'Seanime-Anilist-Session=; Max-Age=0; path=/; domain=' + window.location.hostname;
      localStorage.removeItem('anilist_token');
    }
    
    // If we have a hash in the URL (from a callback), process it
    if (window.location.hash && window.location.hash.includes('access_token')) {
      debugAuth('Found access_token in URL hash');
      // Extract the token
      const token = window.location.hash.replace("#access_token=", "").replace(/&.*/, "")
      if (token) {
        debugAuth('Token extracted, redirecting to callback handler');
        // Store token and redirect to callback handler
        localStorage.setItem('anilist_token', token);
        // Use a small timeout to ensure the token is stored before redirecting
        setTimeout(() => {
          window.location.href = "/auth/callback";
        }, 100);
      }
    }
  }, [])

  return (
    <div className="container max-w-3xl py-10">
      <Card className="md:py-10">
        <AppLayoutStack>
          <div className="text-center space-y-4">
            <div className="mb-4 flex justify-center w-full">
              <img src="/logo.png" alt="logo" className="w-24 h-auto" />
            </div>
            <h3>Welcome to Seanime!</h3>
            
            {/* AniList token input section */}
            <Link
              href={ANILIST_PIN_URL}
              target="_blank"
            >
              <Button
                leftIcon={<svg
                  xmlns="http://www.w3.org/2000/svg" fill="currentColor" width="24" height="24"
                  viewBox="0 0 24 24" role="img"
                >
                  <path
                    d="M6.361 2.943 0 21.056h4.942l1.077-3.133H11.4l1.052 3.133H22.9c.71 0 1.1-.392 1.1-1.101V17.53c0-.71-.39-1.101-1.1-1.101h-6.483V4.045c0-.71-.392-1.102-1.101-1.102h-2.422c-.71 0-1.101.392-1.101 1.102v1.064l-.758-2.166zm2.324 5.948 1.688 5.018H7.144z"
                  />
                </svg>}
                intent="white"
                size="md"
              >Get AniList token</Button>
            </Link>

            <Form
              schema={defineSchema(({ z }) => z.object({
                token: z.string().min(1, "Token is required"),
              }))}
              onSubmit={data => {
                router.push("/auth/callback#access_token=" + data.token.trim())
              }}
            >
              <Field.Textarea
                name="token"
                label="Enter the token"
                fieldClass="px-4"
              />
              <div className="flex gap-2 justify-center">
                <Field.Submit showLoadingOverlayOnSuccess>Continue with Redirect</Field.Submit>
                
                <Button
                  type="button"
                  intent="primary"
                  disabled={isLoading}
                  onClick={() => {
                    // Get token from the textarea
                    const tokenField = document.querySelector('textarea[name="token"]') as HTMLTextAreaElement;
                    if (!tokenField || !tokenField.value.trim()) {
                      alert('Please enter a valid token');
                      return;
                    }
                    
                    const token = tokenField.value.trim();
                    debugAuth('Direct login with token length:', token.length);
                    setIsLoading(true);
                    
                    // Direct login without redirect
                    // Store token in cookie
                    document.cookie = `Seanime-Anilist-Session=${token}; path=/; max-age=31536000`;
                    
                    // Call login API
                    login({ token }, {
                      onSuccess: (data) => {
                        debugAuth('Direct login successful', data);
                        // Force redirect to home
                        window.location.href = '/';
                      },
                      onError: (error) => {
                        debugAuth('Direct login failed', error);
                        setIsLoading(false);
                        alert('Login failed: ' + (error?.message || 'Unknown error'));
                      }
                    });
                  }}
                >
                  {isLoading ? 'Logging in...' : 'Direct Login'}
                </Button>
              </div>
            </Form>
          </div>
        </AppLayoutStack>
      </Card>
    </div>
  )
}
