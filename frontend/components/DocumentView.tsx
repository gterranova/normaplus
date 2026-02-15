'use client';

import { useState, useEffect, useRef, useCallback, memo } from 'react';
import ReactMarkdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import { ScrollArea } from "@/components/ui/scroll-area"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Eye, FileCode, Loader2, MessageCircle, Sparkles, Languages, Download, X, Trash2 } from "lucide-react"
import { useUser } from '@/components/UserProvider';

interface DocumentViewProps {
    docData: any;
    onNavigate: (urn: string) => void;
    onTOCParsed?: (toc: any[]) => void;
    onActiveSectionChange?: (id: string) => void;
    annotations: any[];
    onAnnotationAction: () => void;
}

interface SelectionState {
    x: number;
    y: number;
    text: string;
    locationId?: string;
    offset?: number;
    prefix?: string;
    suffix?: string;
    existingId?: number;
    initialComment?: string;
}

// Optimized Editor to prevent document re-renders during typing
const AnnotationEditor = memo(({
    selection,
    onSave,
    onDelete,
    onCancel,
    aiLoading,
    handleAIAction
}: {
    selection: SelectionState,
    onSave: (text: string) => void,
    onDelete?: (id: number) => void,
    onCancel: () => void,
    aiLoading: boolean,
    handleAIAction: (action: 'summarize' | 'translate') => void
}) => {
    const [input, setInput] = useState(selection.initialComment || '');
    const [showInput, setShowInput] = useState(!!selection.existingId || !!selection.initialComment);
    const [pos, setPos] = useState({ x: selection.x, y: selection.y });
    const [size, setSize] = useState({ w: 320, h: 'auto' as number | 'auto' });
    const editorRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (selection.initialComment) {
            setInput(selection.initialComment);
            setShowInput(true);
        }
    }, [selection.initialComment]);

    // Dragging Logic
    const startDrag = (e: React.MouseEvent) => {
        const startX = e.clientX - pos.x;
        const startY = e.clientY - pos.y;

        const onMouseMove = (moveEvent: MouseEvent) => {
            setPos({
                x: moveEvent.clientX - startX,
                y: moveEvent.clientY - startY
            });
        };

        const onMouseUp = () => {
            document.removeEventListener('mousemove', onMouseMove);
            document.removeEventListener('mouseup', onMouseUp);
        };

        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);
    };

    // Resizing Logic
    const startResize = (e: React.MouseEvent) => {
        e.stopPropagation();
        const startWidth = editorRef.current?.offsetWidth || 320;
        const startHeight = editorRef.current?.offsetHeight || 200;
        const startX = e.clientX;
        const startY = e.clientY;

        const onMouseMove = (moveEvent: MouseEvent) => {
            setSize({
                w: Math.max(280, startWidth + (moveEvent.clientX - startX)),
                h: Math.max(120, startHeight + (moveEvent.clientY - startY))
            });
        };

        const onMouseUp = () => {
            document.removeEventListener('mousemove', onMouseMove);
            document.removeEventListener('mouseup', onMouseUp);
        };

        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);
    };

    return (
        <div
            ref={editorRef}
            className="annotation-editor fixed z-50 bg-popover text-popover-foreground shadow-2xl rounded-xl border border-border flex flex-col animate-in fade-in zoom-in-95 duration-200 overflow-hidden"
            style={{
                left: pos.x,
                top: pos.y,
                width: size.w,
                height: size.h === 'auto' ? 'auto' : size.h,
                transform: 'translate(-50%, -100%)'
            }}
        >
            {/* Drag Handle */}
            <div
                onMouseDown={startDrag}
                className="h-2 w-full bg-muted/20 hover:bg-primary/20 cursor-move flex items-center justify-center transition-colors group"
            >
                <div className="w-8 h-1 bg-border rounded-full group-hover:bg-primary/40" />
            </div>

            <div className="p-3 pt-1 flex flex-col gap-2 relative flex-1 min-h-0">
                {aiLoading ? (
                    <div className="flex items-center px-4 py-6 space-x-2 text-sm text-muted-foreground justify-center">
                        <Loader2 className="h-4 w-4 animate-spin text-primary" />
                        <span className="font-medium">AI is thinking...</span>
                    </div>
                ) : !showInput ? (
                    <div className="flex items-center space-x-1 p-1">
                        <Button size="sm" variant="ghost" className="h-8 px-3 rounded-md hover:bg-primary/5 hover:text-primary transition-all" onClick={() => setShowInput(true)}>
                            <MessageCircle className="h-4 w-4 mr-2" /> Note
                        </Button>
                        <div className="w-px h-4 bg-border mx-1"></div>
                        <Button size="sm" variant="ghost" className="h-8 px-3 text-indigo-500 hover:text-indigo-600 hover:bg-indigo-50 rounded-md transition-all font-medium" onClick={() => handleAIAction('summarize')}>
                            <Sparkles className="h-4 w-4 mr-2" /> Summarize
                        </Button>
                        <Button size="sm" variant="ghost" className="h-8 px-3 text-blue-500 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-all font-medium" onClick={() => handleAIAction('translate')}>
                            <Languages className="h-4 w-4 mr-2" /> Translate
                        </Button>
                    </div>
                ) : (
                    <div className="flex flex-col gap-2 flex-1 min-h-0">
                        <div className="flex justify-between items-center px-1">
                            <p className="text-[10px] text-muted-foreground font-bold uppercase tracking-wider">
                                {selection.existingId ? 'Edit Note' : 'New Annotation'}
                            </p>
                            {selection.existingId && onDelete && (
                                <Button
                                    size="sm"
                                    variant="ghost"
                                    className="h-6 w-6 p-0 text-destructive hover:bg-destructive/10"
                                    onClick={() => onDelete(selection.existingId!)}
                                    title="Delete Annotation"
                                >
                                    <Trash2 className="h-3.5 w-3.5" />
                                </Button>
                            )}
                        </div>
                        <textarea
                            autoFocus
                            className="w-full text-sm bg-background/50 border border-border p-3 rounded-lg flex-1 min-h-[120px] resize-none leading-relaxed focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all shadow-inner"
                            placeholder="Type or paste your legal notes here..."
                            value={input}
                            onChange={(e) => setInput(e.target.value)}
                        />
                        <div className="flex justify-end gap-2 mt-1">
                            <Button size="sm" variant="ghost" className="h-8 text-xs px-4" onClick={onCancel}>Cancel</Button>
                            <Button size="sm" className="h-8 text-xs px-6 shadow-md shadow-primary/20" onClick={() => onSave(input)}>Save Note</Button>
                        </div>
                    </div>
                )}

                {/* Resize Handle */}
                {showInput && !aiLoading && (
                    <div
                        onMouseDown={startResize}
                        className="absolute bottom-0 right-0 w-4 h-4 cursor-nwse-resize flex items-end justify-end p-0.5 group"
                    >
                        <div className="w-2 h-2 border-r-2 border-b-2 border-muted-foreground/30 group-hover:border-primary transition-colors" />
                    </div>
                )}
            </div>
        </div>
    );
});

