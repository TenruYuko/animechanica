import React from "react"

export function FamilyTreeSection({ relations }: { relations: any[] }) {
  // Placeholder for now
  return (
    <section>
      <h2>Family Tree</h2>
      <div>
        {/* Render family tree here */}
        {relations && relations.length > 0 ? (
          <pre>{JSON.stringify(relations, null, 2)}</pre>
        ) : (
          <p>No family tree data available.</p>
        )}
      </div>
    </section>
  )
}
