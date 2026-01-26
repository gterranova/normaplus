'use client';

import { useState } from 'react';
import SearchBar from '@/components/SearchBar';
import Sidebar from '@/components/Sidebar';
import DocumentView from '@/components/DocumentView';
import { Scale, XCircle } from "lucide-react"

interface DocMetadata {
  codice_redazionale: string;
  data_pubblicazione_gazzetta: string;
  title: string;
}

export default function Home() {
  const [results, setResults] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // History State
  const [history, setHistory] = useState<DocMetadata[]>([]);
  const [currentIndex, setCurrentIndex] = useState(-1);

  const selectedDocument = currentIndex >= 0 ? history[currentIndex] : null;

  const handleSearch = async (query: string) => {
    setLoading(true);
    setError('');

    try {
      const response = await fetch(`http://localhost:8080/api/search?q=${encodeURIComponent(query)}`);
      if (!response.ok) throw new Error('Search failed');

      const data = await response.json();
      setResults(data || []);
      // Auto-open results logic? Or sidebar handles state.
    } catch (err) {
      setError('Failed to search. Make sure the backend server is running on port 8080.');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  // Called when user clicks a search result
  const handleSelectDocument = (doc: any) => {
    // Check for duplicate of CURRENT document
    if (selectedDocument && selectedDocument.codice_redazionale === doc.codice_redazionale) {
      return; // Already viewing this document
    }

    const newDoc = {
      codice_redazionale: doc.codice_redazionale,
      data_pubblicazione_gazzetta: doc.data_pubblicazione_gazzetta,
      title: doc.title
    };

    // Normal history behavior: overwrite future if we branched
    const newHistory = history.slice(0, currentIndex + 1);
    newHistory.push(newDoc);
    setHistory(newHistory);
    setCurrentIndex(newHistory.length - 1);
  };

  // Called when user clicks a history item
  const handleSelectHistory = (index: number) => {
    setCurrentIndex(index);
  };

  // Called when user clicks an internal link inside DocumentView
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
        // Check duplicate
        if (selectedDocument && selectedDocument.codice_redazionale === newId) {
          return; // Already viewing
        }

        const newDoc = {
          codice_redazionale: newId,
          data_pubblicazione_gazzetta: newDate,
          title: newTitle || `Documento ${newId}`
        };

        const newHistory = history.slice(0, currentIndex + 1);
        newHistory.push(newDoc);
        setHistory(newHistory);
        setCurrentIndex(newHistory.length - 1);
      } else {
        throw new Error('Invalid document metadata received from backend. CORS or Parse Error.');
      }
    } catch (err: any) {
      console.error("Navigation failed", err);
      setError(err.message || 'Failed to navigate to linked document');
    } finally {
      setLoading(false);
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
          <nav className="text-sm font-medium text-muted-foreground hidden md:flex space-x-6">
            <a href="#" className="hover:text-primary transition-colors">Documentation</a>
          </nav>
        </div>
      </header>

      <div className="flex-1 min-h-0 container mx-auto px-4 py-6">
        {error && (
          <div className="p-4 mb-4 bg-destructive/10 border border-destructive/20 rounded-lg text-destructive text-center max-w-2xl mx-auto shrink-0 flex items-center justify-between group">
            <span className="flex-1">{error}</span>
            <button
              onClick={() => setError('')}
              className="ml-4 text-destructive/60 hover:text-destructive transition-colors"
              title="Dismiss"
            >
              <XCircle className="h-5 w-5" />
            </button>
          </div>
        )}

        {/* Mobile Search (visible only on small screens) */}
        <div className="md:hidden mb-6 shrink-0">
          <SearchBar onSearch={handleSearch} loading={loading} />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 h-full">
          <div className={`lg:col-span-3 transition-all duration-300 h-full overflow-hidden flex flex-col ${!selectedDocument && results.length === 0 ? 'hidden' : ''}`}>
            <Sidebar
              results={results}
              history={history}
              currentIndex={currentIndex}
              onSelectDocument={handleSelectDocument}
              onSelectHistory={handleSelectHistory}
              selectedDocument={selectedDocument}
            />
          </div>

          <div className={`flex flex-col min-h-0 ${!selectedDocument && results.length === 0 ? 'lg:col-span-12' : 'lg:col-span-9'} h-full panel-transition`}>
            {selectedDocument ? (
              <DocumentView
                document={selectedDocument}
                onNavigate={handleNavigate}
              />
            ) : (
              <div className="h-full flex flex-col items-center justify-center text-muted-foreground opacity-50">
                <Scale className="h-24 w-24 mb-4 stroke-1" />
                <p className="text-xl font-light">Select a document to begin reading</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
