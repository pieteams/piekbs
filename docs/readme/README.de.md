<div align="center">
  <img src="logo.png" width="128" alt="PieKBS"><br>
  <h1>PieKBS</h1>
  <p>Eine Wissenssuchmaschine für Agenten — destilliert rohe Dokumente in strukturiertes Markdown-Wiki, durchsucht und gelesen via MCP</p>
  <p>
    <a href="../../README.md">English</a> |
    <a href="README.zh-CN.md">简体中文</a> |
    <a href="README.zh-TW.md">繁體中文</a> |
    <a href="README.ru.md">Русский</a> |
    <strong>Deutsch</strong> |
    <a href="README.fr.md">Français</a> |
    <a href="README.es.md">Español</a> |
    <a href="README.ko.md">한국어</a>
  </p>
  <p>
    <a href="../../LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
    <a href="https://github.com/pieteams/piekbs/releases"><img src="https://img.shields.io/github/v/release/jasen215/piekbs" alt="Release"></a>
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue" alt="Platform">
  </p>
</div>

PieKBS ist eine local-first Wissenssuchmaschine für Agenten. Sie destilliert rohe Dokumente in eine strukturierte, überprüfbare Markdown-Wissensbasis und stellt zwei MCP-Tools zur Verfügung — `kb_search` und `kb_page` — mit denen Agenten in ihrem eigenen Tempo suchen und tief lesen können.

![PieKBS Screenshot](image-001.png)

## Designphilosophie

PieKBS basiert auf einer Beobachtung: **Agenten nutzen externe Wissenstools genauso wie Menschen Suchmaschinen nutzen** — sie stellen mehrere Anfragen aus verschiedenen Blickwinkeln, folgen Links und synthetisieren ihre eigenen Schlussfolgerungen. Sie wollen keine vorgefertigte Antwort; sie wollen das Rohmaterial, um eigene Schlüsse zu ziehen.

Das bedeutet, PieKBSs Aufgabe ist es nicht, Fragen zu beantworten. Es soll sicherstellen, dass wenn ein Agent nach etwas sucht, er die richtigen Dokumente findet — und sie vollständig lesen kann.

```text
piekbs-kb/
  raw/                  Quelle der Wahrheit — Originalmaterialien in jedem Format.
                        Dateien ablegen; der Watcher destilliert sie automatisch.

  wiki/                 Strukturierte Markdown-Wissensschicht (LLM-verwaltet).
    source-notes/       Eine destillierte Notiz pro Quelldokument. FTS-Suchziel.
    concepts/           Dokumentübergreifende Synthese: Konzepte und Methoden.
    comparisons/        Dokumentübergreifende Synthese: Nebeneinandervergleiche.
    decisions/          Dokumentübergreifende Synthese: technische Entscheidungen.
    _draft/             Synthetisierte Seiten mit < 2 Quellen (noch nicht indiziert).

  schema/               KB-lokale Autorenregeln und Seitenvorlagen.
                        Bearbeiten um das destillierte Seitenformat anzupassen.

  index/                Generierte Artefakte (SQLite FTS-Index, Query-Logs).
                        Wird automatisch verwaltet — nicht manuell bearbeiten.
```

## Wie Agenten PieKBS nutzen

Agenten interagieren mit PieKBS über drei MCP-Tools:

**`kb_search(query, limit?)`** — Suche mit einem Schlüsselwort oder einer Phrase. Gibt bis zu 5 Source-Notes und 3 Concept/Comparison/Decision-Seiten pro Aufruf zurück. Jedes Ergebnis enthält ein `related`-Feld mit verknüpften Dokumenten zur Navigation. Verwenden Sie mehrere Suchen mit verschiedenen Schlüsselwörtern, um ein Thema aus mehreren Blickwinkeln abzudecken.

**`kb_page(ids, full?)`** — Vollständigen Inhalt einer oder mehrerer Seiten per ID abrufen (aus `kb_search`-Ergebnissen). Übergeben Sie bis zu 5 IDs um mehrere Dokumente gleichzeitig zu scannen, oder `full=true` mit einer einzelnen ID für vollständigen ungekürzten Text.

**`kb_add(filename, content, source_url?)`** — Ein Textdokument zur Wissensdatenbank hinzufügen. Schreibt den Inhalt nach `raw/<filename>` und löst inkrementelle Indizierung aus. Die Destillation läuft asynchron im Hintergrund. Verwenden Sie das Präfix `converted/` für von Agenten extrahierte PDF/Word/Excel/EPUB-Inhalte.

Empfohlener Agenten-Workflow:

```text
kb_search("Schlüsselwort A")        → relevante Dokumente entdecken
kb_search("Schlüsselwort B")        → einen anderen Blickwinkel abdecken
kb_page(["id1", "id2", "id3"])      → die relevantesten tief lesen
Agent synthetisiert eigene Antwort aus dem Gefundenen
```

