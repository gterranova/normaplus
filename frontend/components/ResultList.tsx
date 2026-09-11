'use client';

import { ScrollArea } from "@/components/ui/scroll-area"
import { SOURCE_LABEL, sourceOf } from "@/lib/source"

interface ResultListProps {
    results: any[];
    onSelectDocument: (doc: any) => void;
    selectedDocument: any;
}

export default function ResultList({ results, onSelectDocument, selectedDocument }: ResultListProps) {
    if (results.length === 0) {
        return (
            <div className="border border-dashed rounded-lg bg-card/50">
                <p className="text-xs text-muted-foreground text-center py-8 italic">Start a search to see results.</p>
            </div>
        );
    }

    return (
        <ScrollArea className="flex-1 pr-3 -mr-3">
            <div className="space-y-2 pb-2">
                {results.map((result, index) => {
                    const src = sourceOf(result);
                    // Selection is by id AND source: the two archives number
                    // documents independently, so an id alone could match a
                    // result the user did not click.
                    const selected = selectedDocument?.codice_redazionale === result.codice_redazionale
                        && sourceOf(selectedDocument) === src;
                    return (
                        <div
                            key={`${src}:${result.codice_redazionale}:${index}`}
                            onClick={() => onSelectDocument(result)}
                            className={`p-3 rounded-md cursor-pointer transition-all border text-sm ${selected
                                ? 'bg-primary/10 border-primary text-primary font-medium'
                                : 'bg-card border-border hover:bg-accent hover:text-accent-foreground'
                                }`}
                        >
                            <p className="line-clamp-2 leading-snug">{result.title}</p>
                            <div className="flex justify-between items-center mt-2 text-xs opacity-70 gap-2">
                                <span className="shrink-0 font-semibold uppercase text-[9px] tracking-wide px-1 rounded border border-current/30">
                                    {SOURCE_LABEL[src]}
                                </span>
                                {/* An EU act has no gazzetta date; the slot is left empty rather
                                    than filled with something that looks like one. */}
                                <span className="truncate">{result.data_pubblicazione_gazzetta}</span>
                                <span className="font-mono bg-muted/50 px-1 rounded shrink-0">{result.codice_redazionale}</span>
                            </div>
                        </div>
                    );
                })}
            </div>
        </ScrollArea>
    );
}
