<div align="center">
  <img src="logo.png" width="128" alt="WikiLoop"><br>
  <h1>WikiLoop</h1>
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
    <a href="https://github.com/jasen215/wikiloop/releases"><img src="https://img.shields.io/github/v/release/jasen215/wikiloop" alt="Release"></a>
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue" alt="Platform">
  </p>
</div>

WikiLoop ist eine local-first Wissenssuchmaschine für Agenten. Sie destilliert rohe Dokumente in eine strukturierte, überprüfbare Markdown-Wissensbasis und stellt zwei MCP-Tools zur Verfügung — `kb_search` und `kb_page` — mit denen Agenten in ihrem eigenen Tempo suchen und tief lesen können.

![WikiLoop Screenshot](image-001.png)

## Designphilosophie

WikiLoop basiert auf einer Beobachtung: **Agenten nutzen externe Wissenstools genauso wie Menschen Suchmaschinen nutzen** — sie stellen mehrere Anfragen aus verschiedenen Blickwinkeln, folgen Links und synthetisieren ihre eigenen Schlussfolgerungen. Sie wollen keine vorgefertigte Antwort; sie wollen das Rohmaterial, um eigene Schlüsse zu ziehen.

Das bedeutet, WikiLoops Aufgabe ist es nicht, Fragen zu beantworten. Es soll sicherstellen, dass wenn ein Agent nach etwas sucht, er die richtigen Dokumente findet — und sie vollständig lesen kann.

```text
wikiloop-kb/
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

## Wie Agenten WikiLoop nutzen

Agenten interagieren mit WikiLoop über zwei MCP-Tools:

**`kb_search(query, limit?)`** — Suche mit einem Schlüsselwort oder einer Phrase. Gibt bis zu 5 Source-Notes und 3 Concept/Comparison/Decision-Seiten pro Aufruf zurück. Jedes Ergebnis enthält ein `related`-Feld mit verknüpften Dokumenten zur Navigation. Verwenden Sie mehrere Suchen mit verschiedenen Schlüsselwörtern, um ein Thema aus mehreren Blickwinkeln abzudecken.

**`kb_page(ids, full?)`** — Vollständigen Inhalt einer oder mehrerer Seiten per ID abrufen (aus `kb_search`-Ergebnissen). Übergeben Sie bis zu 5 IDs um mehrere Dokumente gleichzeitig zu scannen, oder `full=true` mit einer einzelnen ID für vollständigen ungekürzten Text.

Empfohlener Agenten-Workflow:

```text
kb_search("Schlüsselwort A")        → relevante Dokumente entdecken
kb_search("Schlüsselwort B")        → einen anderen Blickwinkel abdecken
kb_page(["id1", "id2", "id3"])      → die relevantesten tief lesen
Agent synthetisiert eigene Antwort aus dem Gefundenen
```

Agenten sollen iterativ suchen, `related`-Links folgen, Quellen gegenseitig prüfen und eigene Schlüsse ziehen. WikiLoop generiert keine Antworten.

## WikiLoop vs RAG

Traditionelles RAG ruft Kontext ab und übergibt ihn dem LLM zum Beantworten. WikiLoop übergibt dem Agenten Rohmaterialien und lässt den Agenten selbst schlussfolgern.

```text
RAG:       Benutzerfrage → Kontext abrufen → LLM antwortet
WikiLoop:  Agent sucht → Agent liest → Agent synthetisiert
```

| | RAG | WikiLoop |
|---|---|---|
| Wissensform | Implizit (Vektoren oder Chunks) | Explizit (Markdown, überprüfbar) |
| Agentenrolle | Passiver Empfänger von Kontext | Aktiver Sucher und Leser |
| Antwortquelle | Systemgeneriert | Agentensynthetisiert |
| Überprüfbar | Nein | Ja — git diff, lint, Konfliktlinks |
| Multi-Hop-Reasoning | LLM-abhängig | Graph-Erweiterung via `related`-Links |
| Embedding | Erforderlich | Nicht erforderlich (reines FTS) |

WikiLoop-Bundles sind konform mit [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf).

## Wissens-Pipeline

Rohe Dokumente durchlaufen eine Destillations-Pipeline bevor Agenten sie durchsuchen können:

**Schritt 1 — Destillieren (automatisch)**

Legen Sie eine beliebige Markdown-Datei in `raw/`. Der `wikiloop serve`-Watcher führt automatisch Destillation + Indizierung durch. Das LLM extrahiert strukturierte Source-Notes in `wiki/source-notes/`, einschließlich:
- `key_claims` mit eingebetteten Aliasen und sprachübergreifenden Äquivalenten (ALIAS RULE) — stellt sicher, dass FTS alle Abfragevarianten trifft
- Benannte Entitätsannotationen im `【entity|type】`-Format
- `related_to`-, `supports`-, `contradicts`-Links — speist das `related`-Feld in Suchergebnissen
- `authority` (1–5) und `doc_type`-Metadaten

**Schritt 2 — Synthetisieren (auf Anfrage)**

```bash
wikiloop synthesize --topic "RAG"
```

Generiert Concept/Comparison/Decision-Seiten aus Source-Notes, wenn genug Quellen zu einem Thema angesammelt wurden. Seiten mit weniger als 2 Quellenreferenzen gehen in `wiki/<type>/_draft/` und werden erst indiziert, wenn mehr Quellen hinzugefügt werden.

**Schritt 3 — Suchen**

Agenten verwenden `kb_search` + `kb_page` via MCP. Die Suche ist reines FTS (SQLite FTS5 mit BM25-Bewertung). Kein Vektormodell erforderlich.

## Installation

Neueste Version herunterladen:

| Plattform | Datei |
|---|---|
| macOS Apple Silicon (ARM64) | `WikiLoop-<version>-macos-arm64.dmg` |
| Linux x86_64 | `wikiloop-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `wikiloop-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `wikiloop-<version>-windows-amd64.zip` |

> **macOS Intel (x86_64):** Kein vorgefertigtes Release. GitHub Actions hat den Intel macOS Runner im April 2025 eingestellt. Bauen Sie auf Ihrem Intel Mac aus dem Quellcode: `CGO_ENABLED=1 go build -tags fts5 -o wikiloop ./cmd/wikiloop/`

**macOS:** Öffnen Sie das DMG und ziehen Sie WikiLoop in Applications. Die App läuft als Menüleistensymbol.

**Linux:**
```bash
tar -xzf wikiloop-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/wikiloop /usr/local/bin/wikiloop
```

**Windows:** Entpacken Sie das ZIP und führen Sie `wikiloop.exe serve` aus (oder `wikiloop.exe stdio` für MCP). Fügen Sie das Verzeichnis zum `PATH` hinzu. Kein CGO erforderlich — reines Go-Binary.

**HarmonyOS PC (Community, experimentell):** WikiLoop wird nicht offiziell für HarmonyOS PC veröffentlicht. Da das Kern-Binary kein CGO benötigt (reines Go + SQLite), kann es nativ auf HarmonyOS mit dem Community-Paketmanager [Harmonybrew](https://harmonybrew.dev) gebaut werden. Siehe [ohos_go_cgo](https://github.com/ohos-go/ohos_go_cgo) für eine Anleitung zur Einrichtung von Go + CGO auf HarmonyOS PC.

```bash
# Auf HarmonyOS PC (nach Installation von Go via Harmonybrew)
CGO_ENABLED=0 go build -tags fts5 -o wikiloop ./cmd/wikiloop/
wikiloop serve
```

## Aus dem Quellcode bauen

Erfordert Go 1.25+. Kein CGO erforderlich.

```bash
# macOS / Linux
go build -tags fts5 -o wikiloop ./cmd/wikiloop/

