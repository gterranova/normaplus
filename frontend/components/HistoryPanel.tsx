'use client';

import { Pin, X } from "lucide-react"
import { SOURCE_LABEL, isEU, sourceOf } from "@/lib/source"

interface HistoryPanelProps {
    history: any[];
    isBookmarked: (doc: any) => boolean;
    onSelectHistory: (index: number) => void;
    onToggleBookmark: (doc: any, e: any) => void;
    currentIndex: number;
    onRemoveHistory: (index: number, e: any) => void;
}
export default function HistoryPanel({
    history,
    isBookmarked,
    onSelectHistory,
    onRemoveHistory,
    onToggleBookmark,
    currentIndex,
}: HistoryPanelProps) {
    return (
        <div className="space-y-2 pb-2 relative">
            {(history?.length || 0) === 0 ? (
                <p className="text-xs text-muted-foreground text-center py-12 italic">History is empty</p>
            ) : (
                history.map((doc, index) => {
                    const pinned = isBookmarked(doc);
                    return (
                        <div
                            key={`hist-${doc.codice_redazionale}-${index}`}
                            onClick={() => onSelectHistory(index)}
                            className={`ml-2 p-2 rounded-md cursor-pointer text-sm transition-all border group relative ${index === currentIndex
                                ? 'bg-accent border-primary/50 text-foreground shadow-sm'
                                : pinned
                                    ? 'bg-amber-50/50 dark:bg-amber-900/10 border-amber-200/50'
                                    : 'bg-transparent border-transparent hover:bg-accent/50 text-muted-foreground'
                                }`}
                        >
                            <div className="flex justify-between items-start">
                                <div className="flex-1">
                                    <p className={`line-clamp-2 ${pinned ? 'font-medium text-amber-700 dark:text-amber-400' : ''}`}>
                                        {doc.title || "Document"}
                                    </p>
                                    <p className="text-xs opacity-60 mt-0.5 flex items-center gap-1.5">
                                        {/* Named only for EU acts: Normattiva is the ordinary case, and a
                                            badge on every row would say nothing while costing a line. */}
                                        {isEU(doc) && (
                                            <span className="shrink-0 font-semibold uppercase text-[9px] tracking-wide px-1 rounded border border-current/40">
                                                {SOURCE_LABEL[sourceOf(doc)]}
                                            </span>
                                        )}
                                        <span className="font-mono truncate">{doc.codice_redazionale}</span>
                                    </p>
                                </div>
                                <div className="flex space-x-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                    <button
                                        onClick={(e) => onToggleBookmark(doc, e)}
                                        className={`p-1 rounded hover:bg-background ${pinned ? 'opacity-100 text-amber-500' : 'text-muted-foreground'}`}
                                        title={pinned ? "Unpin" : "Pin"}
                                    >
                                        <Pin className={`h-3 w-3 ${pinned ? 'fill-current' : ''}`} />
                                    </button>
                                    <button
                                        onClick={(e) => onRemoveHistory(index, e)}
                                        className="p-1 rounded hover:bg-background text-muted-foreground hover:text-destructive"
                                        title="Remove"
                                    >
                                        <X className="h-3 w-3" />
                                    </button>
                                </div>
                            </div>
                        </div>
                    );
                })
            )}
        </div>
    );
}

