'use client';

import { ScrollArea } from "@/components/ui/scroll-area"
import { List, MessageCircle, PanelRightOpen, Trash2, PanelRightClose, Link2 } from "lucide-react"
import { useState } from "react"
import { Button } from "@/components/ui/button"
import TOCPanel from "./TOCPanel";
import AnnotationPanel from "./AnnotationPanel";
import RecepimentoPanel from "./RecepimentoPanel";

interface DocumentSidebarProps {
    toc: any[];
    annotations: any[];
    onSelectSection: (index: number) => void;
    onAnnotationClick: (id: number) => void;
    onDeleteAnnotation: (id: number) => void;
    activeIndex?: number;
    docData: any;
    onOpenEU: (celex: string, title: string) => void;
    onSearchNormattiva: (query: string) => void;
}

export default function DocumentSidebar({
    toc,
    annotations,
    onSelectSection,
    onAnnotationClick,
    onDeleteAnnotation,
    activeIndex,
    docData,
    onOpenEU,
    onSearchNormattiva
}: DocumentSidebarProps) {
    const [activeTab, setActiveTab] = useState<'toc' | 'notes' | 'eu'>('toc');
    const [collapsed, setCollapsed] = useState(false);

    return (
        <div className={`h-full flex flex-col bg-card/50 backdrop-blur-sm border-border transition-all duration-300 ${collapsed ? 'w-12 overflow-hidden' : 'w-80'}`}>
            <div className="flex justify-between items-center px-1 pb-2 mb-2 border-b border-border/50">
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setCollapsed(!collapsed)}
                    className={`h-8 w-8 text-muted-foreground ${collapsed ? 'mx-auto' : 'ml-auto'}`}
                >
                    {collapsed ? <PanelRightOpen className="h-4 w-4" /> : <PanelRightClose className="h-4 w-4" />}
                </Button>
                {!collapsed && (
                    <div className="flex w-full">
                        <div className="flex items-center space-x-4 pl-4">
                            <button
                                className={`flex items-center text-sm font-semibold transition-colors pb-1 border-b-2 ${activeTab === 'toc' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
                                onClick={() => setActiveTab('toc')}
                            >
                                <List className="h-3 w-3 mr-1.5" />
                                Contents
                            </button>
                            <button
                                className={`flex items-center text-sm font-semibold transition-colors pb-1 border-b-2 ${activeTab === 'notes' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
                                onClick={() => setActiveTab('notes')}
                            >
                                <MessageCircle className="h-3 w-3 mr-1.5" />
                                Notes ({annotations?.length || 0})
                            </button>
                            {/* A tool about the open document, so it belongs beside the
                                contents and the notes rather than in the search column —
                                and it is only queried once the tab is selected, because
                                each lookup is a slow SPARQL query against CELLAR. */}
                            <button
                                className={`flex items-center text-sm font-semibold transition-colors pb-1 border-b-2 ${activeTab === 'eu' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
                                onClick={() => setActiveTab('eu')}
                                title="Transposition between EU and national law"
                            >
                                <Link2 className="h-3 w-3 mr-1.5" />
                                EU
                            </button>
                        </div>
                    </div>
                )}
            </div>

            {!collapsed && (
                <ScrollArea className="flex-1">
                    {activeTab === 'toc' && (
                        <TOCPanel toc={toc} onSelectSection={onSelectSection} activeIndex={activeIndex} />
                    )}
                    {activeTab === 'notes' && (
                        <AnnotationPanel annotations={annotations} onAnnotationClick={onAnnotationClick} onDeleteAnnotation={onDeleteAnnotation} />
                    )}
                    {activeTab === 'eu' && (
                        <RecepimentoPanel docData={docData} onOpenEU={onOpenEU} onSearchNormattiva={onSearchNormattiva} />
                    )}
                </ScrollArea>
            )}
        </div>
    );
}
