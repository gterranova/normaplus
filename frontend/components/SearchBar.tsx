'use client';

import { useState, FormEvent } from 'react';
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Search } from "lucide-react"

interface SearchBarProps {
    onSearch: (query: string) => void;
    loading: boolean;
}

export default function SearchBar({ onSearch, loading }: SearchBarProps) {
    const [query, setQuery] = useState('');

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault();
        if (query.trim()) {
            onSearch(query.trim());
        }
    };

    return (
        <form onSubmit={handleSubmit} className="w-full max-w-3xl mx-auto flex items-center space-x-2">
            <div className="relative w-full">
                <Search className="absolute left-2.5 top-2 h-4 w-4 text-muted-foreground" />
                <Input
                    type="text"
                    placeholder="Reference (e.g. Costituzione or 28 dicembre 2000, n. 445)..."
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    className="pl-8 h-8 bg-background border-input shadow-sm"
                    disabled={loading}
                />
            </div>
            <Button type="submit" disabled={loading || !query.trim()} className="h-8 px-8 font-semibold shadow-sm">
                {loading ? 'Searching...' : 'Search'}
            </Button>
        </form>
    );
}
