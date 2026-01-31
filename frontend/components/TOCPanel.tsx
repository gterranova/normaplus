'use client';

import { ScrollArea } from "@/components/ui/scroll-area"
import { List, ChevronDown, ChevronRight } from "lucide-react"
import { useState } from "react"

interface TOCItem {
    level: number;
    text: string;
}

interface TOCPanelProps {
    toc: TOCItem[];
    onSelectSection: (index: number) => void;
    activeIndex?: number;
}

export default function TOCPanel({ toc, onSelectSection, activeIndex }: TOCPanelProps) {
    if (toc.length === 0) return null;

    return (
        <div className="p-3 text-xs space-y-1">
            {toc.length === 0 ? (
                <p className="text-center py-8 text-muted-foreground italic">No contents found</p>
            ) : (
                toc.map((item, index) => {
                    if (item.level > 4) return null;
                    return (
                        <button
                            key={index}
                            onClick={() => onSelectSection(index)}
                            className={`w-full text-left py-1.5 px-3 rounded hover:bg-primary/5 transition-all truncate group relative
                                                ${item.level === 1 ? 'font-bold text-foreground' : ''}
                                                ${item.level === 2 ? 'pl-4 text-muted-foreground' : ''}
                                                ${item.level === 3 ? 'pl-6 text-muted-foreground/80' : ''}
                                                ${item.level >= 4 ? 'pl-8 text-muted-foreground/70' : ''}
                                                ${activeIndex === index ? 'bg-primary/10 text-primary font-bold shadow-sm' : ''}
                                            `}
                            title={item.text}
                        >
                            {item.text}
                        </button>
                    );
                })
            )}
        </div>
    );
}
