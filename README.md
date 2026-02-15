# Norma+ (Beta)

A modern, high-performance interface for exploring Italian legislative documents, powered by [Normattiva](https://www.normattiva.it).

Norma+ enhances the standard Normattiva experience by providing a streamlined, user-friendly interface with advanced features for legal professionals, students, and citizens.

## Features

### Core Experience
*   **Full Act View**: Unlike the standard view which often fragments acts into single articles, Norma+ loads and displays the **entire legislative act** at once, facilitating comprehensive reading and analysis.
*   **Instant Search**: Real-time search functionality that queries the Normattiva database.
*   **Vigenza History**: Easily navigate through the version history of a law by selecting specific *vigenza* dates.

### Enhancements (vs. Standard Normattiva)
*   **Performance & Caching**: Smart server-side caching reduces latency for frequently accessed documents, making subsequent loads instant.
*   **Rich Export Options**: Download documents in multiple formats for offline use or drafting:
    *   **PDF**: Professional print-ready layout.
    *   **DOCX**: Editable Word document.
    *   **Markdown**: Clean text format for note-taking apps.
    *   **HTML**: Web-ready format.
*   **Personalization**:
    *   **Bookmarks**: Save important laws for quick access.
    *   **Annotations**: Highlight text and add personal comments directly to specific articles.
    *   **Navigation History**: Keep track of your research path.
*   **Modern UI**: A responsive, dark-mode compatible interface built with Next.js.

## Technical Architecture

The project is built as a unified full-stack application:

### Backend (Go)
The heart of the application, responsible for:
*   **Scraping & Parsing**: Fetches raw data from Normattiva and converts Akoma Ntoso XML into structured, readable content.
*   **Asset Serving**: Embeds and serves the compiled frontend, allowing the entire app to run as a single binary.
*   **API & Storage**: Manages user data (bookmarks, annotations) using a local SQLite database and exposes REST endpoints.

### Frontend (Next.js)
A dynamic React application featuring:
*   **Interactive Search**: As-you-type search feedback.
*   **Document Viewer**: Advanced viewer with navigation sidebar (Table of Contents) and right sidebar (Tools & Annotations).
*   **Responsive Design**: optimized for desktop and tablet usage.

## Running the Application

### Backend
```bash
cd backend
go run cmd/server/main.go
```
Server will start on http://localhost:8080

### Frontend
```bash
cd frontend
npm install
npm run dev
```
App will be available at http://localhost:3000

## API Endpoints

- `GET /api/search?q=<query>` - Search for documents
- `GET /api/document?id=<code>&date=<date>&format=<xml|markdown>` - Get document content

## Example Usage

1. Start both backend and frontend servers
2. Navigate to http://localhost:3000
3. Search for "Costituzione" 
4. Click on a result to view the document
5. Toggle between Markdown and XML formats

## Technology Stack

- **Backend**: Go, goquery
- **Frontend**: Next.js 15, TypeScript, TailwindCSS, react-markdown
- **Data Source**: normattiva.it
