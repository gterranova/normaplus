'use client';

import { useState, FormEvent } from 'react';
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Search } from "lucide-react"
import { Source, SOURCE_LABEL } from "@/lib/source"

interface SearchBarProps {
    onSearch: (query: string, source: Source) => void;
    loading: boolean;
    // The selection is the page's, not this component's. Two reasons, both
    // observed: the bar is rendered twice (desktop and mobile) and local state
    // let the two disagree, and a search can be started from elsewhere — the
    // transposition panel searches Normattiva for a measure it names — after
    // which a toggle owning its own state goes on claiming EUR-Lex over a list
    // of Italian results.
    source: Source;
    onSourceChange: (source: Source) => void;
}

// The two archives do not answer the same kind of question, so the placeholder
// follows the selection: Normattiva is addressed by a citation, while EUR-Lex is
// searched by the words of a title (or addressed by a CELEX outright).
const PLACEHOLDER: Record<Source, string> = {
    normattiva: "Reference (e.g. Costituzione or 28 dicembre 2000, n. 445)...",
    eurlex: "Title words, CELEX or citation (e.g. cibersicurezza, 32022L2555, direttiva 2022/2555)...",
};

export default function SearchBar({ onSearch, loading, source, onSourceChange }: SearchBarProps) {
    const [query, setQuery] = useState('');

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();
        if (query.trim()) {
            onSearch(query.trim(), source);
        }
    };

    return (
        <form onSubmit={handleSubmit} className="w-full max-w-3xl mx-auto space-y-1">
            <div className="flex items-center space-x-2">
                <div
                    className="flex items-center rounded-md border border-input bg-background p-0.5 shadow-sm shrink-0"
                    role="radiogroup"
                    aria-label="Archive to search"
                >
                    {(Object.keys(SOURCE_LABEL) as Source[]).map((s) => (
                        <button
                            key={s}
                            type="button"
                            role="radio"
                            aria-checked={source === s}
                            disabled={loading}
                            onClick={() => onSourceChange(s)}
                            className={`h-7 px-2.5 rounded text-[11px] font-semibold uppercase tracking-wide transition-colors disabled:opacity-50 ${source === s
                                ? 'bg-primary text-primary-foreground'
                                : 'text-muted-foreground hover:text-foreground'
                                }`}
                        >
                            {SOURCE_LABEL[s]}
                        </button>
                    ))}
                </div>
                <div className="relative w-full">
                    <Search className="absolute left-2.5 top-2 h-4 w-4 text-muted-foreground" />
                    <Input
                        type="text"
                        placeholder={PLACEHOLDER[source]}
                        value={query}
                        onChange={(e) => setQuery(e.target.value)}
                        className="pl-8 h-8 bg-background border-input shadow-sm"
                        disabled={loading}
                    />
                </div>
                <Button type="submit" disabled={loading || !query.trim()} className="h-8 px-8 font-semibold shadow-sm">
                    {loading ? 'Searching...' : 'Search'}
                </Button>
            </div>

            {source === 'eurlex' && (
                // Two facts a reader cannot infer from an empty result list, and
                // which change what they should do about it: only titles are
                // indexed, and the query is slow because CELLAR has no index for
                // it. Without them, "no results" reads as "the Union has not
                // legislated on this".
                <p className="text-[10px] text-muted-foreground text-center px-2 leading-snug">
                    EUR-Lex search runs over the <strong>titles</strong> of directives and regulations, not their text &mdash;
                    an act that does not name the subject in its title will not appear. It can take around twenty seconds.
                </p>
            )}
        </form>
    );
}
