# gmr - Git Merge Request automation

[![CI](https://github.com/slucheninov/gmr/actions/workflows/ci.yml/badge.svg)](https://github.com/slucheninov/gmr/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/slucheninov/gmr?sort=semver)](https://github.com/slucheninov/gmr/releases/latest)
[![License: MIT](https://img.shields.io/github/license/slucheninov/gmr)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/slucheninov/gmr)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/slucheninov/gmr)](https://goreportcard.com/report/github.com/slucheninov/gmr)

CLI-утиліта на Go, яка автоматизує створення Merge Request / Pull Request: стейджить зміни, генерує commit message через AI (Gemini / Claude / OpenAI) у звичайному людському стилі (або Conventional Commits за бажанням), створює гілку і відкриває GitLab MR або GitHub PR - однією командою. Платформа визначається автоматично за URL `origin` remote.

## Installation

### Pre-built binary (рекомендовано)

Завантажити архів для вашої ОС / архітектури з [GitHub Releases](https://github.com/slucheninov/gmr/releases/latest):

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/slucheninov/gmr/releases/latest | jq -r .tag_name)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -L -o gmr.tar.gz \
  "https://github.com/slucheninov/gmr/releases/download/${VERSION}/gmr-${VERSION}-${OS}-${ARCH}.tar.gz"
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

## Update

### Pre-built binary

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/slucheninov/gmr/releases/latest | jq -r .tag_name)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -L -o gmr.tar.gz \
  "https://github.com/slucheninov/gmr/releases/download/${VERSION}/gmr-${VERSION}-${OS}-${ARCH}.tar.gz"
tar -xzf gmr.tar.gz
sudo install -m 0755 gmr /usr/local/bin/gmr
gmr --version
```

### Через `go install`

```bash
go install github.com/slucheninov/gmr/cmd/gmr@latest
```

### З вихідного коду

```bash
cd gmr
git pull
go build -o gmr ./cmd/gmr
sudo install -m 0755 gmr /usr/local/bin/gmr
```

## Requirements

- `glab` - [GitLab CLI](https://gitlab.com/gitlab-org/cli) (для GitLab репо)
- `gh` - [GitHub CLI](https://cli.github.com) (для GitHub репо)
- `git`
- `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, та/або `OPENAI_API_KEY` (хоча б один)
- Авторизований `glab` (`glab auth login`) для GitLab API або авторизований `gh` (`gh auth login`) для GitHub API

> Залежності `jq` і `curl` більше не потрібні - все робиться силами Go-бінарника.

## Usage

```bash
gmr [options] [branch-name]   # full flow: commit + MR/PR
gmr -m                          # generate commit message only
gmr -s                          # after MR/PR, stay on the feature branch
gmr -h                          # help
gmr -v                          # version
gmr deploy [options] [tag]      # cut and publish the next release tag
gmr status [options] [ref]      # show CI/CD pipeline status
```

`deploy` і `status` - зарезервовані слова: якщо вони передані першим аргументом, `gmr` вважає це підкомандою, а не назвою гілки.

Якщо `branch-name` не вказано, назва гілки виводиться з AI commit title (наприклад, `fix-detect`, `feat-add`). При колізії додається числовий суфікс (`fix-detect2`). Fallback: `auto-YYYYMMDD-HHMMSS`.

Якщо `gmr` запущено з уже створеної feature-гілки, яка має коміти відносно основної гілки, він використовує її як source branch і одразу створює MR/PR. Нова гілка та новий коміт не створюються, AI API key не потрібен, а після завершення (або помилки) `gmr` залишається на поточній feature-гілці. Якщо в ній є незакомічені зміни, вони спочатку комітяться у цю ж гілку.

З прапорцем `-m` (`--message`) скрипт лише генерує commit message через AI (виводиться у `stdout`), без створення гілки, коміту чи MR/PR. Працює з будь-якої гілки.

З прапорцем `-s` (`--stay`) після успішного створення MR/PR ти залишаєшся на feature-гілці без жодних питань; без прапорця gmr запитає `Stay on branch '<branch>' or switch to '<main>'? [s/M]:` - `s`/`stay`/`y`/`yes` (без урахування регістру) залишає на гілці, будь-яка інша відповідь або Enter перемикає на основну гілку і робить `git pull`. Якщо stdin не є інтерактивним терміналом, питання пропускається і gmr одразу перемикається на основну гілку.

## How it works

1. Визначає поточну гілку: з основної запускає повний flow зі створенням feature-гілки, а на існуючій feature-гілці використовує вже додані коміти.
2. Визначає платформу (GitLab / GitHub) за URL `origin` remote.
3. Якщо є незакомічені зміни, стейджить їх (`git add -A`).
4. Для нових змін генерує commit message через AI: Gemini → Claude → OpenAI → ручне введення. Для вже закоміченої feature-гілки використовує повідомлення останнього коміту.
5. З основної гілки створює нову feature-гілку й коміт; з існуючої feature-гілки використовує її напряму. Потім відкриває MR (`glab`) або PR (`gh`).
6. Для GitLab передає в `glab` явні `title` і `description`: використовує body commit message, а якщо його немає - генерує короткий `## Summary` із заголовка коміту.
7. Для GitHub - вмикає auto-merge зі squash (gracefully degrade, якщо репо це забороняє).
8. Для нового flow запитує, залишитись на feature-гілці чи перемкнутись на основну (з `-s` / `--stay` питання пропускається і gmr одразу залишається на гілці); за замовчуванням (Enter або інша відповідь) перемикається на основну гілку і виконує `git pull`. При запуску з існуючої feature-гілки завжди залишається на ній без питань.

## `gmr deploy`

Створює наступний semver-тег із комітів з моменту попереднього тега, генерує release notes і semver bump через AI, пушить тег і створює GitHub Release / GitLab Release.

```bash
gmr deploy              # AI обирає bump (patch/minor/major) і пише release notes
gmr deploy --minor      # форсувати minor bump замість вибору AI
gmr deploy v1.4.0       # явний тег - перекриває будь-який bump
gmr deploy --no-release # створити і запушити тег, але не створювати Release
gmr deploy -y           # без підтвердження
```

Правила визначення тега:
- Якщо в репозиторії ще немає semver-тегів - перший реліз `v0.0.1` (префікс з `GMR_TAG_PREFIX`, за замовчуванням `v`).
- Інакше - найновіший тег, збільшений на рівень bump (`--patch`/`--minor`/`--major` або вибір AI), з тим самим префіксом.
- Явний позиційний тег (наприклад `v1.2.3`) перекриває і прапорець, і вибір AI; має відповідати `<prefix>MAJOR.MINOR.PATCH`.

Якщо всі AI-провайдери недоступні, gmr попереджає і використовує `patch` bump та сирий git log як release notes - реліз все одно можна створити, але notes варто переглянути. `gmr deploy` вимагає чистого робочого дерева і запускається з базової гілки (з іншої - попереджає і запитує підтвердження, якщо є TTY).

## `gmr status`

Показує статус останніх CI/CD запусків (GitHub Actions / GitLab Pipelines) для поточної гілки та останнього тега (або явно вказаного ref).

```bash
gmr status              # поточна гілка + останній тег
gmr status my-branch    # конкретна гілка/тег
gmr status --limit 5    # показати 5 останніх запусків на ref (1-20, за замовчуванням 3)
```

Для кожного ref виводить список запусків (✓/✗/●/○/–) з job'ами найновішого запуску та підсумковий рядок (`all pipelines passed` / `FAILED (<jobs>)` / `still running` / `no pipelines found`). Завершується кодом `1`, якщо найновіший запуск будь-якого перевіреного ref провалився - зручно для скриптів (`gmr status || echo "deploy failed"`).

## Configuration

| Змінна | Опис | Default |
|---|---|---|
| `GEMINI_API_KEY` | API ключ Google Gemini | - |
| `ANTHROPIC_API_KEY` | API ключ Anthropic Claude | - |
| `OPENAI_API_KEY` | API ключ OpenAI | - |
| `GEMINI_MODEL` | Модель Gemini | `gemini-flash-latest` |
| `ANTHROPIC_MODEL` | Модель Claude | `claude-sonnet-4-20250514` |
| `OPENAI_MODEL` | Модель OpenAI | `gpt-4o-mini` |
| `GEMINI_BASE_URL` | Base URL для Gemini API override | `https://generativelanguage.googleapis.com/v1beta` |
| `ANTHROPIC_BASE_URL` | Base URL для Claude API override | `https://api.anthropic.com` |
| `OPENAI_BASE_URL` | Base URL для OpenAI-compatible API override (наприклад LiteLLM) | `https://api.openai.com` |
| `GMR_PROVIDERS` | Порядок AI-провайдерів (comma-separated) | `gemini,claude,openai` |
| `GMR_COMMIT_STYLE` | Стиль commit message: `human` (звичайне речення) або `conventional` (`type: description`) | `human` |
| `GMR_MAIN_BRANCH` | Основна гілка | auto (`origin/HEAD`, fallback: `main`/`master`) |
| `GMR_MAX_DIFF` | Макс. рядків diff/log для AI | `500` |
| `GMR_TAG_PREFIX` | Префікс тега для `gmr deploy`, коли тегів ще немає (`""` - без префікса) | `v` |
| `EDITOR` | Редактор для режиму `e(edit)` | `vim` |
| `NO_COLOR` | Вимкнути ANSI кольори у виводі | - |

## Development

Гайд з локальної розробки, тестів, лінту і релізного процесу — у
[DEVELOPMENT.md](DEVELOPMENT.md). Контрибʼюторам також варто прочитати
[CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
