'use client';

import { ScrollArea } from "@/components/ui/scroll-area"
import { List, ChevronDown, ChevronRight } from "lucide-react"
import { useState } from "react"

interface TOCItem {
    level: number;
    text: string;
}

interface TOCSidebarProps {
    toc: TOCItem[];
    onSelectSection: (index: number) => void;
    activeIndex?: number;
}

export default function TOCSidebar({ toc, onSelectSection, activeIndex }: TOCSidebarProps) {
    const [collapsed, setCollapsed] = useState(false);

    if (toc.length === 0) return null;

    return (
        <div className={`h-full flex flex-col bg-card/50 backdrop-blur-sm border-l border-border transition-all duration-300 ${collapsed ? 'w-10 overflow-hidden' : 'w-64'}`}>
            <div className="flex items-center justify-between p-3 border-b border-border h-12">
                <div className={`flex items-center text-sm font-semibold text-muted-foreground ${collapsed ? 'hidden' : 'block'}`}>
                    <List className="h-4 w-4 mr-2" />
                    Contents
                </div>
                <button
                    onClick={() => setCollapsed(!collapsed)}
                    className="p-1 hover:bg-accent rounded-md text-muted-foreground"
                >
                    {collapsed ? <List className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                </button>
            </div>

            {!collapsed && (
                <ScrollArea className="flex-1">
                    <div className="p-3 text-xs space-y-1">
                        {toc.map((item, index) => {
                            if (item.level > 4) return null; // Only show up to H4
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
                        })}
                    </div>
                </ScrollArea>
            )}
        </div>
    );
}
