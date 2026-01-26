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
            <div className="text-center py-12 text-muted-foreground">
                <div className="flex justify-center mb-4">
                    <FileTextIcon className="h-12 w-12 opacity-20" />
                </div>
                <p>No documents found. Start a search to see results.</p>
            </div>
        );
    }

    return (
        <div className="h-[calc(100vh-12rem)] flex flex-col">
            <h2 className="text-xl font-semibold mb-4 text-foreground/80 px-1">
                Results ({results.length})
            </h2>
            <ScrollArea className="flex-1 pr-4">
                <div className="space-y-3 pb-4">
                    {results.map((result, index) => (
                        <Card
                            key={index}
                            onClick={() => onSelectDocument(result)}
                            className={`cursor-pointer transition-all hover:bg-accent/50 ${selectedDocument === result ? 'border-primary ring-1 ring-primary bg-accent' : ''
                                }`}
                        >
                            <CardHeader className="p-4 pb-2">
                                <CardTitle className="text-base leading-snug line-clamp-2">
                                    {result.title}
                                </CardTitle>
                            </CardHeader>
                            <CardContent className="p-4 pt-0">
                                <div className="flex items-center text-sm text-muted-foreground mt-2 space-x-4">
                                    <div className="flex items-center">
                                        <span className="font-mono bg-muted px-1.5 py-0.5 rounded text-xs">
                                            {result.codice_redazionale}
                                        </span>
                                    </div>
                                    <div className="flex items-center">
                                        <CalendarIcon className="mr-1 h-3 w-3" />
                                        <span className="text-xs">{result.data_pubblicazione_gazzetta}</span>
                                    </div>
                                </div>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </ScrollArea>
        </div>
    );
}
