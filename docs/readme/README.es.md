<div align="center">
  <img src="logo.png" width="128" alt="WikiLoop"><br>
  <h1>WikiLoop</h1>
  <p>Un motor de búsqueda de conocimiento para agentes — destila documentos brutos en wiki Markdown estructurado, búsqueda y lectura vía MCP</p>
  <p>
    <a href="../../README.md">English</a> |
    <a href="README.zh-CN.md">简体中文</a> |
    <a href="README.zh-TW.md">繁體中文</a> |
    <a href="README.ru.md">Русский</a> |
    <a href="README.de.md">Deutsch</a> |
    <a href="README.fr.md">Français</a> |
    <strong>Español</strong> |
    <a href="README.ko.md">한국어</a>
  </p>
  <p>
    <a href="../../LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
    <a href="https://github.com/jasen215/wikiloop/releases"><img src="https://img.shields.io/github/v/release/jasen215/wikiloop" alt="Release"></a>
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue" alt="Platform">
  </p>
</div>

WikiLoop es un motor de búsqueda de conocimiento local para agentes. Destila documentos brutos en una base de conocimiento Markdown estructurada y verificable, luego expone dos herramientas MCP — `kb_search` y `kb_page` — que permiten a los agentes buscar y leer en profundidad a su propio ritmo.

![WikiLoop Screenshot](image-001.png)

## Filosofía de diseño

WikiLoop se basa en una observación: **los agentes usan las herramientas de conocimiento externas de la misma manera que los humanos usan los motores de búsqueda** — realizan múltiples consultas desde diferentes ángulos, siguen enlaces y sintetizan sus propias conclusiones. No quieren una respuesta pre-empaquetada; quieren los materiales brutos para formar sus propias conclusiones.

Esto significa que la misión de WikiLoop no es responder preguntas. Es asegurarse de que cuando un agente busca algo, encuentre los documentos correctos — y pueda leerlos en su totalidad.

```text
wikiloop-kb/
  raw/                  Fuente de verdad — materiales originales en cualquier formato.
                        Deposite archivos aquí; el watcher los destila automáticamente.

  wiki/                 Capa de conocimiento Markdown estructurada (mantenida por LLM).
    source-notes/       Una nota destilada por documento fuente. Objetivo de búsqueda FTS.
    concepts/           Síntesis entre documentos: conceptos y metodologías.
    comparisons/        Síntesis entre documentos: comparaciones lado a lado.
    decisions/          Síntesis entre documentos: decisiones técnicas.
    _draft/             Páginas sintetizadas con < 2 fuentes (aún no indexadas).

  schema/               Reglas de autoría y plantillas de página locales de la KB.
                        Edite para personalizar el formato de páginas destiladas.

  index/                Artefactos generados (índice SQLite FTS, registros de consultas).
                        Gestionado automáticamente — no editar manualmente.
```

## Cómo usan los agentes WikiLoop

Los agentes interactúan con WikiLoop a través de dos herramientas MCP:

**`kb_search(query, limit?)`** — Buscar con una palabra clave o frase. Devuelve hasta 5 source-notes y 3 páginas concept/comparison/decision por llamada. Cada resultado incluye un campo `related` con documentos relacionados para navegación. Use múltiples búsquedas con diferentes palabras clave para cubrir un tema desde múltiples ángulos.

**`kb_page(ids, full?)`** — Recupera el contenido completo de una o más páginas por ID (de los resultados de `kb_search`). Pase hasta 5 IDs para escanear varios documentos a la vez, o `full=true` con un solo ID para obtener el texto completo sin truncar.

Flujo de trabajo recomendado del agente:

```text
kb_search("palabra clave A")        → descubrir documentos relevantes
kb_search("palabra clave B")        → cubrir otro ángulo
kb_page(["id1", "id2", "id3"])      → leer en profundidad los más relevantes
El agente sintetiza su propia respuesta a partir de lo encontrado
```

Se espera que los agentes busquen iterativamente, sigan los enlaces `related`, verifiquen fuentes cruzadas y formen sus propias conclusiones. WikiLoop no genera respuestas.

## WikiLoop vs RAG

