<div align="center">
  <img src="logo.png" width="128" alt="PieKBS"><br>
  <h1>PieKBS</h1>
  <p>Поисковый движок знаний для агентов — дистиллирует исходные документы в структурированную Markdown-базу знаний, поиск и чтение через MCP</p>
  <p>
    <a href="../../README.md">English</a> |
    <a href="README.zh-CN.md">简体中文</a> |
    <a href="README.zh-TW.md">繁體中文</a> |
    <strong>Русский</strong> |
    <a href="README.de.md">Deutsch</a> |
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

PieKBS — это локальный поисковый движок знаний для агентов. Он дистиллирует исходные документы в структурированную, проверяемую Markdown-базу знаний, предоставляя два MCP-инструмента — `kb_search` и `kb_page` — которые позволяют агентам искать и углублённо читать материалы в своём темпе.

![PieKBS Screenshot](image-001.png)

## Философия дизайна

PieKBS основан на одном ключевом наблюдении: **агенты используют внешние инструменты знаний так же, как люди используют поисковые системы** — делают несколько запросов с разных сторон, переходят по связанным ссылкам и самостоятельно синтезируют выводы. Им не нужен готовый ответ; им нужны исходные материалы для формирования собственных заключений.

Это означает, что задача PieKBS — не отвечать на вопросы. Она состоит в том, чтобы когда агент ищет что-то, он находил нужные документы и мог читать их полностью.

```text
piekbs-kb/
  raw/                  Источник истины — исходные материалы в любом формате.
                        Положите файлы сюда; наблюдатель автоматически дистиллирует их.

  wiki/                 Структурированный слой знаний Markdown (поддерживается LLM).
    source-notes/       Одна дистиллированная заметка на исходный документ. Цель FTS-поиска.
    concepts/           Синтез по документам: концепции и методологии.
    comparisons/        Синтез по документам: сравнения рядом.
    decisions/          Синтез по документам: технические решения.
    _draft/             Синтезированные страницы с < 2 источниками (ещё не проиндексированы).

  schema/               Локальные правила написания и шаблоны страниц базы знаний.
                        Редактируйте для настройки формата дистиллированных страниц.

  index/                Сгенерированные артефакты (SQLite FTS-индекс, журналы запросов).
                        Управляется автоматически — не редактировать вручную.
```

## Как агенты используют PieKBS

Агенты взаимодействуют с PieKBS через три MCP-инструмента:

**`kb_search(query, limit?)`** — Поиск по ключевому слову или фразе. Возвращает до 5 source-notes и 3 страниц concept/comparison/decision за вызов. Каждый результат содержит поле `related` со списком связанных документов для навигации. Используйте несколько поисков с разными ключевыми словами для охвата темы с разных сторон.

**`kb_page(ids, full?)`** — Получение полного содержимого одной или нескольких страниц по ID (из результатов `kb_search`). Передавайте до 5 ID для просмотра нескольких документов сразу, или `full=true` с одним ID для получения полного неусечённого текста.

**`kb_add(filename, content, source_url?)`** — Добавляет текстовый документ в базу знаний. Записывает содержимое в `raw/<filename>` и запускает инкрементальную индексацию. Дистилляция выполняется асинхронно в фоновом режиме. Используйте префикс `converted/` для контента, извлечённого агентами из PDF/Word/Excel/EPUB.

Рекомендуемый рабочий процесс агента:

```text
kb_search("ключевое слово A")       → обнаружить релевантные документы
kb_search("ключевое слово B")       → охватить другой угол
kb_page(["id1", "id2", "id3"])      → углублённо прочитать наиболее релевантные
Агент синтезирует собственный ответ из найденного
```

Агенты должны искать итеративно, переходить по ссылкам `related`, перекрёстно проверять источники и формулировать собственные выводы. PieKBS не генерирует ответы.

## PieKBS vs RAG

Традиционный RAG извлекает контекст и передаёт его LLM для ответа. PieKBS передаёт агенту исходные материалы и позволяет агенту самому рассуждать.

