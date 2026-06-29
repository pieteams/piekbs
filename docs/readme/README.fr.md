<div align="center">
  <img src="logo.png" width="128" alt="PieKBS"><br>
  <h1>PieKBS</h1>
  <p>Un moteur de recherche de connaissances pour les agents — distille les documents bruts en wiki Markdown structuré, recherche et lecture via MCP</p>
  <p>
    <a href="../../README.md">English</a> |
    <a href="README.zh-CN.md">简体中文</a> |
    <a href="README.zh-TW.md">繁體中文</a> |
    <a href="README.ru.md">Русский</a> |
    <a href="README.de.md">Deutsch</a> |
    <strong>Français</strong> |
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

PieKBS est un moteur de recherche de connaissances local pour les agents. Il distille les documents bruts en une base de connaissances Markdown structurée et vérifiable, puis expose deux outils MCP — `kb_search` et `kb_page` — permettant aux agents de rechercher et lire en profondeur à leur propre rythme.

![PieKBS Screenshot](image-001.png)

## Philosophie de conception

PieKBS repose sur une observation : **les agents utilisent les outils de connaissance externes de la même façon que les humains utilisent les moteurs de recherche** — ils formulent plusieurs requêtes sous différents angles, suivent des liens et synthétisent leurs propres conclusions. Ils ne veulent pas une réponse pré-emballée ; ils veulent les matériaux bruts pour former leurs propres conclusions.

Cela signifie que la mission de PieKBS n'est pas de répondre aux questions. C'est de s'assurer que lorsqu'un agent cherche quelque chose, il trouve les bons documents — et peut les lire intégralement.

```text
piekbs-kb/
  raw/                  Source de vérité — matériaux originaux dans tout format.
                        Déposez des fichiers ici ; le watcher les distille automatiquement.

  wiki/                 Couche de connaissances Markdown structurée (maintenue par LLM).
    source-notes/       Une note distillée par document source. Cible de recherche FTS.
    concepts/           Synthèse inter-documents : concepts et méthodologies.
    comparisons/        Synthèse inter-documents : comparaisons côte à côte.
    decisions/          Synthèse inter-documents : décisions techniques.
    _draft/             Pages synthétisées avec < 2 sources (pas encore indexées).

  schema/               Règles d'édition et modèles de pages locaux à la KB.
                        Éditez pour personnaliser le format des pages distillées.

  index/                Artefacts générés (index SQLite FTS, journaux de requêtes).
                        Géré automatiquement — ne pas éditer manuellement.
```

## Comment les agents utilisent PieKBS

Les agents interagissent avec PieKBS via trois outils MCP :

**`kb_search(query, limit?)`** — Recherche avec un mot-clé ou une phrase. Retourne jusqu'à 5 source-notes et 3 pages concept/comparison/decision par appel. Chaque résultat inclut un champ `related` listant les documents liés pour la navigation. Utilisez plusieurs recherches avec différents mots-clés pour couvrir un sujet sous plusieurs angles.

**`kb_page(ids, full?)`** — Récupère le contenu complet d'une ou plusieurs pages par ID (depuis les résultats `kb_search`). Passez jusqu'à 5 IDs pour parcourir plusieurs documents à la fois, ou `full=true` avec un seul ID pour obtenir le texte complet non tronqué.

**`kb_add(filename, content, source_url?)`** — Ajoute un document texte à la base de connaissances. Écrit le contenu dans `raw/<filename>` et déclenche l'indexation incrémentale. La distillation s'exécute de manière asynchrone en arrière-plan. Utilisez le préfixe `converted/` pour le contenu extrait par des agents depuis des fichiers PDF/Word/Excel/EPUB.

Workflow agent recommandé :

```text
kb_search("mot-clé A")              → découvrir les documents pertinents
kb_search("mot-clé B")              → couvrir un autre angle
kb_page(["id1", "id2", "id3"])      → lire en profondeur les plus pertinents
L'agent synthétise sa propre réponse à partir de ce qu'il a trouvé
```

Les agents sont censés chercher de façon itérative, suivre les liens `related`, vérifier les sources de manière croisée et former leurs propres conclusions. PieKBS ne génère pas de réponses.

## PieKBS vs RAG