El RAG tradicional recupera contexto y lo entrega al LLM para responder. WikiLoop entrega los materiales brutos al agente y deja que el agente razone por sí mismo.

```text
RAG:       pregunta usuario → recuperar contexto → LLM responde
WikiLoop:  agente busca → agente lee → agente sintetiza
```

| | RAG | WikiLoop |
|---|---|---|
| Forma del conocimiento | Implícita (vectores o fragmentos) | Explícita (Markdown, verificable) |
| Rol del agente | Receptor pasivo de contexto | Buscador y lector activo |
| Fuente de respuesta | Generada por el sistema | Sintetizada por el agente |
| Verificable | No | Sí — git diff, lint, enlaces de conflictos |
| Razonamiento multi-salto | Dependiente del LLM | Expansión de grafo vía enlaces `related` |
| Embedding | Requerido | No requerido (FTS puro) |

Los bundles de WikiLoop son conformes con [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf).

## Pipeline de conocimiento

Los documentos brutos pasan por un pipeline de destilación antes de que los agentes puedan buscarlos:

**Paso 1 — Destilar (automático)**

Deposite cualquier archivo Markdown en `raw/`. El watcher de `wikiloop serve` ejecuta automáticamente destilación + indexación. El LLM extrae source-notes estructuradas en `wiki/source-notes/`, incluyendo:
- `key_claims` con alias integrados y equivalentes entre idiomas (ALIAS RULE) — garantiza que FTS coincida con todas las variantes de consulta
- Anotaciones de entidades nombradas en formato `【entity|type】`
- Enlaces `related_to`, `supports`, `contradicts` — alimenta el campo `related` en los resultados de búsqueda
- Metadatos `authority` (1–5) y `doc_type`

**Paso 2 — Sintetizar (bajo demanda)**

```bash
wikiloop synthesize --topic "RAG"
```

Genera páginas concept/comparison/decision a partir de source-notes cuando se acumulan suficientes fuentes sobre un tema. Las páginas con menos de 2 referencias de fuentes van a `wiki/<type>/_draft/` y no se indexan hasta que se agregan más fuentes.

**Paso 3 — Buscar**

Los agentes usan `kb_search` + `kb_page` vía MCP. La búsqueda es FTS puro (SQLite FTS5 con puntuación BM25). No se requiere modelo vectorial.

## Instalación

Descargar la última versión:

| Plataforma | Archivo |
|---|---|
| macOS Apple Silicon (ARM64) | `WikiLoop-<version>-macos-arm64.dmg` |
| Linux x86_64 | `wikiloop-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `wikiloop-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `wikiloop-<version>-windows-amd64.zip` |

> **macOS Intel (x86_64):** Sin versión precompilada. GitHub Actions eliminó el runner Intel macOS en abril de 2025. Compile desde el código fuente en su Mac Intel: `CGO_ENABLED=1 go build -tags fts5 -o wikiloop ./cmd/wikiloop/`

**macOS:** Abra el DMG y arrastre WikiLoop a Aplicaciones. La app funciona como icono en la barra de menús.

**Linux:**
```bash
tar -xzf wikiloop-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/wikiloop /usr/local/bin/wikiloop
```

**Windows:** Extraiga el zip y ejecute `wikiloop.exe serve` (o `wikiloop.exe stdio` para MCP). Agregue el directorio al `PATH`. No se requiere CGO — binario Go puro.

