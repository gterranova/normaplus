'use client';

import { useState, useEffect } from 'react';
import { UserPlus, User as UserIcon } from 'lucide-react';
import { useUser, User } from '@/components/UserProvider';
import { useRouter } from 'next/navigation';

export default function UsersPage() {
    const { login } = useUser();
    const [users, setUsers] = useState<User[]>([]);
    const [isCreating, setIsCreating] = useState(false);
    const [newName, setNewName] = useState('');
    const [newColor, setNewColor] = useState('#3b82f6');
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetchUsers();
    }, []);

    const fetchUsers = async () => {
        try {
            const res = await fetch('/api/users');
            if (res.ok) {
                const data = await res.json();
                setUsers(data || []);
            }
        } catch (err) {
            console.error("Failed to fetch users", err);
        } finally {
            setLoading(false);
        }
    };

    const handleCreateUser = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!newName.trim()) return;

        try {
            const res = await fetch('/api/users', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ name: newName, color: newColor }),
            });

            if (res.ok) {
                const user = await res.json();
                setUsers([...users, user]);
                setIsCreating(false);
                setNewName('');
            }
        } catch (err) {
            console.error("Failed to create user", err);
        }
    };

    return (
        <main className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
            <div className="w-full max-w-4xl">
                <h1 className="text-4xl font-bold text-center mb-12 text-foreground">Who's searching?</h1>

                <div className="flex flex-wrap justify-center gap-8">
                    {loading ? (
                        <div className="text-muted-foreground animate-pulse">Loading profiles...</div>
                    ) : (
                        <>
                            {users.map((user) => (
                                <button
                                    key={user.id}
                                    onClick={() => login(user)}
                                    className="group flex flex-col items-center space-y-4 w-32 focus:outline-none"
                                >
                                    <div
                                        className="w-32 h-32 rounded-full bg-card border-2 border-transparent group-hover:border-primary group-hover:bg-accent/10 flex items-center justify-center transition-all overflow-hidden relative"
                                        style={{ borderColor: user.color }}
                                    >
                                        <div className="w-full h-full absolute inset-0 opacity-20" style={{ backgroundColor: user.color }} />
                                        <UserIcon className="w-16 h-16" style={{ color: user.color }} />
                                    </div>
                                    <span className="text-lg text-muted-foreground group-hover:text-foreground transition-colors font-medium text-center truncate w-full">
                                        {user.name}
                                    </span>
                                </button>
                            ))}

                            <button
                                onClick={() => setIsCreating(true)}
                                className="group flex flex-col items-center space-y-4 w-32 focus:outline-none"
                            >
                                <div className="w-32 h-32 rounded-full bg-card border-2 border-dashed border-muted-foreground/30 group-hover:border-foreground/50 group-hover:bg-accent/10 flex items-center justify-center transition-all">
                                    <UserPlus className="w-12 h-12 text-muted-foreground group-hover:text-foreground" />
                                </div>
                                <span className="text-lg text-muted-foreground group-hover:text-foreground transition-colors font-medium">
                                    Add Profile
                                </span>
                            </button>
                        </>
                    )}
                </div>

                {isCreating && (
                    <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
                        <div className="bg-card w-full max-w-md p-6 rounded-lg shadow-lg border border-border">
                            <h2 className="text-2xl font-bold mb-4">Add Profile</h2>
                            <form onSubmit={handleCreateUser} className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium mb-1">Name</label>
                                    <input
                                        autoFocus
                                        type="text"
                                        value={newName}
                                        onChange={(e) => setNewName(e.target.value)}
                                        className="w-full p-2 rounded-md border bg-background"
                                        placeholder="Enter name"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium mb-1">Color</label>
                                    <div className="flex gap-2">
                                        {['#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899'].map((color) => (
                                            <button
                                                key={color}
                                                type="button"
                                                onClick={() => setNewColor(color)}
                                                className={`w-8 h-8 rounded-full ${newColor === color ? 'ring-2 ring-offset-2 ring-primary' : ''}`}
                                                style={{ backgroundColor: color }}
                                            />
                                        ))}
                                    </div>
                                </div>
                                <div className="flex justify-end gap-2 mt-6">
                                    <button
                                        type="button"
                                        onClick={() => setIsCreating(false)}
                                        className="px-4 py-2 rounded-md hover:bg-accent"
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        type="submit"
                                        className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
                                        disabled={!newName.trim()}
                                    >
                                        Save Profile
                                    </button>
                                </div>
                            </form>
                        </div>
                    </div>
                )}
            </div>
        </main>
    );
}