```text
RAG:       вопрос пользователя → извлечение контекста → LLM отвечает
PieKBS:  агент ищет → агент читает → агент синтезирует
```

| | RAG | PieKBS |
|---|---|---|
| Форма знаний | Неявная (векторы или фрагменты) | Явная (Markdown, проверяемая) |
| Роль агента | Пассивный получатель контекста | Активный исследователь и читатель |
| Источник ответа | Сгенерированный системой | Синтезированный агентом |
| Проверяемость | Нет | Да — git diff, lint, ссылки на конфликты |
| Многоходовое рассуждение | Зависит от LLM | Расширение графа через ссылки `related` |
| Embedding | Требуется | Не требуется (чистый FTS) |

Пакеты PieKBS соответствуют [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf).

## Конвейер знаний

Исходные документы проходят через конвейер дистилляции, прежде чем агенты смогут их искать:

**Шаг 1 — Дистилляция (автоматически)**

Поместите любой Markdown-файл в `raw/`. Наблюдатель `piekbs serve` автоматически запустит дистилляцию + индексирование. LLM извлекает структурированные source-notes в `wiki/source-notes/`, включая:
- `key_claims` со встроенными псевдонимами и кросс-языковыми эквивалентами (ALIAS RULE) — обеспечивает совпадение FTS со всеми вариантами запросов
- Аннотации именованных сущностей в формате `【entity|type】`
- Ссылки `related_to`, `supports`, `contradicts` — питают поле `related` в результатах поиска
- Метаданные `authority` (1–5) и `doc_type`

**Шаг 2 — Синтез (по требованию)**

```bash
piekbs synthesize --topic "RAG"
```

Генерирует страницы concept/comparison/decision из source-notes, когда по теме накапливается достаточно источников. Страницы с менее чем 2 ссылками на источники попадают в `wiki/<type>/_draft/` и не индексируются до добавления новых источников.

**Шаг 3 — Поиск**

Агенты используют `kb_search` + `kb_page` через MCP. Поиск — чистый FTS (SQLite FTS5 с оценкой BM25). Векторная модель не требуется.

## Установка

Скачайте последний релиз:

| Платформа | Файл |
|---|---|
| macOS Apple Silicon (ARM64) | `PieKBS-<version>-macos-arm64.dmg` |
| Linux x86_64 | `piekbs-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `piekbs-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `piekbs-<version>-windows-amd64.zip` |

> **macOS Intel (x86_64):** Готового релиза нет. GitHub Actions отказался от Intel macOS runner в апреле 2025. Соберите из исходников на вашем Intel Mac: `CGO_ENABLED=1 go build -tags fts5 -o piekbs ./cmd/piekbs/`

**macOS:** Откройте DMG и перетащите PieKBS в Applications. Приложение работает как иконка в строке меню.

**Linux:**
```bash
tar -xzf piekbs-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/piekbs /usr/local/bin/piekbs
```

**Windows:** Распакуйте zip и запустите `piekbs.exe serve` (или `piekbs.exe stdio` для MCP). Добавьте директорию в `PATH`. CGO не требуется — чистый Go-бинарник.