# Windows
go build -tags fts5 -o wikiloop.exe ./cmd/wikiloop/
```

Oder verwenden Sie das Multi-Plattform-Build-Skript:

```bash
./scripts/build.sh [version] [target...]
```

| Target | Ausgabe | Plattform |
|---|---|---|
| `darwin-arm64` | `dist/WikiLoop-<version>-macos-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/wikiloop-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/wikiloop-<version>-linux-arm64.tar.gz` | Linux ARM64 |
| `windows-amd64` | `dist/wikiloop-<version>-windows-amd64.zip` | Windows x86_64 |

## Repository-Struktur

```text
wikiloop/
  cmd/wikiloop/        # Haupteinstiegspunkt
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

`wikiloop init` füllt das `schema/`-Verzeichnis der KB mit gebündelten Autorenregeln und Seitenvorlagen:

- `schema/templates/`: Markdown-Vorlagen für Source-Note / Concept / Comparison / Decision-Seiten.
- `schema/references/`: Autorenregeln — Seitentypen, Zitierregeln, Konfliktregeln, Verzeichnisstruktur.

Die Destillations/Synthese-Prompts lesen diese Vorlagen, daher passt ihre Bearbeitung das generierte Wiki-Format pro KB an.

## Schnellstart

```bash
export WIKILOOP_KB=/path/to/your-kb

wikiloop init           # KB-Verzeichnisse erstellen und Schema/Vorlagen kopieren
wikiloop serve          # Server starten: MCP + Web UI + Datei-Watcher
wikiloop index          # FTS-Index erstellen/aktualisieren
wikiloop status         # Index-Statistiken
wikiloop lint           # Wiki-Seiten prüfen
```

