# gmr — Git Merge Request automation

CLI-утиліта на Go, яка автоматизує створення Merge Request / Pull Request: стейджить зміни, генерує commit message через AI (Gemini / Claude / OpenAI), створює гілку і відкриває GitLab MR або GitHub PR — однією командою. Платформа визначається автоматично за URL `origin` remote.

## Installation

### Pre-built binary (рекомендовано)

Завантажити архів для вашої ОС / архітектури з [GitHub Releases](https://github.com/slucheninov/gmr/releases/latest):

```bash
# linux-amd64 (заміни на linux-arm64 / darwin-amd64 / darwin-arm64 за потреби)
VERSION=$(curl -fsSL https://api.github.com/repos/slucheninov/gmr/releases/latest | jq -r .tag_name)
curl -L -o gmr.tar.gz \
  "https://github.com/slucheninov/gmr/releases/download/${VERSION}/gmr-${VERSION}-linux-amd64.tar.gz"
tar -xzf gmr.tar.gz
sudo install -m 0755 gmr /usr/local/bin/gmr
gmr --version
```

Контрольні суми (`checksums.txt`) додаються до кожного релізу:

```bash
curl -L -O "https://github.com/slucheninov/gmr/releases/download/${VERSION}/checksums.txt"
sha256sum -c checksums.txt --ignore-missing
```

### Через `go install`

```bash
go install github.com/slucheninov/gmr/cmd/gmr@latest
```

Бінарник буде у `$(go env GOBIN)` (за замовчуванням `~/go/bin`). Переконайся, що ця тека є в `PATH`.

### З вихідного коду

```bash
git clone https://github.com/slucheninov/gmr.git
cd gmr
go build -o gmr ./cmd/gmr
sudo install -m 0755 gmr /usr/local/bin/gmr
```

## Requirements

- `glab` — [GitLab CLI](https://gitlab.com/gitlab-org/cli) (для GitLab репо)
- `gh` — [GitHub CLI](https://cli.github.com) (для GitHub репо)
- `git`
- `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, та/або `OPENAI_API_KEY` (хоча б один)
- Авторизований `glab` (`glab auth login`) для GitLab API або авторизований `gh` (`gh auth login`) для GitHub API

> Залежності `jq` і `curl` більше не потрібні — все робиться силами Go-бінарника.

## Usage

```bash
gmr [branch-name]    # full flow: commit + MR/PR
gmr -m               # generate commit message only
gmr -h               # help
gmr -v               # version
```

Якщо `branch-name` не вказано, генерується автоматично: `auto/YYYYMMDD-HHMMSS`.

З прапорцем `-m` (`--message`) скрипт лише генерує commit message через AI (виводиться у `stdout`), без створення гілки, коміту чи MR/PR. Працює з будь-якої гілки.

## How it works

1. Перевіряє, що ти на основній гілці і є зміни (у режимі `-m` — лише зміни).
2. Визначає платформу (GitLab / GitHub) за URL `origin` remote.
3. Стейджить всі зміни (`git add -A`).
4. Генерує commit message через AI: Gemini → Claude → OpenAI → ручне введення.
5. Створює гілку, комітить, відкриває MR (`glab`) або PR (`gh`).
6. Для GitLab передає в `glab` явні `title` і `description`: використовує body commit message, а якщо його немає — генерує короткий `## Summary` із заголовка коміту.
7. Для GitHub — вмикає auto-merge зі squash (gracefully degrade, якщо репо це забороняє).
8. Повертається на основну гілку, виконує `git pull`.

## Configuration

| Змінна | Опис | Default |
|---|---|---|
| `GEMINI_API_KEY` | API ключ Google Gemini | — |
| `ANTHROPIC_API_KEY` | API ключ Anthropic Claude | — |
| `OPENAI_API_KEY` | API ключ OpenAI | — |
| `GMR_MAIN_BRANCH` | Основна гілка | auto (`origin/HEAD`, fallback: `main`/`master`) |
| `GMR_GEMINI_MODEL` | Модель Gemini | `gemini-flash-latest` |
| `GMR_ANTHROPIC_MODEL` | Модель Claude | `claude-sonnet-4-20250514` |
| `GMR_OPENAI_MODEL` | Модель OpenAI | `gpt-4o-mini` |
| `GMR_MAX_DIFF` | Макс. рядків diff для AI | `500` |
| `EDITOR` | Редактор для режиму `e(edit)` | `vim` |
| `NO_COLOR` | Вимкнути ANSI кольори у виводі | — |

## Development

### Build

```bash
go build ./cmd/gmr
```

### Tests

```bash
go test ./...
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Тести вкривають:

- Парсинг URL `origin` для визначення платформи (`internal/platform`).
- Витяг `group/project` з GitLab remote.
- Допоміжні функції для генерації MR title / description з commit message (`internal/commit`).
- Логіку резолвінгу основної гілки (`GMR_MAIN_BRANCH` → `origin/HEAD` → `main`/`master`).
- Утиліту обмеження кількості рядків для diff.
- AI-провайдери (Gemini / Claude / OpenAI) — через `httptest`-сервери: успіх, обробка помилок API, обрізання відповіді при `MAX_TOKENS` / `length` / `max_tokens`.

### Lint

```bash
go vet ./...
# опціонально:
golangci-lint run
```

## Releases

Релізи створюються автоматично через GitHub Actions (`.github/workflows/release.yml`):

- Workflow тригериться при push тегу `v*` (також доступний ручний запуск `workflow_dispatch`).
- Запускає тести, кросс-компілює бінарники для `linux/{amd64,arm64}` і `darwin/{amd64,arm64}`.
- Пакує кожен бінарник у `gmr-<TAG>-<os>-<arch>.tar.gz` разом з `LICENSE`, `README.md`, `CHANGELOG.md`.
- Генерує `checksums.txt` з SHA-256 і прикріпляє все до GitHub Release.

Щоб випустити нову версію:

1. Бампнути `Version` у `internal/version/version.go`.
2. Оновити `CHANGELOG.md`.
3. Закомітити, поставити тег `vX.Y.Z`, запушити:

```bash
git tag v0.6.0
git push origin v0.6.0
```

## License

[MIT](LICENSE)
