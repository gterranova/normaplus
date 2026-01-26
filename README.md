# Normattiva Search Application

A full-stack application to search and view Italian legislative documents from normattiva.it.

## Architecture

**Backend (Go):**
- RESTful API on port 8080
- Scrapes normattiva.it for search results
- Fetches Akoma Ntoso XML documents
- Converts XML to Markdown

**Frontend (Next.js):**
- Modern React interface on port 3000
- Search bar with real-time results
- Document viewer with Markdown/XML toggle
- Responsive design with dark theme

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
