'use client';

import { useState, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import { ScrollArea } from "@/components/ui/scroll-area"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Eye, FileCode, Loader2 } from "lucide-react"

interface DocumentViewProps {
    document: any;
    onNavigate: (urn: string) => void;
}

export default function DocumentView({ document, onNavigate }: DocumentViewProps) {
    const [content, setContent] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [format, setFormat] = useState<'markdown' | 'xml'>('markdown');

    useEffect(() => {
        const fetchDocument = async () => {
            setLoading(true);
            setError('');

            try {
                // Fetch document content based on props
                const url = `http://localhost:8080/api/document?id=${encodeURIComponent(document.codice_redazionale)}&date=${encodeURIComponent(document.data_pubblicazione_gazzetta)}&format=${format}`;
                const response = await fetch(url);

                if (!response.ok) throw new Error('Failed to fetch document');

                const text = await response.text();
                setContent(text);
            } catch (err) {
                setError('Failed to load document content');
                setContent('');
            } finally {
                setLoading(false);
            }
        };

        fetchDocument();
    }, [document, format]); // Re-fetch when document prop changes (handled by parent history)

    const LinkRenderer = (props: any) => {
        const href = props.href || '';
        // Looser check: includes normattiva link instead of exact startsWith
        if (href.includes('normattiva.it/uri-res/N2Ls')) {
            return (
                <a
                    href={href}
                    onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation(); // Stop bubbling
                        const parts = href.split('?');
                        // Handle both ?urn=... and direct query
                        const urnPart = parts.length > 1 ? parts[1] : '';
                        if (urnPart) {
                            // If urn= prefix exists, clean it? Our converter just appends the URN.
                            // The URL is usually .../N2Ls?urn:nir...
                            // So parts[1] is "urn:nir..."
                            onNavigate(urnPart);
                        } else {
                            console.warn("Could not extract URN from link:", href);
                        }
                    }}
                    className="text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary transition-colors font-medium cursor-pointer"
                >
                    {props.children}
                </a>
            );
        }
        return <a {...props} className="text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary transition-colors" target="_blank" rel="noopener noreferrer" />;
    };

    return (
        <div className="h-full flex flex-col">
            <div className="flex justify-between items-start mb-4 px-1 shrink-0">
                <div>
                    <h2 className="text-xl font-serif font-bold text-foreground leading-tight line-clamp-2">
                        {document?.title || 'Document'}
                    </h2>
                    <p className="text-xs text-muted-foreground font-mono mt-1">
                        ID: {document.codice_redazionale} • Data: {document.data_pubblicazione_gazzetta}
                    </p>
                </div>
                <div className="bg-muted p-1 rounded-lg flex space-x-1 shrink-0 ml-4">
                    <Button
                        variant={format === 'markdown' ? 'default' : 'ghost'}
                        size="sm"
                        onClick={() => setFormat('markdown')}
                        className="h-8 text-xs"
                    >
                        <Eye className="mr-2 h-3 w-3" />
                        Reader
                    </Button>
                    <Button
                        variant={format === 'xml' ? 'default' : 'ghost'}
                        size="sm"
                        onClick={() => setFormat('xml')}
                        className="h-8 text-xs"
                    >
                        <FileCode className="mr-2 h-3 w-3" />
                        XML
                    </Button>
                </div>
            </div>

            <Card className="flex-1 overflow-hidden bg-background/50 backdrop-blur-sm border-muted shadow-sm flex flex-col min-h-0">
                {loading ? (
                    <div className="h-full flex flex-col items-center justify-center text-muted-foreground">
                        <Loader2 className="h-10 w-10 animate-spin mb-4 opacity-50" />
                        <p>Loading content...</p>
                    </div>
                ) : error ? (
                    <div className="h-full flex items-center justify-center p-8 text-destructive">
                        {error}
                    </div>
                ) : (
                    <ScrollArea className="flex-1 w-full h-[100px]"> {/* h-[100px] is dummy to force flex growth */}
                        <div className="p-8 md:p-12 max-w-4xl mx-auto">
                            {format === 'markdown' ? (
                                <div className="prose prose-slate dark:prose-invert max-w-none 
                                    prose-headings:font-serif prose-headings:font-bold
                                    prose-h1:text-4xl prose-h1:mb-8 prose-h1:text-primary
                                    prose-h2:text-2xl prose-h2:mt-10 prose-h2:mb-4 prose-h2:border-b prose-h2:pb-2
                                    prose-p:leading-relaxed prose-p:text-foreground/90 prose-p:text-lg
                                    prose-li:text-foreground/90">
                                    <ReactMarkdown components={{ a: LinkRenderer }}>{content}</ReactMarkdown>
                                </div>
                            ) : (
                                <pre className="text-xs font-mono whitespace-pre-wrap break-words text-muted-foreground bg-muted/30 p-4 rounded-md border">
                                    {content}
                                </pre>
                            )}
                        </div>
                    </ScrollArea>
                )}
            </Card>
        </div>
    );
}