Le RAG traditionnel récupère le contexte et le transmet au LLM pour répondre. PieKBS transmet les matériaux bruts à l'agent et laisse l'agent raisonner par lui-même.

```text
RAG :       question utilisateur → récupérer contexte → LLM répond
PieKBS :  agent cherche → agent lit → agent synthétise
```

| | RAG | PieKBS |
|---|---|---|
| Forme du savoir | Implicite (vecteurs ou chunks) | Explicite (Markdown, vérifiable) |
| Rôle de l'agent | Récepteur passif de contexte | Chercheur et lecteur actif |
| Source de la réponse | Générée par le système | Synthétisée par l'agent |
| Vérifiable | Non | Oui — git diff, lint, liens de conflits |
| Raisonnement multi-sauts | Dépendant du LLM | Expansion de graphe via liens `related` |
| Embedding | Requis | Non requis (FTS pur) |

Les bundles PieKBS sont conformes à [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf).

## Pipeline de connaissances

Les documents bruts passent par un pipeline de distillation avant que les agents puissent les rechercher :

**Étape 1 — Distiller (automatique)**

Déposez n'importe quel fichier Markdown dans `raw/`. Le watcher de `piekbs serve` exécute automatiquement la distillation + l'indexation. Le LLM extrait des source-notes structurées dans `wiki/source-notes/`, incluant :
- `key_claims` avec des alias intégrés et des équivalents inter-langues (ALIAS RULE) — garantit que FTS correspond à toutes les variantes de requête
- Annotations d'entités nommées au format `【entity|type】`
- Liens `related_to`, `supports`, `contradicts` — alimente le champ `related` dans les résultats de recherche
- Métadonnées `authority` (1–5) et `doc_type`

**Étape 2 — Synthétiser (à la demande)**

```bash
piekbs synthesize --topic "RAG"
```

Génère des pages concept/comparison/decision à partir de source-notes lorsque suffisamment de sources sur un sujet s'accumulent. Les pages avec moins de 2 références de sources vont dans `wiki/<type>/_draft/` et ne sont pas indexées tant que d'autres sources ne sont pas ajoutées.

**Étape 3 — Rechercher**

Les agents utilisent `kb_search` + `kb_page` via MCP. La recherche est FTS pur (SQLite FTS5 avec notation BM25). Aucun modèle vectoriel requis.

## Installation

Télécharger la dernière version :

| Plateforme | Fichier |
|---|---|
| macOS Apple Silicon (ARM64) | `PieKBS-<version>-macos-arm64.dmg` |
| Linux x86_64 | `piekbs-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `piekbs-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `piekbs-<version>-windows-amd64.zip` |

> **macOS Intel (x86_64) :** Pas de version pré-compilée. GitHub Actions a abandonné le runner Intel macOS en avril 2025. Compilez depuis les sources sur votre Mac Intel : `CGO_ENABLED=1 go build -tags fts5 -o piekbs ./cmd/piekbs/`

**macOS :** Ouvrez le DMG et faites glisser PieKBS dans Applications. L'app fonctionne comme une icône dans la barre de menus.

**Linux :**
```bash
tar -xzf piekbs-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/piekbs /usr/local/bin/piekbs
```

**Windows :** Extrayez le zip et exécutez `piekbs.exe serve` (ou `piekbs.exe stdio` pour MCP). Ajoutez le répertoire au `PATH`. Pas de CGO requis — binaire Go pur.

**HarmonyOS PC (communauté, expérimental) :** PieKBS n'est pas officiellement publié pour HarmonyOS PC. Cependant, comme le binaire principal ne nécessite pas CGO (Go pur + SQLite), il peut être compilé nativement sur HarmonyOS avec le gestionnaire de paquets communautaire [Harmonybrew](https://harmonybrew.dev). Voir [ohos_go_cgo](https://github.com/ohos-go/ohos_go_cgo) pour un guide sur la configuration Go + CGO sur HarmonyOS PC.

```bash
# Sur HarmonyOS PC (après installation de Go via Harmonybrew)
CGO_ENABLED=0 go build -tags fts5 -o piekbs ./cmd/piekbs/
piekbs serve
```

## Compiler depuis les sources

Nécessite Go 1.25+. Pas de CGO requis.

```bash
# macOS / Linux
go build -tags fts5 -o piekbs ./cmd/piekbs/

# Windows
go build -tags fts5 -o piekbs.exe ./cmd/piekbs/
```

