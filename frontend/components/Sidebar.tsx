'use client';

import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { History, Search, ChevronDown, ChevronRight } from "lucide-react"
import { useState } from "react"

interface Document {
    codice_redazionale: string;
    data_pubblicazione_gazzetta: string;
    title: string;
}

interface SidebarProps {
    results: any[];
    history: Document[];
    currentIndex: number;
    onSelectDocument: (doc: any) => void;
    onSelectHistory: (index: number) => void;
    selectedDocument: any;
}

export default function Sidebar({
    results,
    history,
    currentIndex,
    onSelectDocument,
    onSelectHistory,
    selectedDocument
}: SidebarProps) {
    const [showResults, setShowResults] = useState(true);

    return (
        <div className="h-full flex flex-col space-y-4">
            {/* Search Results Section */}
            {/* Helper class to control flexible height: flex-1 when open, flex-none when closed */}
            <div className={`flex flex-col min-h-0 transition-all duration-300 ${showResults ? 'flex-1' : 'flex-none'}`}>
                <div className="flex justify-between items-center px-1 mb-2">
                    <button
                        onClick={() => setShowResults(!showResults)}
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
                                                <span className="font-mono">{result.codice_redazionale}</span>
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

            {/* History Section - Takes remaining space */}
            <div className="flex-1 flex flex-col min-h-0">
                <div className="flex items-center px-1 mb-2 text-sm font-semibold text-muted-foreground">
                    <History className="h-4 w-4 mr-2" />
                    History Stack
                </div>
                <ScrollArea className="flex-1 pr-3 -mr-3">
                    <div className="space-y-2 pb-2 relative">
                        {history.map((doc, index) => (
                            <div key={index} className="relative pl-4 group">
                                {/* Connector Line */}
                                {index < history.length - 1 && (
                                    <div className="absolute left-[5px] top-6 bottom-[-8px] w-px bg-border"></div>
                                )}
                                {/* Dot */}
                                <div className={`absolute left-0 top-3 w-2.5 h-2.5 rounded-full border-2 transition-colors ${index === currentIndex
                                    ? 'bg-primary border-primary'
                                    : 'bg-background border-muted-foreground group-hover:border-primary'
                                    }`}></div>

                                <div
                                    onClick={() => onSelectHistory(index)}
                                    className={`ml-2 p-2 rounded-md cursor-pointer text-sm transition-all border ${index === currentIndex
                                        ? 'bg-accent border-primary/50 text-foreground shadow-sm'
                                        : 'bg-transparent border-transparent hover:bg-accent/50 text-muted-foreground'
                                        }`}
                                >
                                    <p className="line-clamp-1">{doc.title || "Document"}</p>
                                    <p className="text-xs opacity-60 font-mono mt-0.5">{doc.codice_redazionale}</p>
                                </div>
                            </div>
                        ))}
                    </div>
                </ScrollArea>
            </div>
        </div>
    );
}
