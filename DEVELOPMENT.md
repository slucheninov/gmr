# Development

Цей документ описує, як локально працювати з кодом `gmr`. Для воркфлоу
контрибʼюторів, гайдлайнів стилю і чеклісту PR — див.
[CONTRIBUTING.md](CONTRIBUTING.md).

## Prerequisites

- Go **1.25+**
- `git`
- Опціонально: [`golangci-lint`](https://golangci-lint.run/) v2 (така ж версія, як у CI)
- Опціонально, для end-to-end перевірки: `gh` та/або `glab`, плюс ключ
  одного з AI-провайдерів

## Project layout

```text
cmd/gmr/main.go             CLI entry point + orchestration
internal/ai/                Provider interface + Gemini / Claude / OpenAI
internal/git/               git wrapper з тестованим Runner interface
internal/platform/          host detection (github.com / gitlab.com) + парсинг GitLab path
internal/commit/            хелпери для commit message (Title / Body / MRDescription)
internal/ui/                логування + ANSI кольори (поважає NO_COLOR), завжди в stderr
internal/version/           Version constant (override через -ldflags)
```

## Build

```bash
go build ./cmd/gmr
./gmr --version
```

З вшитою версією (як у CI/release):

```bash
go build -trimpath \
  -ldflags "-s -w -X github.com/slucheninov/gmr/internal/version.Version=v0.6.0" \
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

### Гайдлайни тестування

- Pure-логіка → стандартні `*_test.go` поряд із файлом, без I/O.
- AI-провайдери → `httptest.NewServer` + override `ai.HTTPClient`.
- Код, що дзвонить `git`, → інтерфейс `git.Runner` + fake-implementation
  (див. `internal/git/git_test.go`).

## Lint

```bash
go vet ./...
golangci-lint run
```

Конфіг: [`.golangci.yml`](.golangci.yml). У CI запускаються ті самі
команди — green локально → green в CI.

## Run locally

`gmr` потребує справжній git-репозиторій з `origin` remote і авторизований
`gh` / `glab`. Найпростіше — створити одноразовий тестовий fork і працювати
там:

```bash
export GEMINI_API_KEY=...   # або ANTHROPIC_API_KEY / OPENAI_API_KEY
go run ./cmd/gmr -m         # safe: лише генерує commit message у stdout
```

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

Щоб випустити нову версію:

1. Бампнути `Version` у `internal/version/version.go`.
2. Оновити `CHANGELOG.md` під новою секцією `[X.Y.Z] - YYYY-MM-DD`.
3. Закомітити, поставити тег `vX.Y.Z`, запушити:

   ```bash
   git commit -am "chore: release v0.6.0"
   git push
   git tag v0.6.0
   git push origin v0.6.0
   ```

Теги з дефісом (наприклад, `v0.6.0-rc.1`) автоматично позначаються як
prerelease.
