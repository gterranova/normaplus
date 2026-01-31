'use client';

import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { History, Search, ChevronDown, ChevronUp, PanelLeftClose, Pin, PanelLeftOpen, Tag } from "lucide-react"
import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import ResultList from "./ResultList";
import HistoryPanel from "./HistoryPanel";
import BookmarksPanel from "./BookmarksPanel";

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

export default function LeftSidebar({
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
        <div className={`h-full flex flex-col transition-all duration-300 ${collapsed ? 'w-12 overflow-hidden' : 'w-80'}`}>
            <div className="flex justify-between items-center px-1 pb-2 mb-2 border-b border-border/50">
                {!collapsed && <span className="text-xs font-bold uppercase text-muted-foreground ml-2">Navigation</span>}
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setCollapsed(!collapsed)}
                    className={`h-8 w-8 text-muted-foreground ${collapsed ? 'mx-auto' : 'ml-auto'}`}
                >
                    {collapsed ? <PanelLeftOpen className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
                </Button>
            </div>

            {!collapsed && (
                <div className="flex flex-col flex-1 space-y-4 min-h-0">
                    {/* Search Results Section */}
                    <div className={`flex flex-col min-h-0 transition-all duration-300 ${showResults ? 'flex-1' : 'flex-none'}`}>
                        <div className="flex justify-between items-center px-1 mb-2">
                            <button
                                onClick={results.length ? handleToggleResults : undefined}
                                className="flex items-center text-sm font-semibold text-muted-foreground hover:text-foreground transition-colors w-full group"
                            >
                                <Search className="h-4 w-4 mr-2" />
                                <span className="flex-1 text-left">Search Results ({results.length})</span>
                                {results.length ? (
                                    showResults ? (
                                        <ChevronUp className="h-4 w-4 mr-2 opacity-50 group-hover:opacity-100" />
                                    ) : (
                                        <ChevronDown className="h-4 w-4 mr-2 opacity-50 group-hover:opacity-100" />
                                    )
                                ) : ""}
                            </button>
                        </div>

                        {showResults && (
                            <div className="flex-1 min-h-0 flex flex-col">
                                <ResultList results={results} onSelectDocument={onSelectDocument} selectedDocument={selectedDocument} />
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
                            {activeTab === 'history' ? (
                                <HistoryPanel history={history} isBookmarked={isBookmarked} onSelectHistory={onSelectHistory} onToggleBookmark={onToggleBookmark} currentIndex={currentIndex} onRemoveHistory={onRemoveHistory} />
                            ) : (
                                <BookmarksPanel bookmarks={bookmarks} onSelectDocument={onSelectDocument} onToggleBookmark={onToggleBookmark} onUpdateBookmarkCategory={onUpdateBookmarkCategory} />
                            )}
                        </ScrollArea>
                    </div>
                </div>
            )}
        </div>
    );
}
