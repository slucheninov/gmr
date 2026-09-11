# Development

Цей документ описує, як локально працювати з кодом `gmr`. Для воркфлоу
контрибʼюторів, гайдлайнів стилю і чеклісту PR — див.
[CONTRIBUTING.md](CONTRIBUTING.md).

## Prerequisites

- Go **1.25+**
- `git`
- Опціонально: [`golangci-lint`](https://golangci-lint.run/) v2. Використовуй
  збірку, сумісну з локальною версією Go.
- Опціонально, для end-to-end перевірки: авторизований `gh` та/або `glab`,
  плюс ключ одного з AI-провайдерів, якщо потрібна AI-генерація

## Project layout

```text
cmd/gmr/main.go             CLI entry point, argument parsing і MR/PR flow
cmd/gmr/deploy.go           orchestration для `gmr deploy`
cmd/gmr/status.go           orchestration і rendering для `gmr status`
internal/ai/                Provider interface + Gemini / Claude / OpenAI
internal/ci/                адаптери GitHub Actions / GitLab Pipelines
internal/commit/            commit title/body, branch name і MR description
internal/git/               git wrapper із тестованим Runner interface
internal/platform/          host detection + парсинг GitLab project path
internal/release/           semver, наступний тег і парсинг AI release response
internal/ui/                stderr-логування + ANSI кольори; поважає NO_COLOR
internal/version/           Version; override через -ldflags
```

## Build

```bash
go build -o gmr ./cmd/gmr
./gmr --version
```

З вшитою версією (як у CI/release):

```bash
RELEASE_TAG=v1.2.3
go build -trimpath \
  -ldflags "-s -w -X github.com/slucheninov/gmr/internal/version.Version=${RELEASE_TAG}" \
  -o gmr ./cmd/gmr
```

## Tests

```bash
go test ./...
go test -race ./...
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Що вкрите тестами

- Парсинг URL `origin` для визначення платформи (`internal/platform`).
- Витяг `group/project` з GitLab remote.
- Хелпери для генерації MR title / description з commit message
  (`internal/commit`).
- Логіка резолвінгу основної гілки
  (`GMR_MAIN_BRANCH` → `origin/HEAD` → `main` / `master`).
- Утиліта обмеження кількості рядків для diff (`LimitLines`).
- AI-провайдери (Gemini / Claude / OpenAI) — через `httptest`-сервери: успіх,
  обробка помилок API, обрізання відповіді при `MAX_TOKENS` / `length` /
  `max_tokens`.
- Semver і release response parsing (`internal/release`).
- Нормалізація GitHub Actions / GitLab pipeline states і JSON-відповідей
  зовнішніх CLI (`internal/ci`).
- Парсинг аргументів та orchestration для основної команди, `deploy` і `status`
  (`cmd/gmr/*_test.go`).

### Гайдлайни тестування

- Pure-логіка → стандартні `*_test.go` поряд із файлом, без I/O.
- AI-провайдери → `httptest.NewServer` + override `ai.HTTPClient`.
- Код, що дзвонить `git`, → інтерфейс `git.Runner` + fake-implementation
  (див. `internal/git/git_test.go`).
- Запити CI через `gh` / `glab` → інтерфейс `ci.Runner` + fake-implementation
  (див. `internal/ci/ci_test.go`).
- Default unit suite не повинен залежати від мережі, справжньої авторизації або
  стану зовнішнього репозиторію.

## Lint

```bash
go vet ./...
test -z "$(gofmt -l .)"
golangci-lint run
```

Конфіг: [`.golangci.yml`](.golangci.yml). CI також запускає race-тести з
coverage і build smoke test на Go 1.25.

## Run locally

Для повного flow `gmr` потрібен справжній git-репозиторій з `origin` remote та
авторизований `gh` / `glab`. Використовуй одноразовий fork або тимчасовий
репозиторій: команда може створювати гілки, коміти, теги, пуші та MR/PR.

Режим `-m` не створює коміт або MR/PR, але все одно виконує `git add -A`, тому
враховуй зміну index:

```bash
export GEMINI_API_KEY=...   # або ANTHROPIC_API_KEY / OPENAI_API_KEY
go run ./cmd/gmr -m         # commit message у stdout, усі зміни staged
```

Для перевірки лише парсингу аргументів використовуй unit-тести, а не реальні
API або remote-операції.

## Releasing

Релізи створюються автоматично через
[`.github/workflows/release.yml`](.github/workflows/release.yml):

- Workflow тригериться при push тегу `v*` (також доступний ручний запуск
  `workflow_dispatch`).
- Запускає тести, кросс-компілює бінарники для
  `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/{amd64,arm64}`.
- Пакує кожен бінарник в `gmr-<TAG>-<os>-<arch>.tar.gz` (для Windows — `.zip`)
  разом з `LICENSE`, `README.md`, `CHANGELOG.md`.
- Генерує `checksums.txt` з SHA-256 і прикріпляє все до GitHub Release.

Щоб випустити нову версію в цьому репозиторії:

1. Бампнути `Version` у `internal/version/version.go`.
2. Оновити `CHANGELOG.md` під новою секцією `[X.Y.Z] - YYYY-MM-DD`.
3. Запустити повний набір перевірок і закомітити зміни.
4. На чистій базовій гілці створити та запушити тег без окремого GitHub Release:

   ```bash
   git commit -am "chore: release v1.2.3"
   git push
   gmr deploy --no-release v1.2.3
   ```

   `--no-release` обов'язковий для стандартного flow цього репозиторію, бо
   GitHub Actions сам створює Release з кросплатформними артефактами.

Альтернатива без `gmr deploy`: створити annotated tag вручну й запушити його в
`origin`. Не створюй lightweight tag, якщо релізний процес очікує release notes.

Вручну запушені теги з дефісом (наприклад, `v1.2.3-rc.1`) workflow позначає як
prerelease. `gmr deploy` приймає лише точний формат
`<prefix>MAJOR.MINOR.PATCH` без prerelease/build suffix.