**HarmonyOS PC (comunidad, experimental):** WikiLoop no se lanza oficialmente para HarmonyOS PC. Sin embargo, dado que el binario principal no requiere CGO (Go puro + SQLite), puede compilarse nativamente en HarmonyOS con el gestor de paquetes comunitario [Harmonybrew](https://harmonybrew.dev). Ver [ohos_go_cgo](https://github.com/ohos-go/ohos_go_cgo) para una guía sobre la configuración de Go + CGO en HarmonyOS PC.

```bash
# En HarmonyOS PC (después de instalar Go via Harmonybrew)
CGO_ENABLED=0 go build -tags fts5 -o wikiloop ./cmd/wikiloop/
wikiloop serve
```

## Compilar desde el código fuente

Requiere Go 1.25+. No se requiere CGO.

```bash
# macOS / Linux
go build -tags fts5 -o wikiloop ./cmd/wikiloop/

# Windows
go build -tags fts5 -o wikiloop.exe ./cmd/wikiloop/
```

O usar el script de compilación multiplataforma:

```bash
./scripts/build.sh [version] [target...]
```

| Target | Salida | Plataforma |
|---|---|---|
| `darwin-arm64` | `dist/WikiLoop-<version>-macos-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/wikiloop-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/wikiloop-<version>-linux-arm64.tar.gz` | Linux ARM64 |
| `windows-amd64` | `dist/wikiloop-<version>-windows-amd64.zip` | Windows x86_64 |

## Estructura del repositorio

```text
wikiloop/
  cmd/wikiloop/        # punto de entrada principal
  internal/
    kb/                # indexación FTS, búsqueda, expansión de grafo, recuperación de páginas
    mcp/               # servidor MCP (stdio + HTTP)
    watcher/           # watcher de archivos para auto-destilación + reindexación
    distill/           # pipeline de destilación LLM
    synthesize/        # generación de páginas concept/comparison/decision
    convert/           # conversión de archivos brutos
    service/           # gestor de servicios del SO (launchd / systemd)
    webui/             # interfaz web
    tray/              # bandeja del sistema macOS (solo darwin)
    config/            # configuración KB (config.yaml)
  scripts/
    build.sh           # script de compilación multiplataforma
```

## Esquema y plantillas

`wikiloop init` rellena el directorio `schema/` de la KB con reglas de autoría y plantillas de página integradas:

- `schema/templates/`: plantillas Markdown para páginas source-note / concept / comparison / decision.
- `schema/references/`: reglas de autoría — tipos de página, reglas de citación, reglas de conflictos, estructura de directorios.

Los prompts de destilación/síntesis leen estas plantillas, por lo que editarlas personaliza el formato wiki generado por KB.

## Inicio rápido

```bash
export WIKILOOP_KB=/path/to/your-kb

wikiloop init           # crear directorios KB y copiar schema/plantillas
wikiloop serve          # iniciar servidor: MCP + Web UI + watcher de archivos
wikiloop index          # construir/actualizar índice FTS
wikiloop status         # estadísticas del índice
wikiloop lint           # verificar páginas wiki
```

## Referencia de comandos

Todos los comandos aceptan un flag global `--kb <path>` (por defecto `$WIKILOOP_KB`, luego `~/wikiloop-kb`).

| Comando | Descripción |
|---|---|
| `wikiloop init [--force]` | Crear directorios KB y copiar schema/plantillas integradas. |
| `wikiloop serve` | Iniciar servidor de larga duración: HTTP MCP (`/mcp`) + Web UI + watcher de archivos. Por defecto sin subcomando. |
| `wikiloop index` | Construir/actualizar índice FTS desde markdown de `wiki/` y `raw/`. |
| `wikiloop search <query>` | Búsqueda FTS por palabras clave; imprime resultados ordenados con rutas y fragmentos. |
| `wikiloop synthesize [--topic X] [--full]` | Generar páginas concept/comparison/decision desde source-notes. |
| `wikiloop synthesize --gaps --topic X` | Análisis de brechas de conocimiento para un tema. |
| `wikiloop import-lark <URL>` | Importar una página Lark/Feishu Wiki y sus tablas integradas en `raw/lark/`. Requiere `lark-cli` autenticado. |
| `wikiloop lint` | Verificar páginas wiki: campos frontmatter faltantes, enlaces de fuentes rotos. |
| `wikiloop status` | Imprimir estadísticas del índice (recuentos de documentos, tamaño del índice). |
| `wikiloop service <install\|uninstall\|start\|stop\|status\|logs>` | Gestionar servicio del SO (launchd / systemd). |

**Configuración LLM** (sección `distill` en `config.yaml` bajo la raíz KB) es necesaria para `distill` y `synthesize`.

## Servidor MCP

WikiLoop expone herramientas KB a través del protocolo MCP.

**Herramientas disponibles:** `kb_search`, `kb_page`

Las operaciones de administración (`status`, `reindex`, `lint`) están disponibles vía la interfaz Web o CLI (`wikiloop status`, `wikiloop index`, `wikiloop lint`).

---

### Escenario 1: Compartir multi-agente local

El modo HTTP es recomendado: un proceso WikiLoop compartido por todos los agentes — Claude Code, Cursor, VS Code (Copilot), Windsurf, Trae, Codex, Hermes, OpenClaw y otros.

**Paso 1: Iniciar WikiLoop**

```bash
export WIKILOOP_KB=/path/to/wikiloop-kb
wikiloop serve
```

> En macOS, haga doble clic en WikiLoop.app para iniciar como icono en la barra de menús.

**Paso 2: Configurar HTTP MCP en cada agente**

Agregar a `~/.claude.json` bajo `mcpServers`:

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

`x-api-key` corresponde a `server.api_key` en `config.yaml`. Omita `headers` si no se establece api_key.

---

### Escenario 2: Entornos de agentes alojados

En entornos alojados (Hermes, OpenClaw, etc.), instale WikiLoop en el volumen persistente e invoque vía **stdio** — WikiLoop inicia como subproceso del host del agente, el watcher se ejecuta automáticamente en segundo plano.

Ejemplo (OpenClaw/Hermes montado en NAS, punto de montaje `/root/.openclaw`):

**1. Instalar en volumen persistente (una vez):**

```bash
tar -xzf wikiloop-linux-amd64.tar.gz -C /root/.openclaw/wikiloop/
chmod +x /root/.openclaw/wikiloop/wikiloop
```

**2. Instalar markitdown (recomendado):**

markitdown permite la conversión de archivos PDF, Word, Excel, PPT y HTML a Markdown antes de la destilación. Sin él, solo se destilan archivos `.md` y `.txt`; los archivos binarios se indexan solo por nombre de archivo.

```bash
pip install markitdown
# verificar
markitdown --version
```

> Verificado en OpenClaw/Hermes (ruta: `/root/.openclaw/workspace/bin/markitdown`). Agregue `workspace/bin` al PATH o establezca la ruta completa en su entorno.

Si markitdown no está disponible, los agentes pueden extraer texto ellos mismos (usando LLM vision u otras herramientas) y escribir el resultado directamente en `$WIKILOOP_KB/raw/converted/<slug>.md` — el watcher lo recoge automáticamente.

**3. Configuración MCP:**

Hermes (`mcp_servers` en la configuración del agente):

```yaml
mcp_servers:
  wikiloop:
    command: /root/.openclaw/wikiloop/wikiloop
    args: [stdio]
    env:
      WIKILOOP_KB: /root/.openclaw/wikiloop-kb
      PATH: /root/.openclaw/workspace/bin:/usr/local/bin:/usr/bin:/bin
```

El directorio KB se crea automáticamente en el primer lanzamiento. No se necesita `init` manual.

**4. Agregar contenido a la base de conocimiento:**

Los agentes con acceso `write_file` pueden escribir directamente en la KB — el watcher detecta cambios y activa automáticamente la indexación y destilación.

| Tipo de contenido | Escribir en |
|---|---|
| Artículos, notas, referencias (Markdown/texto) | `$WIKILOOP_KB/raw/<su-categoría>/<slug>.md` |
| Contenido PDF/Word/Excel/EPUB convertido por agente | `$WIKILOOP_KB/raw/converted/<slug>.md` |

Los archivos en `raw/converted/` se tratan como ya convertidos y van directamente a la destilación, omitiendo el paso de markitdown. Todas las demás rutas bajo `raw/` se procesan a través del pipeline completo (convertir → indexar → destilar).

Organice los subdirectorios según lo que tenga sentido para su contenido — WikiLoop no impone una estructura fija bajo `raw/`.

## Servicio del sistema (opcional)

`wikiloop serve` incluye un watcher integrado que monitorea automáticamente el directorio KB, activa la destilación y reconstruye el índice. No se requiere configuración adicional.

Para que WikiLoop **inicie al arrancar y se ejecute en segundo plano**, instálelo como servicio del sistema (macOS launchd / Linux systemd):

```bash
wikiloop service install --kb /path/to/your-kb
wikiloop service status
wikiloop service uninstall
```

Registros: `{WIKILOOP_KB}/index/watcher.log`
