'use client';

import { Button } from "@/components/ui/button"
import { Trash2 } from "lucide-react"

interface AnnotationPanelProps {
    annotations: any[];
    onAnnotationClick: (id: number) => void;
    onDeleteAnnotation: (id: number) => void;
}
export default function AnnotationPanel({
    annotations,
    onAnnotationClick,
    onDeleteAnnotation
}: AnnotationPanelProps) {
    return (
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
    );
}

