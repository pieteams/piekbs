<div align="center">
  <img src="logo.png" width="128" alt="WikiLoop"><br>
  <h1>WikiLoop</h1>
  <p>에이전트를 위한 지식 검색 엔진 — 원본 문서를 구조화된 Markdown 위키로 정제하고, MCP를 통해 검색 및 읽기</p>
  <p>
    <a href="../../README.md">English</a> |
    <a href="README.zh-CN.md">简体中文</a> |
    <a href="README.zh-TW.md">繁體中文</a> |
    <a href="README.ru.md">Русский</a> |
    <a href="README.de.md">Deutsch</a> |
    <a href="README.fr.md">Français</a> |
    <a href="README.es.md">Español</a> |
    <strong>한국어</strong>
  </p>
  <p>
    <a href="../../LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
    <a href="https://github.com/jasen215/wikiloop/releases"><img src="https://img.shields.io/github/v/release/jasen215/wikiloop" alt="Release"></a>
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue" alt="Platform">
  </p>
</div>

WikiLoop는 에이전트를 위한 로컬 우선 지식 검색 엔진입니다. 원본 문서를 구조화되고 검토 가능한 Markdown 지식 베이스로 정제한 다음, 두 가지 MCP 도구 — `kb_search`와 `kb_page` — 를 통해 에이전트가 자신의 속도에 맞춰 검색하고 깊이 읽을 수 있게 합니다.

![WikiLoop Screenshot](image-001.png)

## 설계 철학

WikiLoop는 하나의 관찰에서 출발합니다: **에이전트는 외부 지식 도구를 인간이 검색 엔진을 사용하는 것처럼 활용합니다** — 다양한 각도에서 여러 쿼리를 날리고, 링크를 따라가며, 스스로 결론을 종합합니다. 미리 포장된 답을 원하는 것이 아니라, 스스로 결론을 도출할 수 있는 원자재가 필요합니다.

따라서 WikiLoop의 역할은 질문에 답하는 것이 아닙니다. 에이전트가 무언가를 검색할 때 올바른 문서를 찾고 — 전체 내용을 읽을 수 있도록 보장하는 것입니다.

```text
wikiloop-kb/
  raw/                  진실의 원천 — 모든 형식의 원본 자료.
                        파일을 여기에 넣으면 watcher가 자동으로 정제합니다.

  wiki/                 구조화된 Markdown 지식 계층 (LLM 유지).
    source-notes/       원본 문서당 하나의 정제된 노트. FTS 검색 대상.
    concepts/           문서 간 합성: 개념 및 방법론.
    comparisons/        문서 간 합성: 나란히 비교.
    decisions/          문서 간 합성: 기술적 결정.
    _draft/             < 2개 소스인 합성 페이지 (아직 색인되지 않음).

  schema/               KB 로컬 저작 규칙 및 페이지 템플릿.
                        정제된 페이지 형식을 커스터마이즈하려면 편집하세요.

  index/                생성된 아티팩트 (SQLite FTS 인덱스, 쿼리 로그).
                        자동으로 관리됨 — 수동으로 편집하지 마세요.
```

## 에이전트가 WikiLoop를 사용하는 방법

에이전트는 두 가지 MCP 도구를 통해 WikiLoop와 상호작용합니다:

**`kb_search(query, limit?)`** — 키워드나 문구로 검색합니다. 호출당 최대 5개의 source-note와 3개의 concept/comparison/decision 페이지를 반환합니다. 각 결과에는 탐색을 위한 관련 문서 목록이 담긴 `related` 필드가 포함됩니다. 여러 다른 키워드로 여러 번 검색하여 주제를 다양한 각도에서 다룹니다.

**`kb_page(ids, full?)`** — ID(kb_search 결과에서)로 하나 이상의 페이지 전체 내용을 가져옵니다. 최대 5개의 ID를 전달하여 여러 문서를 한 번에 스캔하거나, 단일 ID와 `full=true`를 사용하여 완전한 전체 텍스트를 가져옵니다.

권장 에이전트 워크플로우:

```text
kb_search("키워드 A")               → 관련 문서 발견
kb_search("키워드 B")               → 다른 각도 다루기
kb_page(["id1", "id2", "id3"])      → 가장 관련성 높은 것을 깊이 읽기
에이전트가 찾은 내용을 바탕으로 스스로 답변 합성
```

에이전트는 반복적으로 검색하고, `related` 링크를 따라가며, 출처를 교차 검증하고, 스스로 결론을 내릴 것으로 예상됩니다. WikiLoop는 답변을 생성하지 않습니다.

## WikiLoop vs RAG

