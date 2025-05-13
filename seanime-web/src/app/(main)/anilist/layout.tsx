"use client"
import { CustomBackgroundImage } from "@/app/(main)/_features/custom-ui/custom-background-image"
import React from "react"

import { AuthGuard } from "@/app/(main)/AuthGuard"

export default function Layout({ children }: { children: React.ReactNode }) {
    return (
        <AuthGuard>
            {/*[CUSTOM UI]*/}
            <CustomBackgroundImage />
            {children}
        </AuthGuard>
    )
}

export const dynamic = "force-static"