Agenten sollen iterativ suchen, `related`-Links folgen, Quellen gegenseitig prüfen und eigene Schlüsse ziehen. PieKBS generiert keine Antworten.

## PieKBS vs RAG

Traditionelles RAG ruft Kontext ab und übergibt ihn dem LLM zum Beantworten. PieKBS übergibt dem Agenten Rohmaterialien und lässt den Agenten selbst schlussfolgern.

```text
RAG:       Benutzerfrage → Kontext abrufen → LLM antwortet
PieKBS:  Agent sucht → Agent liest → Agent synthetisiert
```

| | RAG | PieKBS |
|---|---|---|
| Wissensform | Implizit (Vektoren oder Chunks) | Explizit (Markdown, überprüfbar) |
| Agentenrolle | Passiver Empfänger von Kontext | Aktiver Sucher und Leser |
| Antwortquelle | Systemgeneriert | Agentensynthetisiert |
| Überprüfbar | Nein | Ja — git diff, lint, Konfliktlinks |
| Multi-Hop-Reasoning | LLM-abhängig | Graph-Erweiterung via `related`-Links |
| Embedding | Erforderlich | Nicht erforderlich (reines FTS) |

PieKBS-Bundles sind konform mit [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf).

## Wissens-Pipeline

Rohe Dokumente durchlaufen eine Destillations-Pipeline bevor Agenten sie durchsuchen können:

**Schritt 1 — Destillieren (automatisch)**

Legen Sie eine beliebige Markdown-Datei in `raw/`. Der `piekbs serve`-Watcher führt automatisch Destillation + Indizierung durch. Das LLM extrahiert strukturierte Source-Notes in `wiki/source-notes/`, einschließlich:
- `key_claims` mit eingebetteten Aliasen und sprachübergreifenden Äquivalenten (ALIAS RULE) — stellt sicher, dass FTS alle Abfragevarianten trifft
- Benannte Entitätsannotationen im `【entity|type】`-Format
- `related_to`-, `supports`-, `contradicts`-Links — speist das `related`-Feld in Suchergebnissen
- `authority` (1–5) und `doc_type`-Metadaten

**Schritt 2 — Synthetisieren (auf Anfrage)**

```bash
piekbs synthesize --topic "RAG"
```

Generiert Concept/Comparison/Decision-Seiten aus Source-Notes, wenn genug Quellen zu einem Thema angesammelt wurden. Seiten mit weniger als 2 Quellenreferenzen gehen in `wiki/<type>/_draft/` und werden erst indiziert, wenn mehr Quellen hinzugefügt werden.

**Schritt 3 — Suchen**

Agenten verwenden `kb_search` + `kb_page` via MCP. Die Suche ist reines FTS (SQLite FTS5 mit BM25-Bewertung). Kein Vektormodell erforderlich.

## Installation

Neueste Version herunterladen:

| Plattform | Datei |
|---|---|
| macOS Apple Silicon (ARM64) | `PieKBS-<version>-macos-arm64.dmg` |
| Linux x86_64 | `piekbs-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `piekbs-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `piekbs-<version>-windows-amd64.zip` |

> **macOS Intel (x86_64):** Kein vorgefertigtes Release. GitHub Actions hat den Intel macOS Runner im April 2025 eingestellt. Bauen Sie auf Ihrem Intel Mac aus dem Quellcode: `CGO_ENABLED=1 go build -tags fts5 -o piekbs ./cmd/piekbs/`

**macOS:** Öffnen Sie das DMG und ziehen Sie PieKBS in Applications. Die App läuft als Menüleistensymbol.

**Linux:**
```bash
tar -xzf piekbs-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/piekbs /usr/local/bin/piekbs
```

**Windows:** Entpacken Sie das ZIP und führen Sie `piekbs.exe serve` aus (oder `piekbs.exe stdio` für MCP). Fügen Sie das Verzeichnis zum `PATH` hinzu. Kein CGO erforderlich — reines Go-Binary.

