// Which archive a document came from.
//
// The application addresses an EU act and an Italian one through the same
// fields: the CELEX occupies the `codice_redazionale` slot, so history,
// bookmarks and annotations need no second identifier. What cannot be shared is
// *where to ask next* — the two are fetched from different archives — so the
// source is the one thing that has to travel alongside the id.

export type Source = 'normattiva' | 'eurlex';

export const SOURCE_LABEL: Record<Source, string> = {
    normattiva: 'Normattiva',
    eurlex: 'EUR-Lex',
};

// sourceOf reads the source off a record, defaulting to normattiva.
//
// Absent means normattiva rather than unknown: every record written before
// EUR-Lex existed is Italian legislation, and a default keeps those working
// untouched. It is deliberately NOT inferred from the shape of the identifier —
// a guess that is right today is a guess that goes wrong the first time either
// archive changes how it numbers things, and silently.
export function sourceOf(doc: { source?: string } | null | undefined): Source {
    return doc?.source === 'eurlex' ? 'eurlex' : 'normattiva';
}

export function isEU(doc: { source?: string } | null | undefined): boolean {
    return sourceOf(doc) === 'eurlex';
}
