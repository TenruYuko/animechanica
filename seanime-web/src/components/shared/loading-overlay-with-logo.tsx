import { TextGenerateEffect } from "@/components/shared/text-generate-effect"
import { Button } from "@/components/ui/button"
import { LoadingOverlay } from "@/components/ui/loading-spinner"
import Image from "next/image"
import React from "react"

export function LoadingOverlayWithLogo({ refetch, title }: { refetch?: () => void, title?: string }) {
    // Use a consistent default title for server rendering to prevent hydration mismatch
    const defaultTitle = "S e a n i m e";
    
    // Use client-side only rendering for components that might cause hydration issues
    const [isMounted, setIsMounted] = React.useState(false);
    
    React.useEffect(() => {
        setIsMounted(true);
    }, []);
    
    return <LoadingOverlay showSpinner={false}>
        <Image
            src="/logo_2.png"
            alt="Loading..."
            priority
            width={180}
            height={180}
            className="animate-pulse"
        />
        
        {/* Always render with default title during server rendering to prevent hydration mismatch */}
        <TextGenerateEffect 
            className="text-lg mt-2 text-[--muted] animate-pulse" 
            words={isMounted ? (title ?? defaultTitle) : defaultTitle} 
        />

        {isMounted && process.env.NEXT_PUBLIC_PLATFORM === "desktop" && !!refetch && (
            <Button
                onClick={() => window.location.reload()}
                className="mt-4"
                intent="gray-outline"
                size="sm"
            >Reload</Button>
        )}
    </LoadingOverlay>
}