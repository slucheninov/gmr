# gmr - Git Merge Request automation

[![CI](https://github.com/slucheninov/gmr/actions/workflows/ci.yml/badge.svg)](https://github.com/slucheninov/gmr/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/slucheninov/gmr?sort=semver)](https://github.com/slucheninov/gmr/releases/latest)
[![License: MIT](https://img.shields.io/github/license/slucheninov/gmr)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/slucheninov/gmr)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/slucheninov/gmr)](https://goreportcard.com/report/github.com/slucheninov/gmr)

CLI-утиліта на Go, яка автоматизує створення Merge Request / Pull Request:
стейджить зміни, генерує commit message через AI (Gemini / Claude / OpenAI),
створює або повторно використовує feature-гілку та відкриває GitLab MR або
GitHub PR. Окремі підкоманди створюють релізи (`gmr deploy`) і показують стан
CI/CD (`gmr status`). Платформа визначається автоматично за URL `origin` remote.

## Installation

### Pre-built binary для Linux/macOS (рекомендовано)

Завантажити архів для вашої ОС / архітектури з [GitHub Releases](https://github.com/slucheninov/gmr/releases/latest):

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/slucheninov/gmr/releases/latest | jq -r .tag_name)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
ARCHIVE="gmr-${VERSION}-${OS}-${ARCH}.tar.gz"
curl -LO "https://github.com/slucheninov/gmr/releases/download/${VERSION}/${ARCHIVE}"
tar -xzf "${ARCHIVE}"
sudo install -m 0755 gmr /usr/local/bin/gmr
gmr --version
```

Контрольні суми (`checksums.txt`) додаються до кожного релізу:

```bash
curl -L -O "https://github.com/slucheninov/gmr/releases/download/${VERSION}/checksums.txt"
CHECKSUM=$(awk -v archive="${ARCHIVE}" '$2 == archive { print $1 }' checksums.txt)
test -n "${CHECKSUM}"
printf '%s  %s\n' "${CHECKSUM}" "${ARCHIVE}" | shasum -a 256 -c -
```

Цей install-приклад використовує `curl`, `jq`, `awk` і `shasum`; вони потрібні
лише для завантаження та перевірки архіву, але не для роботи `gmr`.

### Pre-built binary для Windows

Завантаж `gmr-<VERSION>-windows-amd64.zip` або
`gmr-<VERSION>-windows-arm64.zip` зі сторінки
[GitHub Releases](https://github.com/slucheninov/gmr/releases/latest), звір
SHA-256 із `checksums.txt`, розпакуй `gmr.exe` і додай його теку до `PATH`.

### Через `go install`

```bash
go install github.com/slucheninov/gmr/cmd/gmr@latest
```

Якщо `go env GOBIN` порожній, бінарник буде у `$(go env GOPATH)/bin`;
інакше — у явно налаштованому `GOBIN`. Переконайся, що ця тека є в `PATH`.

### З вихідного коду

```bash
git clone https://github.com/slucheninov/gmr.git
cd gmr
go build -o gmr ./cmd/gmr
sudo install -m 0755 gmr /usr/local/bin/gmr
```

## Update

### Автоматичне оновлення

```bash
gmr -u  # або gmr --update
```

Завантажує останній стабільний реліз з GitHub для поточної ОС
(Linux/macOS/Windows) та архітектури (amd64/arm64), перевіряє SHA-256 архіву
за `checksums.txt` і замінює бінарник, з якого запущено команду. Працює з
будь-якої теки, без Git-репозиторію, Go, `gh`/`glab` чи AI-ключів; потрібен
доступ до GitHub. Додаткового підтвердження немає. Символічні посилання
зберігаються — оновлюється їхня ціль.

Якщо поточна версія така сама або новіша, команда нічого не змінює.
Збірки з нерозпізнаною версією (наприклад, `dev-<sha>`) замінюються останнім
стабільним релізом. Помилки завантаження або перевірки залишають старий
бінарник на місці. На Windows попередній бінарник зберігається як
`gmr.exe.old` до наступного оновлення.

Потрібні права запису до теки встановлення. Для системної інсталяції на
Linux/macOS можна запустити `sudo /usr/local/bin/gmr -u` (вкажи фактичний
шлях до свого бінарника). Прапорець `-u` не поєднується з `-c`, `-m`, `-s`
або назвою гілки. Способи ручного оновлення наведено нижче.

### Pre-built binary для Linux/macOS

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/slucheninov/gmr/releases/latest | jq -r .tag_name)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
ARCHIVE="gmr-${VERSION}-${OS}-${ARCH}.tar.gz"
curl -LO "https://github.com/slucheninov/gmr/releases/download/${VERSION}/${ARCHIVE}"
tar -xzf "${ARCHIVE}"
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

## Runtime requirements

- `git`.
- [GitLab CLI](https://gitlab.com/gitlab-org/cli) `glab` для GitLab або
  [GitHub CLI](https://cli.github.com) `gh` для GitHub; відповідний CLI має
  бути авторизований через `glab auth login` або `gh auth login`.
- `GEMINI_API_KEY`, `ANTHROPIC_API_KEY` та/або `OPENAI_API_KEY`, коли потрібно
  згенерувати commit message. Уже закомічена feature-гілка та `gmr status` не
  потребують AI-ключа; `gmr deploy` може використати fallback без AI.
- `EDITOR` потрібен лише для інтерактивної команди `e(edit)`.

## Usage

```bash
gmr [options] [branch-name]   # full flow: commit + MR/PR
gmr -m                          # generate commit message, then ask to commit it
gmr -c                          # commit and push to the current branch, no MR/PR
gmr -s                          # after MR/PR, stay on the feature branch
gmr -u                          # update gmr to the latest stable release
gmr -h                          # help
gmr -v                          # version
gmr deploy [options] [tag]      # cut and publish the next release tag
gmr status [options] [ref]      # show CI/CD pipeline status
```

`deploy` і `status` - зарезервовані слова: якщо вони передані першим аргументом, `gmr` вважає це підкомандою, а не назвою гілки.

Якщо `branch-name` не вказано, назва гілки виводиться з AI commit title (наприклад, `fix-detect`, `feat-add`). При колізії додається числовий суфікс (`fix-detect2`). Fallback: `auto-YYYYMMDD-HHMMSS`.

Якщо `gmr` запущено з уже створеної feature-гілки, яка має коміти відносно основної гілки, він використовує її як source branch і одразу створює MR/PR. Нова гілка та новий коміт не створюються, AI API key не потрібен, а після завершення (або помилки) `gmr` залишається на поточній feature-гілці. Якщо в ній є незакомічені зміни, вони спочатку комітяться у цю ж гілку.

Якщо `gmr` запущено з основної гілки без незакомічених змін, але в ній є коміти, ще не запушені в `origin` (наприклад, після `gmr -m`, яка закомітила прямо в основну гілку), `gmr` не завершується помилкою: він переносить ці коміти в нову гілку, відкриває MR/PR, а після успіху скидає локальну основну гілку на `origin/<main>`. Якщо при цьому є ще й незакомічені зміни, новий коміт і вже наявні незапушені коміти потрапляють в один MR/PR разом, і основна гілка так само скидається на `origin/<main>` після успіху.

З прапорцем `-m` (`--message`) утиліта стейджить усі зміни через `git add -A`,
генерує commit message через AI і завжди виводить його у `stdout` (тому
`gmr -m | ...` можна пайпити). Далі запитує
`Commit to '<current-branch>'? [Y/n/e(edit)] (n = print only):` -
`y`/`yes`/Enter комітить повідомлення в поточну гілку (без нової гілки, push
чи MR/PR) і показує блок "Next steps" з готовими командами `git push` /
`gh pr create` / `glab mr create` / `gmr`; `n`/`no` (і будь-яка інша
відповідь - безпечний варіант за замовчуванням) лише друкує повідомлення,
зміни лишаються застейдженими (`git reset` для розстейджування); `e`/`edit`
відкриває `$EDITOR` перед комітом. Якщо stdin не інтерактивний термінал,
питання не задається і gmr поводиться як при відповіді `n`; так само
трактується EOF (Ctrl+D або stdin з `/dev/null`). Працює з
будь-якої гілки.

З прапорцем `-c` (`--commit`) утиліта комітить і пушить у поточну гілку без
створення нової гілки та MR/PR (`gh`/`glab` не потрібні). Як і `-m`, вона
стейджить усі зміни, генерує commit message, друкує його у `stdout` і запитує
`Commit and push to '<current-branch>'? [Y/n/e(edit)] (n = print only):` -
`y`/`yes`/Enter комітить і виконує `git push -u origin <branch>`; `e`/`edit`
відкриває `$EDITOR` перед комітом; `n`/`no`, будь-яка інша відповідь або
неінтерактивний stdin лише друкують повідомлення. Якщо незакомічених змін
немає, AI не викликається: `gmr -c` просто пушить коміти, яких ще немає в
`origin` (або саму гілку, якщо її там ще немає), а якщо пушити нічого -
завершується помилкою. На основній гілці виводиться попередження, що push
піде в неї напряму. Не поєднується з `-m`, `-s` і `branch-name`.

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
| `GMR_PROVIDERS` | Порядок AI-провайдерів; comma-separated, `anthropic` = `claude` | `gemini,claude,openai` |
| `GMR_COMMIT_STYLE` | Стиль commit message: `human` (звичайне речення) або `conventional` (`type: description`) | `human` |
| `GMR_MAIN_BRANCH` | Основна гілка | auto (`origin/HEAD`, fallback: `main`/`master`) |
| `GMR_MAX_DIFF` | Макс. рядків diff/log для AI | `500` |
| `GMR_TAG_PREFIX` | Префікс тега для `gmr deploy`, коли тегів ще немає (`""` - без префікса) | `v` |
| `EDITOR` | Редактор для режиму `e(edit)` | `vim` |
| `NO_COLOR` | Вимкнути ANSI кольори у виводі | - |

### Конфіденційність

Під час генерації commit message `gmr` виконує `git add -A` і надсилає
провайдеру diff (до `GMR_MAX_DIFF` рядків) разом зі статистикою змін. Під час
`gmr deploy` провайдер отримує обмежений git log. `GMR_MAX_DIFF` — лише ліміт,
а не спосіб повністю вимкнути передачу даних. Не запускай AI-функції для змін,
які не можна передавати обраному провайдеру; зроби коміт вручну. Детальніше —
у [SECURITY.md](SECURITY.md).

## Development

Гайд з локальної розробки, тестів, лінту і релізного процесу — у
[DEVELOPMENT.md](DEVELOPMENT.md). Контрибʼюторам також варто прочитати
[CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
