'use client';

import { ScrollArea } from "@/components/ui/scroll-area"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { CalendarIcon, FileTextIcon } from "lucide-react"

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
    );
}
