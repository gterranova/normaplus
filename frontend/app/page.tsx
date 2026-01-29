'use client';

import { useState, useEffect } from 'react';
import SearchBar from '@/components/SearchBar';
import Sidebar, { Document as HistoryDef } from '@/components/Sidebar';
import DocumentView from '@/components/DocumentView';
import DocumentSidebar from '@/components/DocumentSidebar';
import { Scale, XCircle, LogOut, Sun, Moon, Settings } from "lucide-react"
import { useUser } from '@/components/UserProvider';

export default function Home() {
  const [results, setResults] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // History State
  const [history, setHistory] = useState<HistoryDef[]>([]);
  const [currentIndex, setCurrentIndex] = useState(-1);

  // Bookmarks State
  const { user, logout, updatePreference } = useUser();
  const [bookmarks, setBookmarks] = useState<HistoryDef[]>([]);

  const uiState = user?.ui_state ? JSON.parse(user.ui_state) : {};

  const handleUIStateChange = (newUIState: any) => {
    updatePreference({ ui_state: JSON.stringify(newUIState) });
  };

  useEffect(() => {
    if (user?.id) {
      fetchBookmarks(user.id);
    } else {
      setBookmarks([]);
    }
  }, [user]);

  const fetchBookmarks = async (userId: number) => {
    try {
      const res = await fetch(`http://localhost:8080/api/bookmarks?userId=${userId}`);
      if (res.ok) {
        const data = await res.json();
        const mapped = data?.map((b: any) => ({
          codice_redazionale: b.doc_id,
          title: b.title,
          data_pubblicazione_gazzetta: b.date,
          category: b.category,
          isPinned: true
        }));
        setBookmarks(mapped || []);
      }
    } catch (e) {
      console.error("Failed to fetch bookmarks", e);
    }
  };

  const handleAnnotationClick = (id: number) => {
    const el = document.querySelector(`mark[data-id="${id}"]`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      // Trigger the highlight animation
      el.classList.remove('animate-highlight');
      void (el as HTMLElement).offsetWidth;
      el.classList.add('animate-highlight');
    }
  };

  const handleUpdateBookmarkCategory = async (docID: string, category: string) => {
    if (!user) return;
    try {
      const res = await fetch(`http://localhost:8080/api/bookmarks?userId=${user.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ doc_id: docID, category })
      });
      if (res.ok) {
        setBookmarks(prev => prev.map(b => b.codice_redazionale === docID ? { ...b, category } : b));
      }
    } catch (e) {
      console.error("Failed to update bookmark category", e);
    }
  };

  // TOC State
  const [toc, setToc] = useState<any[]>([]);
  const [activeIndex, setActiveIndex] = useState(-1);

  // Annotations State
  const [annotations, setAnnotations] = useState<any[]>([]);

  const fetchAnnotations = async (userId: number, docId: string) => {
    try {
      const res = await fetch(`http://localhost:8080/api/annotations?userId=${userId}&docId=${docId}`);
      if (res.ok) {
        setAnnotations(await res.json());
      }
    } catch (e) {
      console.error("Failed to fetch annotations", e);
    }
  };

  const selectedDocument = currentIndex >= 0 ? history[currentIndex] : null;

  useEffect(() => {
    if (user?.id && selectedDocument?.codice_redazionale) {
      fetchAnnotations(user.id, selectedDocument.codice_redazionale);
    } else {
      setAnnotations([]);
    }
  }, [user, selectedDocument?.codice_redazionale]);

  const handleDeleteAnnotation = async (id: number) => {
    try {
      const res = await fetch(`http://localhost:8080/api/annotations?id=${id}`, { method: 'DELETE' });
      if (res.ok) {
        setAnnotations(prev => prev.filter(a => a.id !== id));
      }
    } catch (e) { console.error(e); }
  };

  const handleActiveSectionChange = (id: string) => {
    const index = toc.findIndex(item => item.id === id);
    if (index !== -1) {
      setActiveIndex(index);
    }
  };

  const handleToggleBookmark = async (doc: HistoryDef, e: any) => {
    e.stopPropagation();
    if (!user) return;

    const isBookmarked = bookmarks.some(b => b.codice_redazionale === doc.codice_redazionale);

    if (isBookmarked) {
      // DELETE
      try {
        const res = await fetch(`http://localhost:8080/api/bookmarks?userId=${user.id}&docId=${doc.codice_redazionale}`, { method: 'DELETE' });
        if (res.ok) {
          setBookmarks(bookmarks.filter(b => b.codice_redazionale !== doc.codice_redazionale));
        }
      } catch (e) { console.error(e); }
    } else {
      // ADD
      try {
        const res = await fetch(`http://localhost:8080/api/bookmarks?userId=${user.id}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            doc_id: doc.codice_redazionale,
            title: doc.title,
            date: doc.data_pubblicazione_gazzetta
          })
        });
        if (res.ok) {
          // Optimistic update or refetch? Optimistic for speed
          const newBm = { ...doc, isPinned: true };
          setBookmarks([...bookmarks, newBm]);
        }
      } catch (e) { console.error(e); }
    }
  };

  const handleSearch = async (query: string) => {
    setLoading(true);
    setError('');

    try {
      const response = await fetch(`http://localhost:8080/api/search?q=${encodeURIComponent(query)}`);
      if (!response.ok) throw new Error('Search failed');

      const data = await response.json();
      setResults(data || []);
    } catch (err) {
      setError('Failed to search. Make sure the backend server is running on port 8080.');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  // Helper to add document to unique history
  const navigateToDocument = (newDoc: HistoryDef) => {
    // Check for duplicate by ID
    const existingIndex = history.findIndex(h => h.codice_redazionale === newDoc.codice_redazionale);

    if (existingIndex >= 0) {
      // Already in history, jump to it
      setCurrentIndex(existingIndex);
      // Optional: Update title if it changed?
    } else {
      // Append new
      const newHistory = [...history, newDoc];
      setHistory(newHistory);
      setCurrentIndex(newHistory.length - 1);
    }
  }

  // Called when user clicks a search result
  const handleSelectDocument = (doc: any) => {
    navigateToDocument({
      codice_redazionale: doc.codice_redazionale,
      data_pubblicazione_gazzetta: doc.data_pubblicazione_gazzetta,
      title: doc.title,
      isPinned: false
    });
  };

  // Called when user clicks a history item
  const handleSelectHistory = (index: number) => {
    setCurrentIndex(index);
  };

  const handleRemoveHistory = (index: number, e: any) => {
    e.stopPropagation();
    const newHistory = [...history];
    newHistory.splice(index, 1);
    setHistory(newHistory);

    // Adjust currentIndex
    if (index === currentIndex) {
      setCurrentIndex(-1); // Deselect if removing current
    } else if (index < currentIndex) {
      setCurrentIndex(currentIndex - 1);
    }
  };

  const handlePinHistory = (index: number, e: any) => {
    e.stopPropagation();
    const newHistory = [...history];
    newHistory[index].isPinned = !newHistory[index].isPinned;
    setHistory(newHistory);
  };

  const handleNavigate = async (urn: string) => {
    setLoading(true);
    setError('');

    try {
      const response = await fetch(`http://localhost:8080/api/document?urn=${encodeURIComponent(urn)}&format=xml`);

      if (!response.ok) {
        const msg = await response.text();
        throw new Error(`Link resolution failed: ${msg}`);
      }

      const newId = response.headers.get('X-Document-Id');
      const newDate = response.headers.get('X-Document-Date');
      const newTitle = response.headers.get('X-Document-Title');

      if (newId && newDate) {
        navigateToDocument({
          codice_redazionale: newId,
          data_pubblicazione_gazzetta: newDate,
          title: newTitle || `Documento ${newId}`,
          isPinned: false
        });
      } else {
        throw new Error('Invalid document metadata received from backend');
      }
    } catch (err: any) {
      console.error("Navigation failed", err);
      setError(err.message || 'Failed to navigate to linked document');
    } finally {
      setLoading(false);
    }
  };

  const handleSelectSection = (index: number) => {
    const item = toc[index];
    if (item && item.id && selectedDocument) {
      // Clone and update urnFragment to trigger scrolling in DocumentView
      const updatedDoc = { ...selectedDocument, urnFragment: item.id };
      // We don't want to modify history permanently for scrolling? 
      // Actually, we can just update the history item in place so if we come back, we remember position?
      // Or just force a re-render.
      // Updating history causes re-render.
      const newHistory = [...history];
      newHistory[currentIndex] = updatedDoc;
      setHistory(newHistory);
    }
  };

  return (
    <main className="h-screen bg-background text-foreground antialiased selection:bg-primary/20 selection:text-primary flex flex-col overflow-hidden">
      <header className="shrink-0 border-b bg-card/80 backdrop-blur-md z-50 shadow-sm">
        <div className="container mx-auto px-4 py-3 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="bg-primary/10 p-2 rounded-lg border border-primary/20">
              <Scale className="h-5 w-5 text-primary" />
            </div>
            <div>
              <h1 className="text-lg font-bold tracking-tight text-foreground leading-none">Normattiva Search</h1>
              <p className="text-[10px] text-muted-foreground font-medium uppercase tracking-wider mt-0.5">Italian Legal Corpus</p>
            </div>
          </div>
          <div className="flex-1 max-w-xl mx-8 hidden md:block">
            <SearchBar onSearch={handleSearch} loading={loading} />
          </div>
          <div className="flex items-center space-x-3">
            {user && (
              <div className="flex items-center space-x-2 bg-accent/30 px-4 py-1.5 rounded-full border border-border/50">
                <span className="w-2.5 h-2.5 rounded-full shadow-sm" style={{ backgroundColor: user.color }}></span>
                <span className="text-foreground font-medium">{user.name}</span>
              </div>
            )}
            <div className="flex items-center bg-muted/30 rounded-lg p-0.5 border">
              <button
                onClick={() => updatePreference({ mode: user?.mode === 'dark' ? 'light' : 'dark' })}
                className="p-1.5 rounded-md hover:bg-background transition-all text-muted-foreground hover:text-foreground"
                title={user?.mode === 'dark' ? "Switch to Light Mode" : "Switch to Dark Mode"}
              >
                {user?.mode === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
              </button>
              <div className="w-px h-4 bg-border mx-0.5"></div>
              <button
                onClick={logout}
                className="p-1.5 rounded-md hover:bg-background transition-all text-muted-foreground hover:text-destructive"
                title="Switch Profile"
              >
                <LogOut className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      </header>

      <div className="flex-1 min-h-0 container mx-auto px-4 py-6">
        {error && (
          <div className="p-4 mb-4 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-center max-w-2xl mx-auto shrink-0 flex items-center justify-between group">
            <span className="flex-1">{error}</span>
            <button
              onClick={() => setError('')}
              className="ml-4 text-destructive/60 hover:text-destructive transition-colors"
            >
              <XCircle className="h-5 w-5" />
            </button>
          </div>
        )}

        {/* Mobile Search */}
        <div className="md:hidden mb-6 shrink-0">
          <SearchBar onSearch={handleSearch} loading={loading} />
        </div>

        <div className="flex h-full gap-6">
          {/* Left Sidebar: Search & History */}
          <div className="h-full shrink-0 overflow-hidden">
            <Sidebar
              results={results}
              history={history}
              bookmarks={bookmarks}
              currentIndex={currentIndex}
              uiState={uiState}
              onSelectDocument={handleSelectDocument}
              onSelectHistory={handleSelectHistory}
              onRemoveHistory={handleRemoveHistory}
              onToggleBookmark={handleToggleBookmark}
              onUpdateBookmarkCategory={handleUpdateBookmarkCategory}
              onUIStateChange={handleUIStateChange}
              selectedDocument={selectedDocument}
            />
          </div>

          {/* Main Content */}
          <div className={`flex flex-col min-w-0 min-h-0 flex-1 h-full panel-transition rounded-lg border border-border/50 shadow-sm overflow-hidden`}>
            {selectedDocument ? (
              <DocumentView
                docData={selectedDocument}
                onNavigate={handleNavigate}
                onTOCParsed={setToc}
                onActiveSectionChange={handleActiveSectionChange}
                annotations={annotations}
                onAnnotationAction={() => user && selectedDocument && fetchAnnotations(user.id, selectedDocument.codice_redazionale)}
              />
            ) : (
              <div className="h-full flex flex-col items-center justify-center text-muted-foreground opacity-50 bg-card/30">
                <Scale className="h-24 w-24 mb-4 stroke-1" />
                <p className="text-xl font-light">Select a document to begin reading</p>
              </div>
            )}
          </div>

          {/* Right Sidebar: Unified TOC & Annotations */}
          {selectedDocument && (
            <div className="hidden lg:flex h-full shrink-0">
              <DocumentSidebar
                toc={toc}
                annotations={annotations}
                onSelectSection={handleSelectSection}
                onAnnotationClick={handleAnnotationClick}
                onDeleteAnnotation={handleDeleteAnnotation}
                activeIndex={activeIndex}
              />
            </div>
          )}
        </div>
      </div>
    </main>
  );
}