## Befehlsreferenz

Alle Befehle akzeptieren ein globales `--kb <path>`-Flag (Standard: `$WIKILOOP_KB`, dann `~/wikiloop-kb`).

| Befehl | Beschreibung |
|---|---|
| `wikiloop init [--force]` | KB-Verzeichnisse erstellen und gebündelte Schema/Vorlagen kopieren. |
| `wikiloop serve` | Lang laufenden Server starten: HTTP MCP (`/mcp`) + Web UI + Datei-Watcher. Standard ohne Unterbefehl. |
| `wikiloop index` | FTS-Index aus `wiki/`- und `raw/`-Markdown erstellen/aktualisieren. |
| `wikiloop search <query>` | FTS-Schlüsselwortsuche; gibt gerankte Treffer mit Pfaden und Snippets aus. |
| `wikiloop synthesize [--topic X] [--full]` | Concept/Comparison/Decision-Seiten aus Source-Notes generieren. |
| `wikiloop synthesize --gaps --topic X` | Wissenslückenanalyse für ein Thema. |
| `wikiloop import-lark <URL>` | Eine Lark/Feishu Wiki-Seite und ihre eingebetteten Tabellen in `raw/lark/` importieren. Erfordert ein eingeloggtes `lark-cli`. |
| `wikiloop lint` | Wiki-Seiten prüfen: fehlende Frontmatter-Felder, defekte Quelllinks. |
| `wikiloop status` | Index-Statistiken ausgeben (Dokumentanzahl, Indexgröße). |
| `wikiloop service <install\|uninstall\|start\|stop\|status\|logs>` | OS-Service verwalten (launchd / systemd). |

**LLM-Konfiguration** (Abschnitt `distill` in `config.yaml` unter dem KB-Root) ist für `distill` und `synthesize` erforderlich.

## MCP-Server

WikiLoop stellt KB-Tools über das MCP-Protokoll bereit.

**Verfügbare Tools:** `kb_search`, `kb_page`

Admin-Operationen (`status`, `reindex`, `lint`) sind über die Web-UI oder CLI verfügbar (`wikiloop status`, `wikiloop index`, `wikiloop lint`).

---

### Szenario 1: Lokales Multi-Agenten-Sharing

HTTP-Modus wird empfohlen: ein WikiLoop-Prozess, geteilt von allen Agenten — Claude Code, Cursor, VS Code (Copilot), Windsurf, Trae, Codex, Hermes, OpenClaw und andere.

**Schritt 1: WikiLoop starten**

```bash
export WIKILOOP_KB=/path/to/wikiloop-kb
wikiloop serve
```

> Auf macOS: Doppelklick auf WikiLoop.app zum Starten als Menüleistensymbol.

**Schritt 2: HTTP MCP in jedem Agenten konfigurieren**

Zu `~/.claude.json` unter `mcpServers` hinzufügen:

```json
{
  "mcpServers": {
    "wikiloop": {
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

In gehosteten Umgebungen (Hermes, OpenClaw usw.) WikiLoop auf dem dauerhaften Volume installieren und über **stdio** aufrufen — WikiLoop startet als Unterprozess des Agenten-Hosts, der Watcher läuft automatisch im Hintergrund.

Beispiel (NAS-gemountetes OpenClaw/Hermes, Einhängepunkt `/root/.openclaw`):

**1. Auf dauerhaftem Volume installieren (einmalig):**

```bash
tar -xzf wikiloop-linux-amd64.tar.gz -C /root/.openclaw/wikiloop/
chmod +x /root/.openclaw/wikiloop/wikiloop
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
  wikiloop:
    command: /root/.openclaw/wikiloop/wikiloop
    args: [stdio]
    env:
      WIKILOOP_KB: /root/.openclaw/wikiloop-kb
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

Unterverzeichnisse nach eigenem Ermessen organisieren — WikiLoop erzwingt keine feste Struktur unter `raw/`.

## Systemdienst (optional)

`wikiloop serve` enthält einen eingebauten Watcher, der automatisch das KB-Verzeichnis überwacht, Destillation auslöst und den Index neu aufbaut. Keine weitere Einrichtung erforderlich.

Um WikiLoop beim **Start zu starten und im Hintergrund auszuführen**, als Systemdienst installieren (macOS launchd / Linux systemd):

```bash
wikiloop service install --kb /path/to/your-kb
wikiloop service status
wikiloop service uninstall
```

Logs: `{WIKILOOP_KB}/index/watcher.log`