AnnotationEditor.displayName = 'AnnotationEditor';

export default function DocumentView({ docData, onNavigate, onTOCParsed, onActiveSectionChange, annotations, onAnnotationAction }: DocumentViewProps) {
    const [content, setContent] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [format, setFormat] = useState<'markdown' | 'xml'>('markdown');
    const [vigenza, setVigenza] = useState<string>('');
    const [aiLoading, setAiLoading] = useState(false);
    const scrollRef = useRef<HTMLDivElement>(null);

    const { user } = useUser();
    const [selection, setSelection] = useState<SelectionState | null>(null);

    const parseTOC = useCallback((md: string) => {
        const toc = [];
        const lines = md.split('\n');
        let lastId = '';
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i].trim();
            const idMatch = line.match(/^<span id="([^"]+)"><\/span>$/);
            if (idMatch) { lastId = idMatch[1]; continue; }
            const headerMatch = line.match(/^(#{1,6})\s+(.*)$/);
            if (headerMatch) {
                const level = headerMatch[1].length;
                let text = headerMatch[2].replace(/[\*_`\[\]\(\)#~]/g, '').trim();
                toc.push({ level, text, id: lastId });
                lastId = '';
            } else if (line !== '') lastId = '';
        }
        return toc;
    }, []);

    // Reset vigenza when switching documents
    useEffect(() => {
        setVigenza('');
    }, [docData?.codice_redazionale, docData?.data_pubblicazione_gazzetta]);

    // Fetch Content
    useEffect(() => {
        const fetchDocument = async () => {
            setLoading(true);
            setError('');
            try {
                const url = `http://localhost:8080/api/document?id=${encodeURIComponent(docData.codice_redazionale)}&date=${encodeURIComponent(docData.data_pubblicazione_gazzetta)}&format=${format}&vigenza=${vigenza}`;
                const response = await fetch(url);
                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || 'Failed to fetch document');
                }
                const text = await response.text();
                setContent(text);
                if (format === 'markdown' && onTOCParsed) {
                    onTOCParsed(parseTOC(text));
                }
            } catch (err: any) {
                setError(err.message || 'Failed to load document content');
                setContent('');
            } finally {
                setLoading(false);
            }
        };
        if (docData?.codice_redazionale) fetchDocument();
    }, [docData?.codice_redazionale, docData?.data_pubblicazione_gazzetta, format, vigenza, onTOCParsed, parseTOC]);

    // Annotations fetched by parent, but we still need to provide an update trigger

    // IntersectionObserver for TOC Highlighting
    useEffect(() => {
        if (!content || !onActiveSectionChange || format !== 'markdown') return;
        const observer = new IntersectionObserver(
            (entries) => {
                const visible = entries.find(e => e.isIntersecting);
                if (visible) onActiveSectionChange(visible.target.id);
            },
            { threshold: 0, root: scrollRef.current }
        );
        const anchors = document.querySelectorAll('span[id]');
        anchors.forEach(a => observer.observe(a));
        return () => observer.disconnect();
    }, [content, onActiveSectionChange, format, scrollRef]);

    // Scroll to Fragment/Anchor
    useEffect(() => {
        if (docData?.urnFragment) {
            //console.log("DEBUG: Scrolling to", docData.urnFragment);
            // Use a small timeout or requestAnimationFrame to ensure ReactMarkdown has finished rendering
            const timer = setTimeout(() => {
                const el = document.getElementById(docData.urnFragment);
                if (el) {
                    el.scrollIntoView({ behavior: 'smooth', block: 'start' });

                    let paragraph = el.parentElement?.nextElementSibling as HTMLElement;

                    let elArray = [];
                    while (true) {
                        elArray.push(paragraph);

                        // Remove then add back to re-trigger animation if already there
                        paragraph.classList.remove('animate-highlight');
                        void paragraph.offsetWidth; // Trigger reflow
                        paragraph.classList.add('animate-highlight');

                        paragraph = paragraph.nextElementSibling as HTMLElement;
                        if (!paragraph || paragraph.tagName !== 'P') break;
                    }

                    setTimeout(() => {
                        elArray.forEach(el => {
                            el.classList.remove('animate-highlight');
                            void el.offsetWidth; // Trigger reflow
                        });
                    }, 2000);

                } else {
                    //console.warn("DEBUG: Element not found for scrolling", docData.urnFragment);
                }
            }, 100);
            return () => clearTimeout(timer);
        }
    }, [docData?.urnFragment]);

    // Selection Capture
    useEffect(() => {
        const handleMouseUp = (e: MouseEvent) => {
            const sel = window.getSelection();

            // Don't clear if clicking inside the editor or on a mark/icon
            const isClickInside = (e.target as HTMLElement).closest('.annotation-editor') ||
                (e.target as HTMLElement).closest('.ann-highlight') ||
                (e.target as HTMLElement).closest('.ann-icon');

            if (isClickInside) {
                //console.log("DEBUG: Mouseup inside editor/icon, not clearing.");
                return;
            }

            if (sel && sel.toString().trim().length > 0) {
                const range = sel.getRangeAt(0);
                const rect = range.getBoundingClientRect();

                let locationId = '';
                let node: Node | null = sel.anchorNode;

                // Better ID lookup: Traverse up and look at previous siblings
                let curr: HTMLElement | null = node instanceof HTMLElement ? node : node?.parentElement as HTMLElement;

                // find the first parent element with a P tag
                while (curr && curr.tagName !== 'P' && curr !== scrollRef.current && curr !== document.body) {
                    curr = curr.parentElement;
                }

                // find the first sibling with an ID or a child with an ID
                while (curr && curr !== scrollRef.current && curr !== document.body) {
                    if (curr.id) {
                        locationId = curr.id;
                        break;
                    }

                    // check children of current element
                    const childAnchor = curr.querySelector('span[id]');
                    if (childAnchor) {
                        locationId = childAnchor.id;
                        break;
                    }

                    // Look back at siblings ONLY if we are at a level that should have IDs (like spans/headers)
                    // or just check all siblings as we go up.
                    curr = curr.previousElementSibling as HTMLElement;
                }

                // Fallback: If no ID found, try to find the absolute first anchor in the document
                if (!locationId) {
                    const firstAnchor = scrollRef.current?.querySelector('span[id]');
                    if (firstAnchor) locationId = firstAnchor.id;
                }

                if (scrollRef.current && scrollRef.current.contains(sel.anchorNode)) {
                    // Context Fingerprinting: Capture text before and after utilizing the actual Range
                    const preRange = range.cloneRange();
                    preRange.selectNodeContents(scrollRef.current);
                    preRange.setEnd(range.startContainer, range.startOffset);
                    const prefix = preRange.toString().slice(-60);

                    const postRange = range.cloneRange();
                    postRange.selectNodeContents(scrollRef.current);
                    postRange.setStart(range.endContainer, range.endOffset);
                    const suffix = postRange.toString().slice(0, 60);

                    //console.log("DEBUG: Context Captured", { prefix, selection: sel.toString(), suffix });

                    setSelection({
                        x: rect.left + rect.width / 2,
                        y: rect.top - 10,
                        text: sel.toString().trim(),
                        locationId,
                        offset: range.startOffset,
                        prefix,
                        suffix
                    });
                }
            } else if (!selection?.existingId) {
                setSelection(null);
            }
        };
        document.addEventListener('mouseup', handleMouseUp);
        return () => document.removeEventListener('mouseup', handleMouseUp);
    }, [selection]);

    const handleSaveAnnotation = async (comment: string) => {
        if (!user || !selection || !comment.trim()) return;
        //console.log("DEBUG: Saving annotation", { selection, comment });
        try {
            const method = selection.existingId ? 'PUT' : 'POST';
            const body = selection.existingId
                ? { id: selection.existingId, comment }
                : {
                    user_id: user.id,
                    doc_id: docData.codice_redazionale,
                    selection_data: selection.text,
                    location_id: selection.locationId || '',
                    selection_offset: selection.offset || 0,
                    prefix: selection.prefix || '',
                    suffix: selection.suffix || '',
                    comment
                };

            const res = await fetch(`http://localhost:8080/api/annotations`, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body)
            });
            if (res.ok) {
                //console.log("DEBUG: Save successful");
                onAnnotationAction();
                setSelection(null);
                window.getSelection()?.removeAllRanges();
            } else {
                const errText = await res.text();
                //console.error("DEBUG: Save failed", res.status, errText);
                alert(`Failed to save note: ${errText}`);
            }
        } catch (e) {
            //console.error("DEBUG: Save error", e);
            alert("Error connecting to server");
        }
    };

    const handleDeleteAnnotation = async (id: number) => {
        try {
            const res = await fetch(`http://localhost:8080/api/annotations?id=${id}`, { method: 'DELETE' });
            if (res.ok) {
                onAnnotationAction();
                setSelection(null);
            }
        } catch (e) { console.error(e); }
    };

    const handleAIAction = async (action: 'summarize' | 'translate') => {
        if (!selection) return;
        setAiLoading(true);
        try {
            const res = await fetch(`http://localhost:8080/api/ai/generate`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action, text: selection.text })
            });
            if (res.ok) {
                const data = await res.json();
                setSelection(prev => prev ? { ...prev, initialComment: data.result } : null);
            }
        } catch (e) { console.error(e); } finally { setAiLoading(false); }
    };

    const handleAnnotationClick = (id: string | number) => {
        const ann = annotations.find(a => String(a.id) === String(id));
        if (ann) {
            const el = document.querySelector(`mark[data-id="${id}"]`);
            const rect = el?.getBoundingClientRect();
            //console.log("DEBUG: Annotation clicked", { id, ann, rect });
            setSelection({
                x: rect ? rect.left + rect.width / 2 : window.innerWidth / 2,
                y: rect ? rect.top - 10 : window.innerHeight / 2,
                text: ann.selection_data,
                existingId: ann.id,
                initialComment: ann.comment
            });
        }
    };

    const isAlphaNumeric = (char: string) => {
        // Robust Unicode check for letters and numbers (essential for Italian accents like à, è, ò)
        return /\p{L}|\p{N}|\s/u.test(char);
    };

    const cleanMarkdown = (text: string) => {
        // Remove tags and keep only alphanumeric chars (Unicode aware)
        return Array.from(text.replace(/<[^>]*>/g, ''))
            .filter(isAlphaNumeric)
            .join('')
            .toLowerCase();
    };

    const findMarkdownRange = (prefix: string, selection: string, suffix: string, text: string) => {
        // 1. Create a precise map from 'clean' (alphanumeric only) indices to 'raw' indices
        const cleanToRaw: number[] = [];
        let cleanText = "";
        let withinTag = false;

        for (let i = 0; i < text.length; i++) {
            const char = text[i];
            if (char === '<') withinTag = true;

            // Map character to the clean stream if it's a letter or number outside a tag
            if (!withinTag && isAlphaNumeric(char)) {
                cleanText += char.toLowerCase();
                cleanToRaw.push(i);
            }

            if (char === '>') withinTag = false;
        }

        const cPre = cleanMarkdown(prefix);
        const cSel = cleanMarkdown(selection);
        const cSuf = cleanMarkdown(suffix);

        if (!cSel) return null;

        // 2. Search for the best contextual match
        let bestMatch = { preStart: -1, preEnd: -1, selStart: -1, selEnd: -1, sufStart: -1, sufEnd: -1, score: -1 };
        let searchPos = cleanText.indexOf(cSel, 0);

        while (searchPos !== -1) {
            let score = 0;
            const preStart = Math.max(0, searchPos - cPre.length);
            const actualPre = cleanText.slice(preStart, searchPos);

            if (cPre && actualPre === cPre) score += 20; // High weight for exact context
            else if (cPre && actualPre.includes(cPre)) score += 5;

            const sufEnd = Math.min(cleanText.length, searchPos + cSel.length + cSuf.length);
            const actualSuf = cleanText.slice(searchPos + cSel.length, sufEnd);

            if (cSuf && actualSuf === cSuf) score += 20;
            else if (cSuf && actualSuf.includes(cSuf)) score += 5;

            if (score > bestMatch.score || (score === bestMatch.score && bestMatch.score === -1)) {
                bestMatch = {
                    preStart, preEnd: searchPos,
                    selStart: searchPos, selEnd: searchPos + cSel.length,
                    sufStart: searchPos + cSel.length, sufEnd,
                    score
                };
            }
            searchPos = cleanText.indexOf(cSel, searchPos + 1);
        }

        if (bestMatch.score === -1) return null;

        // 3. Resolve base raw boundaries
        let rawStart = cleanToRaw[bestMatch.selStart];
        let rawEnd = cleanToRaw[bestMatch.selEnd - 1] + 1;

        // 4. "Ironclad" Boundary Refinement (Greedy Expansion)
        // Shift start backwards to swallow adjacent formatting (markers, tags) until we hit 
        // the end of the logical prefix match or a block break.
        const limitStart = bestMatch.preEnd > 0 ? cleanToRaw[bestMatch.preEnd - 1] + 1 : 0;
        let expandedStart = rawStart;
        while (expandedStart > limitStart) {
            const char = text[expandedStart - 1];
            if (char === '\n') break; // Never cross block boundaries
            if (isAlphaNumeric(char)) {
                const preRawEndLimit = cleanToRaw[bestMatch.preEnd - 1];
                if (expandedStart - 1 <= preRawEndLimit) break;
            }
            if (char === '>') {
                const tagOpen = text.lastIndexOf('<', expandedStart - 1);
                if (tagOpen !== -1 && tagOpen >= limitStart) {
                    expandedStart = tagOpen;
                    continue;
                }
            }
            expandedStart--;
        }

        // Shift end forwards to swallow adjacent formatting up to the suffix match
        const limitEnd = bestMatch.sufStart < cleanToRaw.length ? cleanToRaw[bestMatch.sufStart] : text.length;
        let expandedEnd = rawEnd;
        while (expandedEnd < limitEnd) {
            const char = text[expandedEnd];
            if (char === '\n') break;
            if (isAlphaNumeric(char)) {
                const sufRawStartLimit = cleanToRaw[bestMatch.sufStart];
                if (expandedEnd >= sufRawStartLimit) break;
            }
            if (char === '<') {
                const tagClose = text.indexOf('>', expandedEnd);
                if (tagClose !== -1 && tagClose < limitEnd) {
                    expandedEnd = tagClose + 1;
                    continue;
                }
            }
            expandedEnd++;
        }

        return { start: expandedStart, end: expandedEnd };
    };

    const getProcessedContent = () => {
        if (format !== 'markdown' || !content) return content;
        if (!annotations || annotations.length === 0) return content;

        // Sort annotations by their presence in the document to avoid jumping around?
        // Actually, we'll apply them in reverse order of their appearance to keep indices stable.
        const matches: { start: number, end: number, id: number, isLast: boolean }[] = [];

        annotations.forEach(ann => {
            const range = findMarkdownRange(ann.prefix, ann.selection_data, ann.suffix, content);
            if (range) {
                // Refine range to include leading/trailing non-alphanumeric characters 
                // that were part of the original selection but skipped by the mapping.
                const leadingNonAlphas = ann.selection_data.match(/^[^A-Za-z0-9]+/)?.[0].length || 0;
                const trailingNonAlphas = ann.selection_data.match(/[^A-Za-z0-9]+$/)?.[0].length || 0;

                const finalStart = Math.max(0, range.start - leadingNonAlphas);
                const finalEnd = Math.min(content.length, range.end + trailingNonAlphas);

                matches.push({ start: finalStart, end: finalEnd, id: ann.id, isLast: true });
            }
        });

        // Sort matches descending by start index to avoid index shifting during replacement
        matches.sort((a, b) => b.start - a.start);

        let result = content;
        matches.forEach(m => {
            const before = result.slice(0, m.start);
            const mid = result.slice(m.start, m.end);
            const after = result.slice(m.end);

            // Wrap mid in mark. We need to handle internal newlines by splitting mid?
            // ReactMarkdown handles <mark> fine across paragraphs if it's the only tag.
            // However, it's safer to wrap lines individually if they contain block-level breaks.
            const lines = mid.split(/\r?\n/);
            const wrappedMid = lines.map((line, idx) => {
                if (!line.trim()) return line;
                const isLastLine = idx === lines.length - 1;
                return `<mark class="ann-highlight" data-id="${m.id}">${line}</mark>${isLastLine ? `<span class="ann-icon" data-id="${m.id}"></span>` : ''}`;
            }).join('\n');

            result = before + wrappedMid + after;
        });

        return result;
    };

    const LinkRenderer = (props: any) => {
        const href = props.href || '';
        if (href.includes('normattiva.it/uri-res/N2Ls')) {
            return (
                <a href={href} onClick={(e) => {
                    e.preventDefault();
                    const urnPart = href.split('?')[1] || '';
                    if (urnPart) onNavigate(urnPart);
                }} className="text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary transition-colors font-bold cursor-pointer">
                    {props.children}
                </a>
            );
        }
        return <a {...props} className="text-secondary hover:text-primary transition-colors underline" target="_blank" rel="noopener noreferrer" />;
    };

    const AnnotationComponent = ({ node, ...props }: any) => {
        const id = props['data-id'] || props.dataId;
        return (
            <mark className="ann-highlight bg-amber-200/60 dark:bg-amber-800/40 dark:text-amber-200 cursor-pointer border-b-2 border-amber-400/50 dark:border-amber-500/50 hover:bg-amber-300 dark:hover:bg-amber-800/60 transition-colors shadow-sm" data-id={id} onClick={() => handleAnnotationClick(id)}>
                {props.children}
            </mark>
        );
    };

    const AnnotationIconComponent = ({ node, ...props }: any) => {
        const id = props['data-id'] || props.dataId;
        return (
            <span className="ann-icon inline-flex items-center justify-center w-5 h-5 ml-1 bg-primary/10 text-primary rounded-full cursor-pointer hover:bg-primary/20 transition-all scale-75 align-middle" data-id={id} onClick={() => handleAnnotationClick(id)} title="View Note">
                <MessageCircle className="h-3 w-3" />
            </span>
        );
    };

    return (
        <div className="h-full flex flex-col relative">
            <div className="px-2 mb-2">
                <h2 className="text-xl text-center font-serif font-bold text-foreground truncate" title={docData?.title}>
                    {docData?.title || 'Document'}
                </h2>
            </div>
            <div className="flex justify-center items-center mb-4 px-2 shrink-0 h-10 space-x-3">
                <div className="flex items-center bg-muted/30 rounded-lg p-1 border border-border/50">
                    <span className="text-[10px] text-muted-foreground uppercase font-bold px-2 border-r border-border/50 mr-1">Export</span>
                    <Button variant="ghost" size="sm" className="h-6 text-[10px] px-2 hover:text-primary transition-colors" onClick={() => window.open(`http://localhost:8080/api/export?id=${docData.codice_redazionale}&date=${docData.data_pubblicazione_gazzetta}&vigenza=${vigenza}&format=pdf`)}>PDF</Button>
                    <Button variant="ghost" size="sm" className="h-6 text-[10px] px-2 hover:text-primary transition-colors" onClick={() => window.open(`http://localhost:8080/api/export?id=${docData.codice_redazionale}&date=${docData.data_pubblicazione_gazzetta}&vigenza=${vigenza}&format=docx`)}>DOCX</Button>
                    <Button variant="ghost" size="sm" className="h-6 text-[10px] px-2 hover:text-primary transition-colors" onClick={() => window.open(`http://localhost:8080/api/export?id=${docData.codice_redazionale}&date=${docData.data_pubblicazione_gazzetta}&vigenza=${vigenza}&format=md`)}>MD</Button>
                </div>
                <div className="bg-muted/50 p-1 rounded-lg flex space-x-2 border items-center">
                    <div className="flex items-center px-2 space-x-2">
                        <span className="text-[10px] text-muted-foreground uppercase font-medium">Vigenza</span>
                        <input type="date" className="bg-transparent text-xs border-none focus:ring-0 p-0 h-6 w-26 font-mono text-muted-foreground focus:text-foreground" value={vigenza} onChange={(e) => setVigenza(e.target.value)} />
                    </div>
                </div>
                {/*
                <div className="bg-muted/50 p-1 rounded-lg flex space-x-2 border items-center">
                    <Button variant={format === 'markdown' ? 'secondary' : 'ghost'} size="sm" className="h-6 text-xs px-3 shadow-none" onClick={() => setFormat('markdown')}>
                        <Eye className="mr-2 h-3.5 w-3.5" /> Reader
                    </Button>
                    <Button variant={format === 'xml' ? 'secondary' : 'ghost'} size="sm" className="h-6 text-xs px-3 shadow-none" onClick={() => setFormat('xml')}>
                        <FileCode className="mr-2 h-3.5 w-3.5" /> XML
                    </Button>
                </div>
                */}
            </div>

            <div className="flex-1 flex overflow-hidden gap-4">
                <Card className="flex-1 overflow-hidden bg-[#fafaf8] dark:bg-[#0c0c0e] border-muted/30 flex flex-col min-h-0 rounded-md relative shadow-sm">
                    {loading ? (
                        <div className="h-full flex flex-col items-center justify-center text-muted-foreground">
                            <Loader2 className="h-10 w-10 animate-spin mb-4 opacity-50 text-primary" />
                            <p className="text-sm font-medium animate-pulse">Consulting archive...</p>
                        </div>
                    ) : error ? (
                        <div className="h-full flex items-center justify-center p-8 text-destructive italic">{error}</div>
                    ) : (
                        <ScrollArea className="flex-1 w-full" ref={scrollRef}>
                            <div className="p-10 max-w-[850px] mx-auto font-serif text-[19px] leading-relaxed text-[#1a1a1a] dark:text-[#e0e0e0]">
                                {format === 'markdown' ? (
                                    <div className="prose prose-zinc dark:prose-invert max-w-none">
                                        <ReactMarkdown rehypePlugins={[rehypeRaw]} components={{
                                            a: LinkRenderer, mark: AnnotationComponent,
                                            span: (props: any) => props.className === 'ann-icon' ? <AnnotationIconComponent {...props} /> : <span {...props} />
                                        }}>
                                            {getProcessedContent()}
                                        </ReactMarkdown>
                                    </div>
                                ) : (
                                    <pre className="text-xs font-mono whitespace-pre-wrap text-muted-foreground bg-muted/20 p-6 rounded-lg border border-dashed leading-relaxed">{content}</pre>
                                )}
                            </div>
                        </ScrollArea>
                    )}
                </Card>
            </div>

            {
                selection && (
                    <AnnotationEditor
                        selection={selection}
                        onSave={handleSaveAnnotation}
                        onDelete={handleDeleteAnnotation}
                        onCancel={() => setSelection(null)}
                        aiLoading={aiLoading}
                        handleAIAction={handleAIAction}
                    />
                )
            }
        </div>
    );
}
