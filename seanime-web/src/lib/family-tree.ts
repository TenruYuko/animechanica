import path from "path";
import { promises as fs } from "fs";

const DB_PATH = path.resolve(process.cwd(), "external/anime-offline-database.json");

export interface FamilyTreeEntry {
  id: number;
  title: string;
  type?: string;
  relationType?: string;
}

export interface FamilyTreeData {
  canonical: FamilyTreeEntry[];
  chronological: FamilyTreeEntry[];
  alternatives: FamilyTreeEntry[];
  charactersFrom: FamilyTreeEntry[];
}

/**
 * Looks up a media entry by local id, extracts AniList ID, and fetches family tree data from AniList.
 * @param mediaId Local media id (string or number)
 * @returns FamilyTreeData or throws error
 */
export async function getFamilyTreeForMediaId(mediaId: string | number): Promise<FamilyTreeData> {
  let db: any;
  try {
    const data = await fs.readFile(DB_PATH, "utf-8");
    db = JSON.parse(data);
  } catch (e) {
    throw new Error("Could not load offline database");
  }

  // Find the anime entry in offline DB by mediaId
  const entry = (db.data || db.anime)?.find((a: any) => String(a.id) === String(mediaId));
  if (!entry) throw new Error("Not found");

  // Extract AniList ID from sources
  let aniListId: string | undefined = undefined;
  if (Array.isArray(entry.sources)) {
    for (const src of entry.sources) {
      const match = src.match(/anilist.co\/anime\/(\d+)/);
      if (match) {
        aniListId = match[1];
        break;
      }
    }
  }
  if (!aniListId) {
    throw new Error("AniList ID not found in sources");
  }

  // Query AniList GraphQL API for relations
  const query = `
    query ($id: Int!) {
      Media (id: $id) {
        id
        title { romaji }
        relations {
          edges {
            relationType
            node {
              id
              title { romaji }
              type
            }
          }
        }
      }
    }
  `;
  let relationsData: any[] = [];
  try {
    const resp = await fetch("https://graphql.anilist.co", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ query, variables: { id: Number(aniListId) } })
    });
    const json = await resp.json() as {
      data?: {
        Media?: {
          relations?: {
            edges?: any[]
          }
        }
      }
    };
    relationsData = json.data?.Media?.relations?.edges || [];

  } catch (e) {
    throw new Error("Failed to fetch from AniList");
  }

  // Map AniList relation types to FamilyTreeData
  function filterByType(typeArr: string[]) {
    return relationsData.filter((edge: any) => typeArr.includes(edge.relationType)).map((edge: any) => ({
      id: edge.node.id,
      title: edge.node.title.romaji,
      type: edge.node.type,
      relationType: edge.relationType
    }));
  }
  const familyTreeData: FamilyTreeData = {
    canonical: filterByType(["PREQUEL", "SEQUEL", "PARENT", "CHILD"]),
    chronological: filterByType(["ADAPTATION", "SOURCE", "SUMMARY", "SPIN_OFF"]),
    alternatives: filterByType(["ALTERNATIVE", "SIDE_STORY", "CHARACTER"]),
    charactersFrom: filterByType(["CHARACTER"]),
  };

  return familyTreeData;
}