전통적인 RAG는 컨텍스트를 검색하여 LLM에게 답변하도록 전달합니다. WikiLoop는 에이전트에게 원자재를 전달하고 에이전트 스스로 추론하게 합니다.

```text
RAG:       사용자 질문 → 컨텍스트 검색 → LLM 답변
WikiLoop:  에이전트 검색 → 에이전트 읽기 → 에이전트 합성
```

| | RAG | WikiLoop |
|---|---|---|
| 지식 형태 | 암묵적 (벡터 또는 청크) | 명시적 (Markdown, 검증 가능) |
| 에이전트 역할 | 컨텍스트의 수동적 수신자 | 능동적 검색자 및 독자 |
| 답변 출처 | 시스템 생성 | 에이전트 합성 |
| 검증 가능 | 아니오 | 예 — git diff, lint, 충돌 링크 |
| 멀티홉 추론 | LLM 의존 | `related` 링크를 통한 그래프 확장 |
| 임베딩 | 필요 | 불필요 (순수 FTS) |

WikiLoop 번들은 [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf)을 준수합니다.

## 지식 파이프라인

원본 문서는 에이전트가 검색하기 전에 정제 파이프라인을 통과합니다:

**단계 1 — 정제 (자동)**

`raw/`에 Markdown 파일을 넣습니다. `wikiloop serve` watcher가 자동으로 정제 + 인덱싱을 실행합니다. LLM은 `wiki/source-notes/`에 구조화된 source-note를 추출합니다. 포함 내용:
- `key_claims`에 인라인 별칭 및 교차 언어 동등어 포함 (ALIAS RULE) — FTS가 모든 쿼리 변형과 일치하도록 보장
- `【entity|type】` 형식의 명명된 엔티티 주석
- `related_to`, `supports`, `contradicts` 링크 — 검색 결과의 `related` 필드 구동
- `authority` (1–5) 및 `doc_type` 메타데이터

**단계 2 — 합성 (필요 시)**

```bash
wikiloop synthesize --topic "RAG"
```

주제에 충분한 소스가 축적되면 source-note에서 concept/comparison/decision 페이지를 생성합니다. 소스 참조가 2개 미만인 페이지는 `wiki/<type>/_draft/`에 들어가며 소스가 더 추가될 때까지 인덱싱되지 않습니다.

**단계 3 — 검색**

에이전트는 MCP를 통해 `kb_search` + `kb_page`를 사용합니다. 검색은 순수 FTS (BM25 점수를 사용한 SQLite FTS5)입니다. 벡터 모델이 필요하지 않습니다.

## 설치

최신 릴리스 다운로드:

| 플랫폼 | 파일 |
|---|---|
| macOS Apple Silicon (ARM64) | `WikiLoop-<version>-macos-arm64.dmg` |
| Linux x86_64 | `wikiloop-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `wikiloop-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `wikiloop-<version>-windows-amd64.zip` |

> **macOS Intel (x86_64):** 사전 빌드된 릴리스 없음. GitHub Actions가 2025년 4월에 Intel macOS 러너를 중단했습니다. Intel Mac에서 소스 코드로 빌드하세요: `CGO_ENABLED=1 go build -tags fts5 -o wikiloop ./cmd/wikiloop/`

**macOS:** DMG를 열고 WikiLoop를 Applications로 드래그합니다. 앱은 메뉴바 아이콘으로 실행됩니다.

**Linux:**
```bash
tar -xzf wikiloop-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/wikiloop /usr/local/bin/wikiloop
```

**Windows:** zip을 압축 해제하고 `wikiloop.exe serve`를 실행합니다 (MCP용 `wikiloop.exe stdio`). 디렉토리를 `PATH`에 추가합니다. CGO 불필요 — 순수 Go 바이너리.

