//go:build fts5

package kb

const schemaSQL = `
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    layer TEXT NOT NULL,
    kind TEXT,
    title TEXT,
    description TEXT,
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    source_uri TEXT,
    updated_at INTEGER NOT NULL,
    authority INTEGER NOT NULL DEFAULT 3,
    doc_timestamp INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_doc_id TEXT NOT NULL,
    target_doc_id TEXT NOT NULL,
    relation TEXT NOT NULL,
    anchor TEXT,
    confidence REAL NOT NULL DEFAULT 1.0,
    FOREIGN KEY (source_doc_id) REFERENCES documents(id),
    FOREIGN KEY (target_doc_id) REFERENCES documents(id)
);

CREATE VIRTUAL TABLE IF NOT EXISTS document_fts USING fts5(
    id,
    title,
    content,
    kind,
    layer,
    content='documents',
    content_rowid='rowid',
    tokenize='trigram'
);

CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
    INSERT INTO document_fts(rowid, id, title, content, kind, layer)
    VALUES (new.rowid, new.id, new.title, new.content, new.kind, new.layer);
END;

CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
    INSERT INTO document_fts(document_fts, rowid, id, title, content, kind, layer)
    VALUES ('delete', old.rowid, old.id, old.title, old.content, old.kind, old.layer);
END;

CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE ON documents BEGIN
    INSERT INTO document_fts(document_fts, rowid, id, title, content, kind, layer)
    VALUES ('delete', old.rowid, old.id, old.title, old.content, old.kind, old.layer);
    INSERT INTO document_fts(rowid, id, title, content, kind, layer)
    VALUES (new.rowid, new.id, new.title, new.content, new.kind, new.layer);
END;

CREATE TABLE IF NOT EXISTS embeddings (
    doc_id TEXT PRIMARY KEY,
    model TEXT NOT NULL,
    dim INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (doc_id) REFERENCES documents(id)
);

CREATE TABLE IF NOT EXISTS distill_queue (
    path        TEXT PRIMARY KEY,
    status      TEXT NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error  TEXT,
    queued_at   INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
`
