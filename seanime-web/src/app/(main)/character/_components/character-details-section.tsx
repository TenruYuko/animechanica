import { AL_CharacterDetailsByID_Character } from "@/api/generated/types"
import {
    MediaPageHeader,
    MediaPageHeaderDetailsContainer,
    MediaPageHeaderEntryDetails,
} from "@/app/(main)/_features/media/_components/media-page-header-components"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/components/ui/core/styling"
import { Separator } from "@/components/ui/separator"
import { ThemeMediaPageInfoBoxSize, useThemeSettings } from "@/lib/theme/hooks"
import React from "react"
import { BiHeart, BiSolidHeart } from "react-icons/bi"
import { SiAnilist } from "react-icons/si"

type CharacterDetailsSectionProps = {
    character: AL_CharacterDetailsByID_Character
}

export function CharacterDetailsSection({ character }: CharacterDetailsSectionProps) {
    const ts = useThemeSettings()
    
    if (!character) return null

    const altNames = character.name?.alternative?.filter(Boolean) || []
    const birthday = character.dateOfBirth && (character.dateOfBirth.year || character.dateOfBirth.month || character.dateOfBirth.day)
        ? formatBirthday(character.dateOfBirth)
        : null

    const Details = () => (
        <div className="flex flex-col gap-4">
            {/* Character Stats */}
            <div className="flex flex-wrap gap-2">
                {character.gender && (
                    <Badge intent="primary" className="text-xs font-medium">
                        {character.gender}
                    </Badge>
                )}
                {character.age && (
                    <Badge intent="info" className="text-xs font-medium">
                        Age {character.age}
                    </Badge>
                )}
                {character.bloodType && (
                    <Badge intent="gray" className="text-xs font-medium">
                        Blood Type {character.bloodType}
                    </Badge>
                )}
                {birthday && (
                    <Badge intent="gray" className="text-xs font-medium">
                        Born {birthday}
                    </Badge>
                )}
                {typeof character.favourites === "number" && character.favourites > 0 && (
                    <Badge intent="alert" className="text-xs font-medium">
                        <BiSolidHeart className="w-3 h-3 mr-1" />
                        {character.favourites.toLocaleString()} favorites
                    </Badge>
                )}
            </div>

            {/* Alternative Names */}
            {altNames.length > 0 && (
                <div className="space-y-2">
                    <h3 className="text-sm font-semibold text-[--muted]">Alternative Names</h3>
                    <div className="flex flex-wrap gap-1">
                        {altNames.map((alt, i) => (
                            <Badge key={i} intent="gray" className="text-xs">
                                {alt}
                            </Badge>
                        ))}
                    </div>
                </div>
            )}
        </div>
    )

    // Create a mock media object to satisfy the MediaPageHeaderEntryDetails requirements
    const mockMedia = {
        id: character.id,
        title: {
            userPreferred: character.name?.full,
            english: character.name?.full,
            romaji: character.name?.native,
        },
        coverImage: {
            large: character.image?.large,
            medium: character.image?.medium,
            color: "#6366f1",
        },
        description: character.description,
    } as any

    return (
        <MediaPageHeader
            backgroundImage={character.image?.large}
            coverImage={character.image?.large}
        >
            <MediaPageHeaderDetailsContainer>
                <MediaPageHeaderEntryDetails
                    coverImage={character.image?.large || character.image?.medium}
                    title={character.name?.full}
                    englishTitle={character.name?.native}
                    color={"#6366f1"} // Default character color
                    description={character.description}
                    media={mockMedia}
                    type="anime" // Required prop
                    listData={undefined}
                >
                    {ts.mediaPageBannerInfoBoxSize === ThemeMediaPageInfoBoxSize.Fluid && <Details />}
                </MediaPageHeaderEntryDetails>

                {/* Action Buttons */}
                <div className="flex gap-3 items-center flex-wrap mt-4">
                    <Button
                        leftIcon={<BiHeart />}
                        intent="gray-outline"
                        size="md"
                        disabled
                    >
                        Add to Favorites
                    </Button>

                    {character.siteUrl && (
                        <a
                            href={character.siteUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-2 px-4 py-2 rounded-md border border-[--border] bg-transparent text-[--foreground] hover:bg-[--muted] transition text-sm font-medium"
                        >
                            <SiAnilist className="text-[#02a9ff] text-lg" />
                            View on AniList
                        </a>
                    )}
                </div>

                {/* Boxed Layout Details */}
                {ts.mediaPageBannerInfoBoxSize === ThemeMediaPageInfoBoxSize.Boxed && (
                    <div className="mt-6">
                        <Separator />
                        <div className="mt-6">
                            <Details />
                        </div>
                    </div>
                )}
            </MediaPageHeaderDetailsContainer>
        </MediaPageHeader>
    )
}

// Helper function to format birthday
function formatBirthday(dateOfBirth: { year?: number | null, month?: number | null, day?: number | null }) {
    const parts = []
    if (dateOfBirth.month) parts.push(new Date(2000, dateOfBirth.month - 1).toLocaleString('default', { month: 'short' }))
    if (dateOfBirth.day) parts.push(dateOfBirth.day)
    if (dateOfBirth.year) parts.push(dateOfBirth.year)
    return parts.join(' ')
}
