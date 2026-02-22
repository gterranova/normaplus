'use client';

import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useRouter, usePathname } from 'next/navigation';

export interface User {
    id: number;
    name: string;
    color: string;
    theme: string;
    ui_language: string;
    mode: 'light' | 'dark';
    ui_state: string; // JSON blob
}

interface UserContextType {
    user: User | null;
    login: (user: User) => void;
    logout: () => void;
    updatePreference: (prefs: Partial<User>) => Promise<void>;
}

const UserContext = createContext<UserContextType | undefined>(undefined);

export function UserProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);
    const router = useRouter();
    const pathname = usePathname();

    useEffect(() => {
        const stored = localStorage.getItem('normattiva_user');
        if (stored) {
            try {
                const u = JSON.parse(stored);
                setUser(u);
            } catch (e) {
                localStorage.removeItem('normattiva_user');
            }
        }
        setLoading(false);
    }, []);

    // Apply Mode/Theme
    useEffect(() => {
        if (user) {
            const root = window.document.documentElement;
            if (user.mode === 'dark') {
                root.classList.add('dark');
            } else {
                root.classList.remove('dark');
            }
            // Add theme class if needed
            root.setAttribute('data-theme', user.theme || 'default');
        }
    }, [user?.mode, user?.theme]);

    useEffect(() => {
        if (!loading && !user && pathname !== '/users') {
            router.push('/users');
        }
    }, [user, loading, pathname, router]);

    const login = (u: User) => {
        setUser(u);
        localStorage.setItem('normattiva_user', JSON.stringify(u));
        router.push('/');
    };

    const logout = () => {
        setUser(null);
        localStorage.removeItem('normattiva_user');
        router.push('/users');
    };

    const updatePreference = async (prefs: Partial<User>) => {
        if (!user) return;
        const updated = { ...user, ...prefs };

        try {
            const res = await fetch(`/api/users`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(updated)
            });
            if (res.ok) {
                setUser(updated);
                localStorage.setItem('normattiva_user', JSON.stringify(updated));
            }
        } catch (e) {
            console.error("Failed to update preferences", e);
        }
    };

    if (loading) return null;

    return (
        <UserContext.Provider value={{ user, login, logout, updatePreference }}>
            {children}
        </UserContext.Provider>
    );
}

export const useUser = () => {
    const context = useContext(UserContext);
    if (context === undefined) {
        throw new Error('useUser must be used within a UserProvider');
    }
    return context;
};