Ou utiliser le script de compilation multi-plateforme :

```bash
./scripts/build.sh [version] [target...]
```

| Target | Sortie | Plateforme |
|---|---|---|
| `darwin-arm64` | `dist/PieKBS-<version>-macos-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/piekbs-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/piekbs-<version>-linux-arm64.tar.gz` | Linux ARM64 |
| `windows-amd64` | `dist/piekbs-<version>-windows-amd64.zip` | Windows x86_64 |

## Structure du dépôt

```text
piekbs/
  cmd/piekbs/        # point d'entrée principal
  internal/
    kb/                # indexation FTS, recherche, expansion de graphe, récupération de pages
    mcp/               # serveur MCP (stdio + HTTP)
    watcher/           # watcher de fichiers pour auto-distillation + réindexation
    distill/           # pipeline de distillation LLM
    synthesize/        # génération de pages concept/comparison/decision
    convert/           # conversion de fichiers bruts
    service/           # gestionnaire de service OS (launchd / systemd)
    webui/             # interface web
    tray/              # barre système macOS (darwin uniquement)
    config/            # configuration KB (config.yaml)
  scripts/
    build.sh           # script de compilation multi-plateforme
```

## Schéma & Modèles

`piekbs init` peuple le répertoire `schema/` de la KB avec des règles d'édition et des modèles de pages intégrés :

- `schema/templates/` : modèles Markdown pour les pages source-note / concept / comparison / decision.
- `schema/references/` : règles d'édition — types de pages, règles de citation, règles de conflits, structure des répertoires.

Les prompts de distillation/synthèse lisent ces modèles, donc les modifier personnalise le format wiki généré par KB.

## Démarrage rapide

```bash
export PIEKBS_KB=/path/to/your-kb

piekbs init           # créer les répertoires KB et copier schema/modèles
piekbs serve          # démarrer le serveur : MCP + Web UI + watcher de fichiers
piekbs index          # construire/mettre à jour l'index FTS
piekbs status         # statistiques d'index
piekbs lint           # vérifier les pages wiki
```

## Référence des commandes

Toutes les commandes acceptent un flag global `--kb <path>` (par défaut `$PIEKBS_KB`, puis `~/piekbs-kb`).

| Commande | Description |
|---|---|
| `piekbs init [--force]` | Créer les répertoires KB et copier les schema/modèles intégrés. |
| `piekbs serve` | Démarrer le serveur longue durée : HTTP MCP (`/mcp`) + Web UI + watcher de fichiers. Par défaut sans sous-commande. |
| `piekbs index` | Construire/mettre à jour l'index FTS depuis le markdown `wiki/` et `raw/`. |
| `piekbs search <query>` | Recherche FTS par mots-clés ; affiche les résultats classés avec chemins et extraits. |
| `piekbs synthesize [--topic X] [--full]` | Générer des pages concept/comparison/decision depuis les source-notes. |
| `piekbs synthesize --gaps --topic X` | Analyse des lacunes de connaissances pour un sujet. |
| `piekbs import-lark <URL>` | Importer une page Lark/Feishu Wiki et ses tables intégrées dans `raw/lark/`. Nécessite un `lark-cli` connecté. |
| `piekbs lint` | Vérifier les pages wiki : champs frontmatter manquants, liens sources cassés. |
| `piekbs status` | Afficher les statistiques d'index (nombres de documents, taille d'index). |
| `piekbs service <install\|uninstall\|start\|stop\|status\|logs>` | Gérer le service OS (launchd / systemd). |

**Configuration LLM** (section `distill` dans `config.yaml` sous la racine KB) est requise pour `distill` et `synthesize`.

## Serveur MCP

PieKBS expose les outils KB via le protocole MCP.

**Outils disponibles :** `kb_search`, `kb_page`, `kb_add`

Les opérations d'administration (`status`, `reindex`, `lint`) sont disponibles via l'interface Web ou CLI (`piekbs status`, `piekbs index`, `piekbs lint`).

---

### Scénario 1 : Partage multi-agents local

Le mode HTTP est recommandé : un seul processus PieKBS partagé par tous les agents — Claude Code, Cursor, VS Code (Copilot), Windsurf, Trae, Codex, Hermes, OpenClaw et autres.

