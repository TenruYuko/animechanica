import { useServerQuery, useServerMutation } from "@/api/client/requests"
import { GetCharacterMedia_Variables } from "@/api/generated/endpoint.types"
import { API_ENDPOINTS } from "@/api/generated/endpoints"
import {
    AL_CharacterDetailsByID_Character,
    AL_CharacterDetailsByID_Character_Media,
    Nullish,
} from "@/api/generated/types"

export function useGetCharacterDetails(id: Nullish<number | string>) {
    return useServerQuery<AL_CharacterDetailsByID_Character>({
        endpoint: API_ENDPOINTS.CHARACTER.GetCharacterDetails.endpoint.replace("{id}", String(id)),
        method: API_ENDPOINTS.CHARACTER.GetCharacterDetails.methods[0],
        queryKey: [API_ENDPOINTS.CHARACTER.GetCharacterDetails.key, String(id)],
        enabled: !!id,
    })
}

export function useGetCharacterMedia(id: Nullish<number | string>) {
    return useServerMutation<AL_CharacterDetailsByID_Character_Media, GetCharacterMedia_Variables>({
        endpoint: API_ENDPOINTS.CHARACTER.GetCharacterMedia.endpoint.replace("{id}", String(id)),
        method: API_ENDPOINTS.CHARACTER.GetCharacterMedia.methods[0],
        mutationKey: [API_ENDPOINTS.CHARACTER.GetCharacterMedia.key, String(id)],
    })
}
