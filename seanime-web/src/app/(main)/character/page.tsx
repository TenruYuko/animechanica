"use client"
import { useRouter, useSearchParams } from "next/navigation"
import React from "react"
import { CharacterDetailsPage } from "./_containers"

export const dynamic = "force-static"

export default function Page() {
    const router = useRouter()
    const searchParams = useSearchParams()
    const characterId = searchParams.get("id")

    React.useEffect(() => {
        if (!characterId) {
            router.push("/")
        }
    }, [characterId, router])

    if (!characterId) return null

    return <CharacterDetailsPage characterId={characterId} />
}