**HarmonyOS PC (сообщество, экспериментально):** PieKBS официально не выпущен для HarmonyOS PC. Однако, поскольку основной бинарник не требует CGO (чистый Go + SQLite), его можно собрать нативно на HarmonyOS с помощью пакетного менеджера сообщества [Harmonybrew](https://harmonybrew.dev). См. [ohos_go_cgo](https://github.com/ohos-go/ohos_go_cgo) для настройки Go + CGO на HarmonyOS PC.

```bash
# На HarmonyOS PC (после установки Go через Harmonybrew)
CGO_ENABLED=0 go build -tags fts5 -o piekbs ./cmd/piekbs/
piekbs serve
```

## Сборка из исходников

Требуется Go 1.25+. CGO не требуется.

```bash
# macOS / Linux
go build -tags fts5 -o piekbs ./cmd/piekbs/

# Windows
go build -tags fts5 -o piekbs.exe ./cmd/piekbs/
```

Или используйте скрипт многоплатформенной сборки:

```bash
./scripts/build.sh [version] [target...]
```

| Target | Вывод | Платформа |
|---|---|---|
| `darwin-arm64` | `dist/PieKBS-<version>-macos-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/piekbs-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/piekbs-<version>-linux-arm64.tar.gz` | Linux ARM64 |
| `windows-amd64` | `dist/piekbs-<version>-windows-amd64.zip` | Windows x86_64 |

## Структура репозитория

```text
piekbs/
  cmd/piekbs/        # главная точка входа
  internal/
    kb/                # FTS-индексирование, поиск, расширение графа, получение страниц
    mcp/               # MCP-сервер (stdio + HTTP)
    watcher/           # наблюдатель файлов для авто-дистилляции + переиндексирования
    distill/           # конвейер дистилляции LLM
    synthesize/        # генерация страниц concept/comparison/decision
    convert/           # конвертация исходных файлов
    service/           # менеджер ОС-сервисов (launchd / systemd)
    webui/             # веб-интерфейс
    tray/              # системный трей macOS (только darwin)
    config/            # конфигурация базы знаний (config.yaml)
  scripts/
    build.sh           # скрипт многоплатформенной сборки
```

## Схема и шаблоны

`piekbs init` заполняет директорию `schema/` базы знаний встроенными правилами написания и шаблонами страниц:

- `schema/templates/`: Markdown-шаблоны для страниц source-note / concept / comparison / decision.
- `schema/references/`: правила написания — типы страниц, правила цитирования, правила конфликтов, структура директорий.

Промпты distill/synthesize читают эти шаблоны, поэтому их редактирование настраивает формат генерируемой вики для каждой базы знаний.

## Быстрый старт

```bash
export WIKILOOP_KB=/path/to/your-kb

piekbs init           # создать структуру базы знаний и скопировать schema/шаблоны
piekbs serve          # запустить сервер: MCP + Web UI + наблюдатель файлов
piekbs index          # собрать/обновить FTS-индекс
piekbs status         # статистика индекса
piekbs lint           # проверка wiki-страниц
```

## Справочник команд

Все команды принимают глобальный флаг `--kb <path>` (по умолчанию `$WIKILOOP_KB`, затем `~/piekbs-kb`).

| Команда | Описание |
|---|---|
| `piekbs init [--force]` | Создать структуру базы знаний и скопировать встроенные schema/шаблоны. |
| `piekbs serve` | Запустить длительно работающий сервер: HTTP MCP (`/mcp`) + Web UI + наблюдатель файлов. По умолчанию при отсутствии подкоманды. |
| `piekbs index` | Собрать/обновить FTS-индекс из `wiki/` и `raw/` markdown. |
| `piekbs search <query>` | FTS-поиск по ключевым словам; выводит ранжированные результаты с путями и фрагментами. |
| `piekbs synthesize [--topic X] [--full]` | Генерировать страницы concept/comparison/decision из source-notes. |
| `piekbs synthesize --gaps --topic X` | Анализ пробелов в знаниях по теме. |
| `piekbs import-lark <URL>` | Импортировать страницу Lark/Feishu Wiki и её встроенные таблицы в `raw/lark/`. Требует авторизованного `lark-cli`. |
| `piekbs lint` | Проверка wiki-страниц: отсутствующие поля frontmatter, битые ссылки на источники. |
| `piekbs status` | Вывод статистики индекса (количество документов, размер индекса). |
| `piekbs service <install\|uninstall\|start\|stop\|status\|logs>` | Управление ОС-сервисом (launchd / systemd). |

**Конфигурация LLM** (секция `distill` в `config.yaml` под корнем базы знаний) требуется для `distill` и `synthesize`.

## MCP-сервер

PieKBS предоставляет инструменты базы знаний через протокол MCP.

**Доступные инструменты:** `kb_search`, `kb_page`, `kb_add`

Административные операции (`status`, `reindex`, `lint`) доступны через Web UI или CLI (`piekbs status`, `piekbs index`, `piekbs lint`).

---

### Сценарий 1: Локальный многоагентный обмен

Рекомендуется режим HTTP: один процесс PieKBS, общий для всех агентов — Claude Code, Cursor, VS Code (Copilot), Windsurf, Trae, Codex, Hermes, OpenClaw и других.

**Шаг 1: Запустить PieKBS**

```bash
export WIKILOOP_KB=/path/to/piekbs-kb
piekbs serve
```

> На macOS можно дважды кликнуть PieKBS.app для запуска как иконки в строке меню.

**Шаг 2: Настроить HTTP MCP в каждом агенте**

Добавьте в `~/.claude.json` под `mcpServers`:

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

`x-api-key` соответствует `server.api_key` в `config.yaml`. Опустите `headers`, если api_key не задан.

---

### Сценарий 2: Размещённые агентные среды

В размещённых средах (Hermes, OpenClaw и т.д.) установите PieKBS на постоянный том и вызывайте через **stdio** — PieKBS запускается как дочерний процесс хоста агента, наблюдатель работает в фоне автоматически.

Пример (OpenClaw/Hermes на NAS, точка монтирования `/root/.openclaw`):

**1. Установить на постоянный том (однократно):**

```bash
tar -xzf piekbs-linux-amd64.tar.gz -C /root/.openclaw/piekbs/
chmod +x /root/.openclaw/piekbs/piekbs
```

**2. Установить markitdown (рекомендуется):**

markitdown позволяет конвертировать PDF, Word, Excel, PPT и HTML файлы в Markdown перед дистилляцией. Без него дистиллируются только файлы `.md` и `.txt`; бинарные файлы индексируются только по имени.

```bash
pip install markitdown
# проверить
markitdown --version
```

> Проверено на OpenClaw/Hermes (путь: `/root/.openclaw/workspace/bin/markitdown`). Добавьте `workspace/bin` в PATH или укажите полный путь в вашем окружении.

Если markitdown недоступен, агенты могут самостоятельно извлечь текст (с помощью LLM vision или других инструментов) и записать результат напрямую в `$WIKILOOP_KB/raw/converted/<slug>.md` — наблюдатель автоматически подхватит файл.

**3. Конфигурация MCP:**

Hermes (`mcp_servers` в конфиге агента):

```yaml
mcp_servers:
  piekbs:
    command: /root/.openclaw/piekbs/piekbs
    args: [stdio]
    env:
      WIKILOOP_KB: /root/.openclaw/piekbs-kb
      PATH: /root/.openclaw/workspace/bin:/usr/local/bin:/usr/bin:/bin
```

Директория базы знаний создаётся автоматически при первом запуске. Ручной `init` не нужен.

**4. Добавление контента в базу знаний:**

Агенты с доступом `write_file` могут писать напрямую в базу знаний — наблюдатель обнаруживает изменения и автоматически запускает индексирование и дистилляцию.

| Тип контента | Записывать в |
|---|---|
| Статьи, заметки, справочные материалы (Markdown/текст) | `$WIKILOOP_KB/raw/<ваша-категория>/<slug>.md` |
| Контент PDF/Word/Excel/EPUB, конвертированный агентом | `$WIKILOOP_KB/raw/converted/<slug>.md` |

Файлы в `raw/converted/` рассматриваются как уже конвертированные и отправляются прямо на дистилляцию, минуя шаг markitdown. Все остальные пути под `raw/` обрабатываются через полный конвейер (конвертация → индексирование → дистилляция).

Организуйте поддиректории так, как это имеет смысл для вашего контента — PieKBS не навязывает фиксированную структуру под `raw/`.

## Системный сервис (опционально)

`piekbs serve` включает встроенный наблюдатель, который автоматически отслеживает директорию базы знаний, запускает дистилляцию и перестраивает индекс. Дополнительная настройка не требуется.

Чтобы PieKBS **запускался при загрузке и работал в фоне**, установите его как системный сервис (macOS launchd / Linux systemd):

```bash
piekbs service install --kb /path/to/your-kb
piekbs service status
piekbs service uninstall
```

Журналы: `{WIKILOOP_KB}/index/watcher.log`
