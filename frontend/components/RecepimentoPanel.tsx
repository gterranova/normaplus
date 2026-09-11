'use client';

// Transposition: which Italian measures implement the directive on screen, and
// which EU acts the Italian act on screen implements.
//
// It is a panel about the CURRENT document rather than a search: the question
// only exists once something is open, and the answer is a property of that
// document. Which direction is asked therefore follows the document's source —
// the reader never picks one, because only one of the two is meaningful.
//
// The reference is editable. For an EU act it is the CELEX and is exact; for an
// Italian act it is the document's own title, which the server has to parse, and
// when that fails the server says what a valid citation looks like. Showing the
// field is what makes that message actionable instead of a dead end.

import { useCallback, useEffect, useState } from 'react';
import { Loader2, ExternalLink, Search, ArrowRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { isEU } from '@/lib/source';

interface RecepimentoPanelProps {
    docData: any;
    onOpenEU: (celex: string, title: string) => void;
    onSearchNormattiva: (query: string) => void;
}

export default function RecepimentoPanel({ docData, onOpenEU, onSearchNormattiva }: RecepimentoPanelProps) {
    const eu = isEU(docData);
    const verso = eu ? 'eu-it' : 'it-eu';
    const iniziale = (eu ? docData?.codice_redazionale : docData?.title) || '';

    const [rif, setRif] = useState<string>(iniziale);
    const [data, setData] = useState<any>(null);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const lookup = useCallback(async (reference: string) => {
        if (!reference.trim()) return;
        setLoading(true);
        setError('');
        setData(null);
        try {
            const res = await fetch(`/api/eu/recepimento?rif=${encodeURIComponent(reference.trim())}&verso=${verso}`);
            const text = await res.text();
            if (!res.ok) throw new Error(text || 'Lookup failed');
            setData(JSON.parse(text));
        } catch (e: any) {
            // The server's own message names either the archive that failed or
            // the citation form it needs, so it is shown verbatim rather than
            // replaced by a generic one.
            setError(e?.message || 'Lookup failed');
        } finally {
            setLoading(false);
        }
    }, [verso]);

    // Re-ask when the document changes, and only then: this is one SPARQL query
    // against CELLAR and takes seconds, so it must not fire on every keystroke
    // in the reference field.
    useEffect(() => {
        setRif(iniziale);
        setData(null);
        setError('');
        if (iniziale) lookup(iniziale);
    }, [iniziale, lookup]);

    return (
        <div className="p-3 space-y-3 text-sm">
            <form
                onSubmit={(e) => { e.preventDefault(); lookup(rif); }}
                className="flex items-center gap-1.5"
            >
                <input
                    value={rif}
                    onChange={(e) => setRif(e.target.value)}
                    placeholder={eu ? '32022L2555' : 'D.Lgs. 138/2024'}
                    className="flex-1 min-w-0 h-7 px-2 text-xs bg-background border border-input rounded font-mono focus:outline-none focus:ring-1 focus:ring-primary/30"
                />
                <Button type="submit" size="sm" variant="secondary" className="h-7 px-3 text-xs shrink-0" disabled={loading || !rif.trim()}>
                    {loading ? <Loader2 className="h-3 w-3 animate-spin" /> : 'Look up'}
                </Button>
            </form>

            <p className="text-[10px] text-muted-foreground leading-snug">
                {eu
                    ? 'National measures notified to the Commission as transposing this directive.'
                    : 'EU acts this Italian measure was notified as transposing.'}
            </p>

            {error && (
                <p className="text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded p-2 leading-snug">{error}</p>
            )}

            {loading && !data && (
                <div className="flex items-center gap-2 text-xs text-muted-foreground py-6 justify-center">
                    <Loader2 className="h-3.5 w-3.5 animate-spin" /> Querying CELLAR&hellip;
                </div>
            )}

            {data && verso === 'eu-it' && <EuToIt data={data} onSearchNormattiva={onSearchNormattiva} />}
            {data && verso === 'it-eu' && <ItToEu data={data} onOpenEU={onOpenEU} />}
        </div>
    );
}

function EuToIt({ data, onSearchNormattiva }: { data: any; onSearchNormattiva: (q: string) => void }) {
    const misure: any[] = data.misure || [];
    return (
        <div className="space-y-3">
            <div className="rounded border border-border bg-muted/30 p-2">
                <p className="text-xs leading-snug">{data.direttiva?.titolo || data.direttiva?.celex}</p>
                <div className="flex flex-wrap gap-x-3 gap-y-1 mt-1.5 text-[10px] text-muted-foreground">
                    <span className="font-mono">{data.direttiva?.celex}</span>
                    {data.direttiva?.termine_recepimento && (
                        <span>Deadline {data.direttiva.termine_recepimento}</span>
                    )}
                    {data.direttiva?.link && (
                        <a href={data.direttiva.link} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 hover:text-primary">
                            EUR-Lex <ExternalLink className="h-2.5 w-2.5" />
                        </a>
                    )}
                </div>
            </div>

            {misure.length === 0 ? (
                // "Nothing has been notified yet" and "this directive does not
                // exist" are different answers; the server already refuses the
                // second, so reaching here means the first, and it is said.
                <p className="text-xs text-muted-foreground italic py-4 text-center leading-snug">
                    No national measure has been notified for {data.paese}.
                </p>
            ) : (
                <div className="space-y-1.5">
                    <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-semibold">
                        {misure.length} {misure.length === 1 ? 'measure' : 'measures'} &middot; {data.paese}
                    </p>
                    {misure.map((m, i) => (
                        <div key={m.celex || i} className="rounded border border-border bg-card p-2">
                            {m.tipo && <p className="text-[10px] uppercase tracking-wide text-muted-foreground">{m.tipo}</p>}
                            <p className="text-xs leading-snug mt-0.5">{m.riferimento || m.titolo || m.celex}</p>
                            {m.riferimento && m.titolo && m.titolo !== m.riferimento && (
                                <p className="text-[10px] text-muted-foreground leading-snug mt-0.5 line-clamp-3">{m.titolo}</p>
                            )}
                            <div className="flex items-center justify-between mt-1.5 gap-2">
                                <span className="text-[10px] text-muted-foreground">
                                    {m.entrata_in_vigore ? `In force ${m.entrata_in_vigore}` : ''}
                                </span>
                                {m.riferimento && (
                                    // It searches rather than opening: CELLAR names the
                                    // measure, it does not give its Normattiva id, so
                                    // opening one would mean opening a guess. The
                                    // reference goes into the search the reader can see.
                                    <button
                                        onClick={() => onSearchNormattiva(m.riferimento)}
                                        className="inline-flex items-center gap-1 text-[10px] text-muted-foreground hover:text-primary shrink-0"
                                        title="Search Normattiva for this reference"
                                    >
                                        <Search className="h-2.5 w-2.5" /> Normattiva
                                    </button>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {(data.note || []).map((n: string, i: number) => (
                <p key={i} className="text-[10px] text-muted-foreground italic leading-snug">{n}</p>
            ))}
        </div>
    );
}

function ItToEu({ data, onOpenEU }: { data: any; onOpenEU: (celex: string, title: string) => void }) {
    const direttive: any[] = data.direttive || [];
    return (
        <div className="space-y-3">
            {direttive.length === 0 ? (
                <p className="text-xs text-muted-foreground italic py-4 text-center leading-snug">
                    This act is not recorded as transposing an EU measure.
                </p>
            ) : (
                <div className="space-y-1.5">
                    <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-semibold">
                        {direttive.length} {direttive.length === 1 ? 'EU act' : 'EU acts'}
                    </p>
                    {direttive.map((d, i) => (
                        <button
                            key={d.celex || i}
                            onClick={() => onOpenEU(d.celex, d.titolo || d.celex)}
                            className="w-full text-left rounded border border-border bg-card p-2 hover:bg-accent hover:border-primary/50 transition-colors group"
                        >
                            <p className="text-xs leading-snug">{d.titolo || d.celex}</p>
                            <div className="flex items-center justify-between mt-1.5 gap-2 text-[10px] text-muted-foreground">
                                <span className="font-mono">{d.celex}</span>
                                <span className="inline-flex items-center gap-1 group-hover:text-primary">
                                    Open <ArrowRight className="h-2.5 w-2.5" />
                                </span>
                            </div>
                            {d.termine_recepimento && (
                                <p className="text-[10px] text-muted-foreground mt-0.5">Deadline {d.termine_recepimento}</p>
                            )}
                        </button>
                    ))}
                </div>
            )}

            {(data.note || []).map((n: string, i: number) => (
                <p key={i} className="text-[10px] text-muted-foreground italic leading-snug">{n}</p>
            ))}
        </div>
    );
}
