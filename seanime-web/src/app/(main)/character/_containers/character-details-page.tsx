import { useGetCharacterDetails } from "@/api/hooks/character.hooks"
import { MediaEntryPageLoadingDisplay } from "@/app/(main)/_features/media/_components/media-entry-page-loading-display"
import { CustomLibraryBanner } from "@/app/(main)/(library)/_containers/custom-library-banner"
import { CharacterDetailsSection, CharacterMediaSection } from "@/app/(main)/character/_components"
import { PageWrapper } from "@/components/shared/page-wrapper"
import React from "react"

type CharacterDetailsPageProps = {
    characterId: string
}

export function CharacterDetailsPage(props: CharacterDetailsPageProps) {
    const { characterId } = props

    const { data: characterDetails, isLoading: characterDetailsLoading } = useGetCharacterDetails(characterId)

    React.useEffect(() => {
        if (characterDetails?.name?.full) {
            document.title = `${characterDetails.name.full} | Seanime`
        }
    }, [characterDetails])

    if (characterDetailsLoading) return <MediaEntryPageLoadingDisplay />
    if (!characterDetails) return null

    return (
        <>
            <CustomLibraryBanner />
            
            <CharacterDetailsSection character={characterDetails} />
            
            <PageWrapper className="px-4 md:px-8 relative z-[8]">
                <div className="space-y-10 pb-10 pt-4">
                    <CharacterMediaSection characterId={characterId} character={characterDetails} />
                </div>
            </PageWrapper>
        </>
    )
}
