import React from "react"

export interface FamilyTreeData {
  canonical: any[]
  chronological: any[]
  alternatives: any[]
  charactersFrom: any[]
}

export function FamilyTreeSection({ data }: { data: FamilyTreeData | undefined }) {
  if (!data) return <div className="text-center text-slate-400 my-8">No family tree data available.</div>;
  const { canonical, chronological, alternatives, charactersFrom } = data;

  const hasAny = canonical.length || chronological.length || alternatives.length || charactersFrom.length;
  if (!hasAny) return <section className="my-8 p-6 rounded-xl bg-gradient-to-br from-slate-900/80 to-slate-800/40 border border-slate-700 text-center text-slate-400">No family tree found for this anime.</section>;

  return (
    <section className="my-8 p-6 rounded-xl bg-gradient-to-br from-slate-900/80 to-slate-800/40 border border-slate-700">
      <h2 className="text-2xl font-bold mb-4 text-sky-300 flex items-center gap-2">
        <span>Family Tree</span>
        <span className="text-xs font-normal text-slate-400">(Canonical, Chronological, Alternatives, Characters)</span>
      </h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {canonical.length > 0 && (
          <div>
            <h3 className="font-semibold text-sky-200 mb-2">Canonical (Sequels/Prequels)</h3>
            <ul className="space-y-1">
              {canonical.map((entry, i) => (
                <li key={i} className="text-slate-100">{entry.title} <span className="text-xs text-slate-400">({entry.type})</span></li>
              ))}
            </ul>
          </div>
        )}
        {chronological.length > 0 && (
          <div>
            <h3 className="font-semibold text-sky-200 mb-2">Chronological</h3>
            <ul className="space-y-1">
              {chronological.map((entry, i) => (
                <li key={i} className="text-slate-100">{entry.title} <span className="text-xs text-slate-400">({entry.type})</span></li>
              ))}
            </ul>
          </div>
        )}
        {alternatives.length > 0 && (
          <div>
            <h3 className="font-semibold text-sky-200 mb-2">Alternatives</h3>
            <ul className="space-y-1">
              {alternatives.map((entry, i) => (
                <li key={i} className="text-slate-100">{entry.title} <span className="text-xs text-slate-400">({entry.type})</span></li>
              ))}
            </ul>
          </div>
        )}
        {charactersFrom.length > 0 && (
          <div>
            <h3 className="font-semibold text-sky-200 mb-2">Character Used From</h3>
            <ul className="space-y-1">
              {charactersFrom.map((entry, i) => (
                <li key={i} className="text-slate-100">{entry.title} <span className="text-xs text-slate-400">({entry.type})</span></li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </section>
  )
}