**Étape 1 : Démarrer PieKBS**

```bash
export PIEKBS_KB=/path/to/piekbs-kb
piekbs serve
```

> Sur macOS, double-cliquez sur PieKBS.app pour démarrer comme icône dans la barre de menus.

**Étape 2 : Configurer HTTP MCP dans chaque agent**

Ajouter à `~/.claude.json` sous `mcpServers` :

```json
{
  "mcpServers": {
    "piekbs": {
      "type": "http",
      "url": "http://127.0.0.1:8766/mcp",
      "headers": {
        "x-api-key": "${PIEKBS_API_KEY}"
      }
    }
  }
}
```

`x-api-key` correspond à `server.api_key` dans `config.yaml`. Omettez `headers` si aucun api_key n'est défini.

---

### Scénario 2 : Environnements d'agents hébergés

Dans les environnements hébergés (Hermes, OpenClaw, etc.), installez PieKBS sur le volume persistant et invoquez via **stdio** — PieKBS démarre comme sous-processus de l'hôte de l'agent, le watcher s'exécute automatiquement en arrière-plan.

Exemple (OpenClaw/Hermes monté sur NAS, point de montage `/root/.openclaw`) :

**1. Installer sur le volume persistant (une fois) :**

```bash
tar -xzf piekbs-linux-amd64.tar.gz -C /root/.openclaw/piekbs/
chmod +x /root/.openclaw/piekbs/piekbs
```

**2. Installer markitdown (recommandé) :**

markitdown permet la conversion de fichiers PDF, Word, Excel, PPT et HTML en Markdown avant la distillation. Sans lui, seuls les fichiers `.md` et `.txt` sont distillés ; les fichiers binaires sont indexés uniquement par nom de fichier.

```bash
pip install markitdown
# vérifier
markitdown --version
```

> Vérifié sur OpenClaw/Hermes (chemin : `/root/.openclaw/workspace/bin/markitdown`). Ajoutez `workspace/bin` au PATH ou définissez le chemin complet dans votre environnement.

Si markitdown n'est pas disponible, les agents peuvent extraire le texte eux-mêmes (avec LLM vision ou d'autres outils) et écrire le résultat directement dans `$PIEKBS_KB/raw/converted/<slug>.md` — le watcher le récupère automatiquement.

**3. Configuration MCP :**

Hermes (`mcp_servers` dans la config agent) :

```yaml
mcp_servers:
  piekbs:
    command: /root/.openclaw/piekbs/piekbs
    args: [stdio]
    env:
      PIEKBS_KB: /root/.openclaw/piekbs-kb
      PATH: /root/.openclaw/workspace/bin:/usr/local/bin:/usr/bin:/bin
```

Le répertoire KB est créé automatiquement au premier lancement. Pas de `init` manuel nécessaire.

**4. Ajouter du contenu à la base de connaissances :**

Les agents avec accès `write_file` peuvent écrire directement dans la KB — le watcher détecte les changements et déclenche automatiquement l'indexation et la distillation.

| Type de contenu | Écrire dans |
|---|---|
| Articles, notes, références (Markdown/texte) | `$PIEKBS_KB/raw/<votre-catégorie>/<slug>.md` |
| Contenu PDF/Word/Excel/EPUB converti par agent | `$PIEKBS_KB/raw/converted/<slug>.md` |

Les fichiers dans `raw/converted/` sont traités comme déjà convertis et vont directement à la distillation, sautant l'étape markitdown. Tous les autres chemins sous `raw/` sont traités par le pipeline complet (convertir → indexer → distiller).

Organisez les sous-répertoires selon ce qui est logique pour votre contenu — PieKBS n'impose pas de structure fixe sous `raw/`.

## Service système (optionnel)

`piekbs serve` inclut un watcher intégré qui surveille automatiquement le répertoire KB, déclenche la distillation et reconstruit l'index. Aucune configuration supplémentaire requise.

Pour que PieKBS **démarre au boot et s'exécute en arrière-plan**, installez-le comme service système (macOS launchd / Linux systemd) :

```bash
piekbs service install --kb /path/to/your-kb
piekbs service status
piekbs service uninstall
```

Journaux : `{PIEKBS_KB}/index/watcher.log`
