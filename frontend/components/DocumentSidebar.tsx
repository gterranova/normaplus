'use client';

import { ScrollArea } from "@/components/ui/scroll-area"
import { List, MessageCircle, ChevronRight, Trash2, X } from "lucide-react"
import { useState } from "react"
import { Button } from "@/components/ui/button"

interface DocumentSidebarProps {
    toc: any[];
    annotations: any[];
    onSelectSection: (index: number) => void;
    onAnnotationClick: (id: number) => void;
    onDeleteAnnotation: (id: number) => void;
    activeIndex?: number;
}

export default function DocumentSidebar({
    toc,
    annotations,
    onSelectSection,
    onAnnotationClick,
    onDeleteAnnotation,
    activeIndex
}: DocumentSidebarProps) {
    const [activeTab, setActiveTab] = useState<'toc' | 'notes'>('toc');
    const [collapsed, setCollapsed] = useState(false);

    return (
        <div className={`h-full flex flex-col bg-card/50 backdrop-blur-sm border-border transition-all duration-300 ${collapsed ? 'w-10 overflow-hidden' : 'w-80'}`}>
            <div className="flex items-center justify-between p-2 border-b border-border h-12 shrink-0">
                {!collapsed && (
                    <div className="flex bg-muted/50 p-1 rounded-md">
                        <Button
                            variant={activeTab === 'toc' ? 'secondary' : 'ghost'}
                            size="sm"
                            className="h-7 text-[10px] px-2 shadow-none uppercase font-bold tracking-wider"
                            onClick={() => setActiveTab('toc')}
                        >
                            <List className="h-3 w-3 mr-1.5" />
                            Contents
                        </Button>
                        <Button
                            variant={activeTab === 'notes' ? 'secondary' : 'ghost'}
                            size="sm"
                            className="h-7 text-[10px] px-2 shadow-none uppercase font-bold tracking-wider"
                            onClick={() => setActiveTab('notes')}
                        >
                            <MessageCircle className="h-3 w-3 mr-1.5" />
                            Notes ({annotations?.length || 0})
                        </Button>
                    </div>
                )}
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setCollapsed(!collapsed)}
                    className="h-8 w-8 text-muted-foreground ml-auto"
                >
                    {collapsed ? <List className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                </Button>
            </div>

            {!collapsed && (
                <ScrollArea className="flex-1">
                    {activeTab === 'toc' ? (
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
                    ) : (
                        <div className="p-3 space-y-3">
                            {(annotations?.length || 0) === 0 ? (
                                <p className="text-xs text-muted-foreground text-center py-12 italic">No annotations for this document</p>
                            ) : (
                                annotations.map(ann => (
                                    <div
                                        key={ann.id}
                                        className="p-3 rounded-lg border bg-background hover:border-primary/50 transition-all cursor-pointer group shadow-sm active:scale-[0.98]"
                                        onClick={() => onAnnotationClick(ann.id)}
                                    >
                                        <div className="flex justify-between items-start mb-2">
                                            <p className="text-[10px] font-bold text-primary uppercase tracking-wider">Note #{ann.id}</p>
                                            <Button
                                                size="sm"
                                                variant="ghost"
                                                className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 text-destructive hover:bg-destructive/10 transition-opacity"
                                                onClick={(e) => { e.stopPropagation(); onDeleteAnnotation(ann.id); }}
                                            >
                                                <Trash2 className="h-3.5 w-3.5" />
                                            </Button>
                                        </div>
                                        <p className="text-[13px] font-medium leading-relaxed italic border-l-2 border-primary/20 pl-2 mb-3 break-words text-foreground/80">
                                            "{ann.selection_data}"
                                        </p>
                                        <div className="text-[13px] text-foreground leading-relaxed bg-muted/30 p-2.5 rounded-md border border-border/50">
                                            {ann.comment}
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>
                    )}
                </ScrollArea>
            )}
        </div>
    );
}
