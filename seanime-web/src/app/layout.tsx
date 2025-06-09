import { TauriManager } from "@/app/(main)/_tauri/tauri-manager"
import { ClientProviders } from "@/app/client-providers"
import type { Metadata } from "next"
// Font import removed to fix compilation issue
import "./globals.css"
import React from "react"

export const dynamic = "force-static"

// Using system fonts instead of Inter to fix compilation issue

export const metadata: Metadata = {
    title: "Seanime",
    description: "Self-hosted, user-friendly media server for anime and manga.",
    icons: {
        icon: "/icons/favicon.ico",
    },
}

export default function RootLayout({ children }: {
    children: React.ReactNode
}) {
    return (
        <html lang="en" suppressHydrationWarning>
        {/*<head>*/}
        {/*    {process.env.NODE_ENV === "development" && <script src="https://unpkg.com/react-scan/dist/auto.global.js" async></script>}*/}
        {/*</head>*/}
        <body className="font-sans" suppressHydrationWarning>
        {/*{process.env.NODE_ENV === "development" && <script src="http://localhost:8097"></script>}*/}
        <ClientProviders>
            {process.env.NEXT_PUBLIC_PLATFORM === "desktop" && <TauriManager />}
            {children}
        </ClientProviders>
        </body>
        </html>
    )
}