**HarmonyOS PC (커뮤니티, 실험적):** WikiLoop는 HarmonyOS PC용으로 공식 출시되지 않습니다. 그러나 핵심 바이너리는 CGO가 필요 없으므로 (순수 Go + SQLite), 커뮤니티 패키지 관리자 [Harmonybrew](https://harmonybrew.dev)를 사용하여 HarmonyOS PC에서 네이티브로 빌드할 수 있습니다. HarmonyOS PC에서 Go + CGO 설정 가이드는 [ohos_go_cgo](https://github.com/ohos-go/ohos_go_cgo)를 참조하세요.

```bash
# HarmonyOS PC에서 (Harmonybrew를 통해 Go 설치 후)
CGO_ENABLED=0 go build -tags fts5 -o wikiloop ./cmd/wikiloop/
wikiloop serve
```

## 소스 코드에서 빌드

Go 1.25+ 필요. CGO 불필요.

```bash
# macOS / Linux
go build -tags fts5 -o wikiloop ./cmd/wikiloop/

# Windows
go build -tags fts5 -o wikiloop.exe ./cmd/wikiloop/
```

또는 멀티 플랫폼 빌드 스크립트 사용:

```bash
./scripts/build.sh [version] [target...]
```

| Target | 출력 | 플랫폼 |
|---|---|---|
| `darwin-arm64` | `dist/WikiLoop-<version>-macos-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/wikiloop-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/wikiloop-<version>-linux-arm64.tar.gz` | Linux ARM64 |
| `windows-amd64` | `dist/wikiloop-<version>-windows-amd64.zip` | Windows x86_64 |

## 저장소 구조

```text
wikiloop/
  cmd/wikiloop/        # 메인 진입점
  internal/
    kb/                # FTS 인덱싱, 검색, 그래프 확장, 페이지 가져오기
    mcp/               # MCP 서버 (stdio + HTTP)
    watcher/           # 자동 정제 + 재인덱싱을 위한 파일 watcher
    distill/           # LLM 정제 파이프라인
    synthesize/        # concept/comparison/decision 페이지 생성
    convert/           # 원본 파일 변환
    service/           # OS 서비스 관리자 (launchd / systemd)
    webui/             # 웹 UI
    tray/              # macOS 시스템 트레이 (darwin 전용)
    config/            # KB 설정 (config.yaml)
  scripts/
    build.sh           # 멀티 플랫폼 빌드 스크립트
```

## 스키마 & 템플릿

`wikiloop init`은 번들된 저작 규칙과 페이지 템플릿으로 KB의 `schema/` 디렉토리를 채웁니다:

- `schema/templates/`: source-note / concept / comparison / decision 페이지를 위한 Markdown 템플릿.
- `schema/references/`: 저작 규칙 — 페이지 유형, 인용 규칙, 충돌 규칙, 디렉토리 구조.

정제/합성 프롬프트는 이 템플릿을 읽으므로, 편집하면 KB별로 생성된 위키 형식을 커스터마이즈합니다.

## 빠른 시작

```bash
export WIKILOOP_KB=/path/to/your-kb

wikiloop init           # KB 디렉토리 스캐폴딩 및 schema/템플릿 복사
wikiloop serve          # 서버 시작: MCP + Web UI + 파일 watcher
wikiloop index          # FTS 인덱스 빌드/업데이트
wikiloop status         # 인덱스 통계
wikiloop lint           # wiki 페이지 상태 점검
```

## 명령어 참조

모든 명령은 전역 `--kb <path>` 플래그를 허용합니다 (기본값은 `$WIKILOOP_KB`, 그 다음 `~/wikiloop-kb`).

| 명령 | 설명 |
|---|---|
| `wikiloop init [--force]` | KB 디렉토리 스캐폴딩 및 번들된 schema/템플릿 복사. |
| `wikiloop serve` | 장시간 실행 서버 시작: HTTP MCP (`/mcp`) + Web UI + 파일 watcher. 서브커맨드 없을 때 기본값. |
| `wikiloop index` | `wiki/`와 `raw/` markdown에서 FTS 인덱스 빌드/업데이트. |
| `wikiloop search <query>` | FTS 키워드 검색; 경로와 스니펫이 있는 순위별 결과 출력. |
| `wikiloop synthesize [--topic X] [--full]` | source-notes에서 concept/comparison/decision 페이지 생성. |
| `wikiloop synthesize --gaps --topic X` | 주제에 대한 지식 격차 분석. |
| `wikiloop import-lark <URL>` | Lark/Feishu Wiki 페이지와 임베디드 테이블을 `raw/lark/`로 가져옵니다. 로그인된 `lark-cli` 필요. |
| `wikiloop lint` | wiki 페이지 상태 점검: 누락된 frontmatter 필드, 깨진 소스 링크. |
| `wikiloop status` | 인덱스 통계 출력 (문서 수, 인덱스 크기). |
| `wikiloop service <install\|uninstall\|start\|stop\|status\|logs>` | OS 서비스 관리 (launchd / systemd). |

**LLM 설정** (KB 루트 아래 `config.yaml`의 `distill` 섹션)은 `distill`과 `synthesize`에 필요합니다.

## MCP 서버

WikiLoop는 MCP 프로토콜을 통해 KB 도구를 노출합니다.

**사용 가능한 도구:** `kb_search`, `kb_page`

관리 작업 (`status`, `reindex`, `lint`)은 Web UI 또는 CLI를 통해 사용 가능합니다 (`wikiloop status`, `wikiloop index`, `wikiloop lint`).

---

### 시나리오 1: 로컬 멀티 에이전트 공유

HTTP 모드 권장: 하나의 WikiLoop 프로세스를 모든 에이전트가 공유 — Claude Code, Cursor, VS Code (Copilot), Windsurf, Trae, Codex, Hermes, OpenClaw 등.

**1단계: WikiLoop 시작**

```bash
export WIKILOOP_KB=/path/to/wikiloop-kb
wikiloop serve
```

> macOS에서: WikiLoop.app을 더블 클릭하여 메뉴바 아이콘으로 실행합니다.

**2단계: 각 에이전트에서 HTTP MCP 설정**

`~/.claude.json`의 `mcpServers`에 추가:

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

`x-api-key`는 `config.yaml`의 `server.api_key`에 해당합니다. api_key가 설정되지 않은 경우 `headers`를 생략합니다.

---

### 시나리오 2: 호스팅된 에이전트 환경

호스팅 환경 (Hermes, OpenClaw 등)에서는 WikiLoop를 영구 볼륨에 설치하고 **stdio**를 통해 호출합니다 — WikiLoop는 에이전트 호스트의 하위 프로세스로 시작하며, watcher는 자동으로 백그라운드에서 실행됩니다.

예시 (NAS 마운트된 OpenClaw/Hermes, 마운트 포인트 `/root/.openclaw`):

**1. 영구 볼륨에 설치 (1회):**

```bash
tar -xzf wikiloop-linux-amd64.tar.gz -C /root/.openclaw/wikiloop/
chmod +x /root/.openclaw/wikiloop/wikiloop
```

**2. markitdown 설치 (권장):**

markitdown을 사용하면 PDF, Word, Excel, PPT, HTML 파일을 Markdown으로 변환한 후 정제할 수 있습니다. 없으면 `.md`와 `.txt` 파일만 정제되고, 바이너리 파일은 파일 이름으로만 인덱싱됩니다.

```bash
pip install markitdown
# 확인
markitdown --version
```

> OpenClaw/Hermes에서 검증됨 (경로: `/root/.openclaw/workspace/bin/markitdown`). `workspace/bin`을 PATH에 추가하거나 환경에서 전체 경로를 설정합니다.

markitdown을 사용할 수 없는 경우, 에이전트는 텍스트를 직접 추출하여 (LLM vision 또는 다른 도구 사용) 결과를 `$WIKILOOP_KB/raw/converted/<slug>.md`에 직접 쓸 수 있습니다 — watcher가 자동으로 처리합니다.

**3. MCP 설정:**

Hermes (에이전트 설정의 `mcp_servers`):

```yaml
mcp_servers:
  wikiloop:
    command: /root/.openclaw/wikiloop/wikiloop
    args: [stdio]
    env:
      WIKILOOP_KB: /root/.openclaw/wikiloop-kb
      PATH: /root/.openclaw/workspace/bin:/usr/local/bin:/usr/bin:/bin
```

KB 디렉토리는 첫 실행 시 자동으로 생성됩니다. 수동 `init` 불필요.

**4. 지식 베이스에 콘텐츠 추가:**

`write_file` 액세스 권한이 있는 에이전트는 KB에 직접 쓸 수 있습니다 — watcher가 변경 사항을 감지하고 자동으로 인덱싱 및 정제를 트리거합니다.

| 콘텐츠 유형 | 쓰기 위치 |
|---|---|
| 기사, 노트, 참조 자료 (Markdown/텍스트) | `$WIKILOOP_KB/raw/<카테고리>/<slug>.md` |
| 에이전트가 변환한 PDF/Word/Excel/EPUB 콘텐츠 | `$WIKILOOP_KB/raw/converted/<slug>.md` |

`raw/converted/`의 파일은 이미 변환된 것으로 처리되어 markitdown 단계를 건너뛰고 바로 정제됩니다. `raw/` 아래의 다른 모든 경로는 전체 파이프라인 (변환 → 인덱싱 → 정제)을 통해 처리됩니다.

`raw/` 아래의 하위 디렉토리는 콘텐츠에 맞게 자유롭게 구성하세요 — WikiLoop는 `raw/` 아래에 고정된 구조를 강요하지 않습니다.

## 시스템 서비스 (선택 사항)

`wikiloop serve`에는 KB 디렉토리를 자동으로 모니터링하고, 정제를 트리거하고, 인덱스를 재구성하는 내장 watcher가 포함됩니다. 추가 설정이 필요하지 않습니다.

WikiLoop가 **부팅 시 시작하고 백그라운드에서 실행**되게 하려면 시스템 서비스로 설치합니다 (macOS launchd / Linux systemd):

```bash
wikiloop service install --kb /path/to/your-kb
wikiloop service status
wikiloop service uninstall
```

로그: `{WIKILOOP_KB}/index/watcher.log`