**HarmonyOS PC (Community, experimentell):** PieKBS wird nicht offiziell für HarmonyOS PC veröffentlicht. Da das Kern-Binary kein CGO benötigt (reines Go + SQLite), kann es nativ auf HarmonyOS mit dem Community-Paketmanager [Harmonybrew](https://harmonybrew.dev) gebaut werden. Siehe [ohos_go_cgo](https://github.com/ohos-go/ohos_go_cgo) für eine Anleitung zur Einrichtung von Go + CGO auf HarmonyOS PC.

```bash
# Auf HarmonyOS PC (nach Installation von Go via Harmonybrew)
CGO_ENABLED=0 go build -tags fts5 -o piekbs ./cmd/piekbs/
piekbs serve
```

## Aus dem Quellcode bauen

Erfordert Go 1.25+. Kein CGO erforderlich.

```bash
# macOS / Linux
go build -tags fts5 -o piekbs ./cmd/piekbs/

# Windows
go build -tags fts5 -o piekbs.exe ./cmd/piekbs/
```

Oder verwenden Sie das Multi-Plattform-Build-Skript:

```bash
./scripts/build.sh [version] [target...]
```

| Target | Ausgabe | Plattform |
|---|---|---|
| `darwin-arm64` | `dist/PieKBS-<version>-macos-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/piekbs-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/piekbs-<version>-linux-arm64.tar.gz` | Linux ARM64 |
| `windows-amd64` | `dist/piekbs-<version>-windows-amd64.zip` | Windows x86_64 |

## Repository-Struktur

```text
piekbs/
  cmd/piekbs/        # Haupteinstiegspunkt
  internal/
    kb/                # FTS-Indizierung, Suche, Graph-Erweiterung, Seitenabruf
    mcp/               # MCP-Server (stdio + HTTP)
    watcher/           # Datei-Watcher für Auto-Destillation + Neuindizierung
    distill/           # LLM-Destillations-Pipeline
    synthesize/        # Concept/Comparison/Decision-Seitengenerierung
    convert/           # Rohdateikonvertierung
    service/           # OS-Service-Manager (launchd / systemd)
    webui/             # Web-UI
    tray/              # macOS Systemtray (nur darwin)
    config/            # KB-Konfiguration (config.yaml)
  scripts/
    build.sh           # Multi-Plattform-Build-Skript
```

## Schema & Vorlagen

`piekbs init` füllt das `schema/`-Verzeichnis der KB mit gebündelten Autorenregeln und Seitenvorlagen:

- `schema/templates/`: Markdown-Vorlagen für Source-Note / Concept / Comparison / Decision-Seiten.
- `schema/references/`: Autorenregeln — Seitentypen, Zitierregeln, Konfliktregeln, Verzeichnisstruktur.

Die Destillations/Synthese-Prompts lesen diese Vorlagen, daher passt ihre Bearbeitung das generierte Wiki-Format pro KB an.

## Schnellstart

```bash
export WIKILOOP_KB=/path/to/your-kb

piekbs init           # KB-Verzeichnisse erstellen und Schema/Vorlagen kopieren
piekbs serve          # Server starten: MCP + Web UI + Datei-Watcher
piekbs index          # FTS-Index erstellen/aktualisieren
piekbs status         # Index-Statistiken
piekbs lint           # Wiki-Seiten prüfen
```

## Befehlsreferenz

Alle Befehle akzeptieren ein globales `--kb <path>`-Flag (Standard: `$WIKILOOP_KB`, dann `~/piekbs-kb`).

| Befehl | Beschreibung |
|---|---|
| `piekbs init [--force]` | KB-Verzeichnisse erstellen und gebündelte Schema/Vorlagen kopieren. |
| `piekbs serve` | Lang laufenden Server starten: HTTP MCP (`/mcp`) + Web UI + Datei-Watcher. Standard ohne Unterbefehl. |
| `piekbs index` | FTS-Index aus `wiki/`- und `raw/`-Markdown erstellen/aktualisieren. |
| `piekbs search <query>` | FTS-Schlüsselwortsuche; gibt gerankte Treffer mit Pfaden und Snippets aus. |
| `piekbs synthesize [--topic X] [--full]` | Concept/Comparison/Decision-Seiten aus Source-Notes generieren. |
| `piekbs synthesize --gaps --topic X` | Wissenslückenanalyse für ein Thema. |
| `piekbs import-lark <URL>` | Eine Lark/Feishu Wiki-Seite und ihre eingebetteten Tabellen in `raw/lark/` importieren. Erfordert ein eingeloggtes `lark-cli`. |
| `piekbs lint` | Wiki-Seiten prüfen: fehlende Frontmatter-Felder, defekte Quelllinks. |
| `piekbs status` | Index-Statistiken ausgeben (Dokumentanzahl, Indexgröße). |
| `piekbs service <install\|uninstall\|start\|stop\|status\|logs>` | OS-Service verwalten (launchd / systemd). |

**LLM-Konfiguration** (Abschnitt `distill` in `config.yaml` unter dem KB-Root) ist für `distill` und `synthesize` erforderlich.

## MCP-Server

PieKBS stellt KB-Tools über das MCP-Protokoll bereit.

**Verfügbare Tools:** `kb_search`, `kb_page`, `kb_add`

Admin-Operationen (`status`, `reindex`, `lint`) sind über die Web-UI oder CLI verfügbar (`piekbs status`, `piekbs index`, `piekbs lint`).

---

### Szenario 1: Lokales Multi-Agenten-Sharing

HTTP-Modus wird empfohlen: ein PieKBS-Prozess, geteilt von allen Agenten — Claude Code, Cursor, VS Code (Copilot), Windsurf, Trae, Codex, Hermes, OpenClaw und andere.

**Schritt 1: PieKBS starten**

```bash
export WIKILOOP_KB=/path/to/piekbs-kb
piekbs serve
```

> Auf macOS: Doppelklick auf PieKBS.app zum Starten als Menüleistensymbol.

**Schritt 2: HTTP MCP in jedem Agenten konfigurieren**

Zu `~/.claude.json` unter `mcpServers` hinzufügen:

```json
{
  "mcpServers": {
    "piekbs": {
      "type": "http",
      "url": "http://127.0.0.1:8766/mcp",
      "headers": {
        "x-api-key": "${WIKILOOP_API_KEY}"
      }
    }
  }
}
```

`x-api-key` entspricht `server.api_key` in `config.yaml`. `headers` weglassen, wenn kein api_key gesetzt ist.

---

### Szenario 2: Gehostete Agentenumgebungen

In gehosteten Umgebungen (Hermes, OpenClaw usw.) PieKBS auf dem dauerhaften Volume installieren und über **stdio** aufrufen — PieKBS startet als Unterprozess des Agenten-Hosts, der Watcher läuft automatisch im Hintergrund.

Beispiel (NAS-gemountetes OpenClaw/Hermes, Einhängepunkt `/root/.openclaw`):

**1. Auf dauerhaftem Volume installieren (einmalig):**

```bash
tar -xzf piekbs-linux-amd64.tar.gz -C /root/.openclaw/piekbs/
chmod +x /root/.openclaw/piekbs/piekbs
```

**2. markitdown installieren (empfohlen):**

markitdown ermöglicht die Konvertierung von PDF-, Word-, Excel-, PPT- und HTML-Dateien in Markdown vor der Destillation. Ohne es werden nur `.md`- und `.txt`-Dateien destilliert; Binärdateien werden nur nach Dateinamen indiziert.

```bash
pip install markitdown
# überprüfen
markitdown --version
```

> Auf OpenClaw/Hermes verifiziert (Pfad: `/root/.openclaw/workspace/bin/markitdown`). `workspace/bin` zum PATH hinzufügen oder vollständigen Pfad in Ihrer Umgebung setzen.

Wenn markitdown nicht verfügbar ist, können Agenten selbst Text extrahieren (mit LLM Vision oder anderen Tools) und das Ergebnis direkt in `$WIKILOOP_KB/raw/converted/<slug>.md` schreiben — der Watcher nimmt es automatisch auf.

**3. MCP-Konfiguration:**

Hermes (`mcp_servers` in Agent-Konfiguration):

```yaml
mcp_servers:
  piekbs:
    command: /root/.openclaw/piekbs/piekbs
    args: [stdio]
    env:
      WIKILOOP_KB: /root/.openclaw/piekbs-kb
      PATH: /root/.openclaw/workspace/bin:/usr/local/bin:/usr/bin:/bin
```

Das KB-Verzeichnis wird beim ersten Start automatisch erstellt. Kein manuelles `init` nötig.

**4. Inhalte zur Wissensbasis hinzufügen:**

Agenten mit `write_file`-Zugriff können direkt in die KB schreiben — der Watcher erkennt Änderungen und löst automatisch Indizierung und Destillation aus.

| Inhaltstyp | Schreiben nach |
|---|---|
| Artikel, Notizen, Referenzen (Markdown/Text) | `$WIKILOOP_KB/raw/<ihre-kategorie>/<slug>.md` |
| Von Agenten konvertierter PDF/Word/Excel/EPUB-Inhalt | `$WIKILOOP_KB/raw/converted/<slug>.md` |

Dateien in `raw/converted/` werden als bereits konvertiert behandelt und gehen direkt zur Destillation, überspringen den markitdown-Schritt. Alle anderen Pfade unter `raw/` werden durch die vollständige Pipeline verarbeitet (Konvertieren → Indizieren → Destillieren).

Unterverzeichnisse nach eigenem Ermessen organisieren — PieKBS erzwingt keine feste Struktur unter `raw/`.

## Systemdienst (optional)

`piekbs serve` enthält einen eingebauten Watcher, der automatisch das KB-Verzeichnis überwacht, Destillation auslöst und den Index neu aufbaut. Keine weitere Einrichtung erforderlich.

Um PieKBS beim **Start zu starten und im Hintergrund auszuführen**, als Systemdienst installieren (macOS launchd / Linux systemd):

```bash
piekbs service install --kb /path/to/your-kb
piekbs service status
piekbs service uninstall
```

Logs: `{WIKILOOP_KB}/index/watcher.log`
