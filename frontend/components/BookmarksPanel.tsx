'use client';

import { Pin, Tag } from "lucide-react"

export interface Document {
    codice_redazionale: string;
    data_pubblicazione_gazzetta: string;
    title: string;
    isPinned?: boolean;
    category?: string;
}

interface BookmarksPanelProps {
    bookmarks: Document[];
    onSelectDocument: (doc: Document) => void;
    onUpdateBookmarkCategory?: (docID: string, category: string) => void;
    onToggleBookmark: (doc: Document, e: any) => void;
}
export default function BookmarksPanel({
    bookmarks,
    onSelectDocument,
    onUpdateBookmarkCategory,
    onToggleBookmark
}: BookmarksPanelProps) {
    return (
        <div className="space-y-2 pb-2 relative">
            {(bookmarks?.length || 0) === 0 ? (
                <p className="text-xs text-muted-foreground text-center py-12 italic">No bookmarks yet</p>
            ) : (
                bookmarks.map((doc, index) => (
                    <div
                        key={`bk-${doc.codice_redazionale}`}
                        onClick={() => onSelectDocument(doc)}
                        className="ml-2 p-2 rounded-md cursor-pointer text-sm transition-all border group relative bg-amber-50/50 dark:bg-amber-900/10 border-amber-200/50 mb-2"
                    >
                        <div className="flex justify-between items-start">
                            <div className="flex-1">
                                <p className="font-medium text-amber-700 dark:text-amber-400 line-clamp-2">
                                    {doc.title || "Document"}
                                </p>
                                <div className="flex items-center mt-1 space-x-2">
                                    <Tag className="h-3 w-3 text-amber-600/50" />
                                    <input
                                        className="text-[10px] bg-transparent border-none p-0 h-4 w-full focus:ring-0 text-amber-600/70 placeholder:text-amber-600/30"
                                        placeholder="Set category..."
                                        defaultValue={doc.category}
                                        onClick={(e) => e.stopPropagation()}
                                        onBlur={(e) => onUpdateBookmarkCategory?.(doc.codice_redazionale, e.target.value)}
                                        onKeyDown={(e) => e.key === 'Enter' && e.currentTarget.blur()}
                                    />
                                </div>
                                <p className="text-xs opacity-60 font-mono mt-0.5">{doc.codice_redazionale}</p>
                            </div>
                            <div className="flex space-x-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                <button
                                    onClick={(e) => onToggleBookmark(doc, e)}
                                    className="p-1 rounded hover:bg-background text-amber-500"
                                    title="Unpin"
                                >
                                    <Pin className="h-3 w-3 fill-current" />
                                </button>
                            </div>
                        </div>
                    </div>
                ))
            )}
        </div>
    );
}

