'use client';

import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { History, Search, ChevronDown, ChevronRight, Pin, X, Tag } from "lucide-react"
import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export interface Document {
    codice_redazionale: string;
    data_pubblicazione_gazzetta: string;
    title: string;
    isPinned?: boolean;
    category?: string;
}

interface SidebarProps {
    results: any[];
    history: Document[];
    bookmarks: Document[];
    currentIndex: number;
    uiState?: any;
    onSelectDocument: (doc: any) => void;
    onSelectHistory: (index: number) => void;
    onRemoveHistory: (index: number, e: any) => void;
    onToggleBookmark: (doc: Document, e: any) => void;
    onUpdateBookmarkCategory?: (docID: string, category: string) => void;
    onUIStateChange?: (state: any) => void;
    selectedDocument: any;
}

export default function Sidebar({
    results,
    history,
    bookmarks,
    currentIndex,
    uiState,
    onSelectDocument,
    onSelectHistory,
    onRemoveHistory,
    onToggleBookmark,
    onUpdateBookmarkCategory,
    onUIStateChange,
    selectedDocument
}: SidebarProps) {
    const [showResults, setShowResults] = useState(uiState?.showResults ?? true);
    const [activeTab, setActiveTab] = useState<'history' | 'bookmarks'>(uiState?.activeTab ?? 'history');
    const [collapsed, setCollapsed] = useState(false);

    // Auto-collapse if no results on search
    useEffect(() => {
        if (results.length === 0 && showResults) {
            setShowResults(false);
        } else if (results.length > 0 && !showResults && !uiState?.userCollapsed) {
            setShowResults(true);
        }
    }, [results.length]);

    const handleToggleResults = () => {
        const newState = !showResults;
        setShowResults(newState);
        onUIStateChange?.({ ...uiState, showResults: newState, userCollapsed: !newState });
    };

    const handleTabChange = (tab: 'history' | 'bookmarks') => {
        setActiveTab(tab);
        onUIStateChange?.({ ...uiState, activeTab: tab });
    };

    const isBookmarked = (doc: Document) => {
        return bookmarks.some(b => b.codice_redazionale === doc.codice_redazionale);
    };

    return (
        <div className={`h-full flex flex-col transition-all duration-300 ${collapsed ? 'w-12 overflow-hidden' : 'w-80 lg:w-96'}`}>
            <div className="flex items-center justify-between mb-4 px-1 shrink-0 h-10 border-b border-border/50">
                {!collapsed && <span className="text-xs font-bold uppercase tracking-widest text-muted-foreground ml-2">Navigation</span>}
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setCollapsed(!collapsed)}
                    className={`h-8 w-8 text-muted-foreground ${collapsed ? 'mx-auto' : 'ml-auto'}`}
                >
                    {collapsed ? <ChevronRight className="h-4 w-4" /> : <X className="h-4 w-4" />}
                </Button>
            </div>

            {!collapsed && (
                <div className="flex flex-col flex-1 space-y-4 min-h-0">
                    {/* Search Results Section */}
                    <div className={`flex flex-col min-h-0 transition-all duration-300 ${showResults ? 'flex-1' : 'flex-none'}`}>
                        <div className="flex justify-between items-center px-1 mb-2">
                            <button
                                onClick={handleToggleResults}
                                className="flex items-center text-sm font-semibold text-muted-foreground hover:text-foreground transition-colors w-full group"
                            >
                                <Search className="h-4 w-4 mr-2" />
                                <span className="flex-1 text-left">Search Results ({results.length})</span>
                                {showResults ? (
                                    <ChevronDown className="h-4 w-4 opacity-50 group-hover:opacity-100" />
                                ) : (
                                    <ChevronRight className="h-4 w-4 opacity-50 group-hover:opacity-100" />
                                )}
                            </button>
                        </div>

                        {showResults && (
                            <div className="flex-1 min-h-0 flex flex-col">
                                {results.length > 0 ? (
                                    <ScrollArea className="flex-1 pr-3 -mr-3">
                                        <div className="space-y-2 pb-2">
                                            {results.map((result, index) => (
                                                <div
                                                    key={index}
                                                    onClick={() => onSelectDocument(result)}
                                                    className={`p-3 rounded-md cursor-pointer transition-all border text-sm ${selectedDocument?.codice_redazionale === result.codice_redazionale
                                                        ? 'bg-primary/10 border-primary text-primary font-medium'
                                                        : 'bg-card border-border hover:bg-accent hover:text-accent-foreground'
                                                        }`}
                                                >
                                                    <p className="line-clamp-2 leading-snug">{result.title}</p>
                                                    <div className="flex justify-between items-center mt-2 text-xs opacity-70">
                                                        <span>{result.data_pubblicazione_gazzetta}</span>
                                                        <span className="font-mono bg-muted/50 px-1 rounded">{result.codice_redazionale}</span>
                                                    </div>
                                                </div>
                                            ))}
                                        </div>
                                    </ScrollArea>
                                ) : (
                                    <div className="text-center py-8 text-muted-foreground text-sm border border-dashed rounded-lg bg-card/50">
                                        No results
                                    </div>
                                )}
                            </div>
                        )}
                    </div>

                    <Separator />

                    {/* History / Bookmarks Tabs */}
                    <div className="flex-1 flex flex-col min-h-0">
                        <div className="flex items-center space-x-4 px-1 mb-2 border-b border-border/50 pb-2">
                            <button
                                onClick={() => handleTabChange('history')}
                                className={`flex items-center text-sm font-semibold transition-colors pb-1 border-b-2 ${activeTab === 'history' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
                            >
                                <History className="h-4 w-4 mr-2" />
                                History
                            </button>
                            <button
                                onClick={() => handleTabChange('bookmarks')}
                                className={`flex items-center text-sm font-semibold transition-colors pb-1 border-b-2 ${activeTab === 'bookmarks' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
                            >
                                <Pin className="h-4 w-4 mr-2" />
                                Bookmarks
                            </button>
                        </div>

                        <ScrollArea className="flex-1 pr-3 -mr-3">
                            <div className="space-y-2 pb-2 relative">
                                {activeTab === 'history' ? (
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
                                                        <p className="text-xs opacity-60 font-mono mt-0.5">{doc.codice_redazionale}</p>
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
                                {activeTab === 'bookmarks' && bookmarks.length === 0 && (
                                    <div className="text-center py-8 text-muted-foreground text-sm">
                                        No bookmarks yet
                                    </div>
                                )}
                                {activeTab === 'history' && history.length === 0 && (
                                    <div className="text-center py-8 text-muted-foreground text-sm">
                                        History is empty
                                    </div>
                                )}
                            </div>
                        </ScrollArea>
                    </div>
                </div>
            )}
        </div>
    );
}
