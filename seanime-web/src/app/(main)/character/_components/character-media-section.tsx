import { AL_CharacterDetailsByID_Character } from "@/api/generated/types"
import { useGetCharacterMedia } from "@/api/hooks/character.hooks"
import { MediaCardLazyGrid } from "@/app/(main)/_features/media/_components/media-card-grid"
import { MediaEntryCard } from "@/app/(main)/_features/media/_components/media-entry-card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { Select } from "@/components/ui/select"
import { StaticTabs } from "@/components/ui/tabs"
import { TextInput } from "@/components/ui/text-input"
import { useDebounce } from "@/hooks/use-debounce"
import { capitalize } from "lodash"
import React from "react"
import { BiSearch } from "react-icons/bi"

type CharacterMediaSectionProps = {
    characterId: string
    character: AL_CharacterDetailsByID_Character
}

export function CharacterMediaSection(props: CharacterMediaSectionProps) {
    const { characterId, character } = props

    const { mutate: getCharacterMedia, data: characterMedia, isPending: characterMediaLoading } = useGetCharacterMedia(characterId)

    const [selectedRole, setSelectedRole] = React.useState<string>("all")
    const [selectedType, setSelectedType] = React.useState<string>("all") 
    const [searchQuery, setSearchQuery] = React.useState("")
    const debouncedSearchQuery = useDebounce(searchQuery, 300)

    // Load character media on mount
    React.useEffect(() => {
        getCharacterMedia({ page: 1, perPage: 50 })
    }, [getCharacterMedia])

    const mediaEntries = React.useMemo(() => {
        if (!characterMedia?.edges) return []
        
        return characterMedia.edges
            .filter(edge => {
                if (!edge?.node) return false
                
                // Filter by role
                if (selectedRole !== "all" && edge.characterRole !== selectedRole.toUpperCase()) {
                    return false
                }
                
                // Filter by type
                if (selectedType !== "all" && edge.node.type !== selectedType.toUpperCase()) {
                    return false
                }
                
                // Filter by search query
                if (debouncedSearchQuery) {
                    const query = debouncedSearchQuery.toLowerCase()
                    const title = edge.node.title?.english?.toLowerCase() || 
                                 edge.node.title?.romaji?.toLowerCase() || 
                                 edge.node.title?.native?.toLowerCase() || ""
                    return title.includes(query)
                }
                
                return true
            })
            .slice(0, 25) // Limit to 25 entries for performance
    }, [characterMedia?.edges, selectedRole, selectedType, debouncedSearchQuery])

    const roleOptions = React.useMemo(() => {
        if (!characterMedia?.edges) return []
        
        const roles = new Set<string>()
        characterMedia.edges.forEach(edge => {
            if (edge?.characterRole) {
                roles.add(edge.characterRole)
            }
        })
        
        return Array.from(roles).map(role => ({
            label: capitalize(role),
            value: role.toLowerCase()
        }))
    }, [characterMedia?.edges])

    function mapCharacterMediaNodeToBaseMedia(node: any) {
        return {
            ...node,
            title: {
                ...node.title,
                userPreferred: node.title?.english || node.title?.romaji || node.title?.native || "Untitled",
            },
            coverImage: {
                ...node.coverImage,
                extraLarge: node.coverImage?.large || node.coverImage?.medium || "/no-cover.png",
            },
        }
    }

    if (!characterMedia?.edges?.length && !characterMediaLoading) {
        return (
            <div className="text-center py-8">
                <p className="text-[--muted]">No media appearances found for this character.</p>
            </div>
        )
    }

    return (
        <div className="space-y-6">
            <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
                <div className="space-y-2">
                    <h2 className="text-2xl font-bold">Media Appearances</h2>
                    {characterMedia?.edges && (
                        <p className="text-[--muted]">
                            {mediaEntries.length} of {characterMedia.edges.length} entries
                        </p>
                    )}
                </div>

                {/* Filters */}
                <div className="flex flex-col sm:flex-row gap-3">
                    <div className="relative">
                        <TextInput
                            placeholder="Search media..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            leftIcon={<BiSearch />}
                            className="min-w-[200px]"
                        />
                    </div>

                    <Select
                        value={selectedType}
                        onValueChange={setSelectedType}
                        options={[
                            { label: "All Types", value: "all" },
                            { label: "Anime", value: "anime" },
                            { label: "Manga", value: "manga" }
                        ]}
                    />

                    {roleOptions.length > 0 && (
                        <Select
                            value={selectedRole}
                            onValueChange={setSelectedRole}
                            options={[
                                { label: "All Roles", value: "all" },
                                ...roleOptions
                            ]}
                        />
                    )}
                </div>
            </div>

            {/* Media Grid */}
            {characterMediaLoading ? (
                <div className="flex justify-center py-8">
                    <LoadingSpinner />
                </div>
            ) : (
                <MediaCardLazyGrid itemCount={mediaEntries.length}>
                    {mediaEntries.map((edge) => {
                        const node = edge.node;
                        if (!node) return null;
                        const mappedMedia = mapCharacterMediaNodeToBaseMedia(node);
                        return (
                            <MediaEntryCard
                                key={node.id}
                                media={mappedMedia}
                                type={node.type === "MANGA" ? "manga" : "anime"}
                            />
                        )
                    })}
                </MediaCardLazyGrid>
            )}

            {/* Load More Button */}
            {characterMedia?.edges && characterMedia.edges.length > 25 && (
                <div className="flex justify-center pt-4">
                    <Button
                        intent="gray-outline"
                        onClick={() => {
                            // This would implement pagination in a real scenario
                            // For now, we're just showing the first 25 entries
                        }}
                        disabled
                    >
                        Load More (Coming Soon)
                    </Button>
                </div>
            )}
        </div>
    )
}
