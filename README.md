# gmr — Git Merge Request automation

CLI-утиліта, яка автоматизує створення Merge Request / Pull Request: стейджить зміни, генерує commit message через AI (Gemini / Claude / OpenAI), створює гілку і відкриває GitLab MR або GitHub PR — однією командою. Платформа визначається автоматично за URL `origin` remote.

## Installation

**curl:**

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/slucheninov/gmr/master/install.sh)
```

**wget:**

```bash
bash <(wget -qO- https://raw.githubusercontent.com/slucheninov/gmr/master/install.sh)
```

**git clone:**

```bash
git clone https://github.com/slucheninov/gmr.git
cd gmr
./install.sh
```

The script is installed to `~/.gmr/bin/gmr` with a symlink in `/usr/local/bin`. To change the symlink directory:

```bash
GMR_INSTALL_DIR=~/.local/bin bash <(curl -fsSL https://raw.githubusercontent.com/slucheninov/gmr/master/install.sh)
```

## Update

Force reinstall to get the latest version:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/slucheninov/gmr/master/install.sh) -f
```

## Requirements

- `glab` — [GitLab CLI](https://gitlab.com/gitlab-org/cli) (for GitLab repos)
- `gh` — [GitHub CLI](https://cli.github.com) (for GitHub repos)
- `jq`
- `curl`
- `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, та/або `OPENAI_API_KEY` (хоча б один)
- Авторизований `glab` (`glab auth login`) для GitLab API або авторизований `gh` (`gh auth login`) для GitHub API

## Usage

```bash
gmr [branch-name]    # full flow: commit + MR/PR
gmr -m               # generate commit message only
```

Якщо `branch-name` не вказано, генерується автоматично: `auto/YYYYMMDD-HHMMSS`.

З прапорцем `-m` (`--message`) скрипт лише генерує commit message через AI, без створення гілки, коміту чи MR/PR. Працює з будь-якої гілки.

## How it works

1. Перевіряє, що ти на основній гілці і є зміни (у режимі `-m` — лише зміни)
2. Визначає платформу (GitLab / GitHub) за URL `origin` remote
3. Стейджить всі зміни (`git add -A`)
4. Генерує commit message через AI: Gemini → Claude → OpenAI → ручне введення
5. Створює гілку, комітить, відкриває MR (`glab`) або PR (`gh`)
6. Для GitLab передає в `glab` явні `title` і `description` для MR без конфліктного `--fill`: використовує body commit message, а якщо його немає — генерує короткий опис із заголовка коміту
7. Для GitHub — вмикає auto-merge зі squash
8. Повертається на основну гілку

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
| `GMR_INSTALL_BRANCH` | Preferred branch for installer download | `master` (fallback: `main`) |
| `GMR_INSTALL_DIR` | Symlink directory | `/usr/local/bin` |

## Releases

Релізи створюються автоматично через GitHub Actions (`.github/workflows/release.yml`):

- Workflow тригериться при push у `master`/`main`, якщо змінився файл `gmr` (також доступний ручний запуск `workflow_dispatch`).
- Зчитує `GMR_VERSION` зі скрипта `gmr` і, якщо тегу `vX.Y.Z` ще немає, створює його.
- Витягує release notes для цієї версії з `CHANGELOG.md` (фолбек — секція `[Unreleased]`).
- Збирає та прикріпляє до релізу такі ассети:
  - `gmr` — самодостатній скрипт
  - `install.sh` — інсталятор
  - `gmr-X.Y.Z.tar.gz` і `gmr-X.Y.Z.zip` — архіви з `gmr`, `install.sh`, `README.md`, `LICENSE`, `CHANGELOG.md`
  - `gmr-X.Y.Z.sha256` — SHA-256 контрольні суми для архівів

Щоб випустити нову версію — у одному коміті бампнути `GMR_VERSION` у `gmr` та оновити `CHANGELOG.md`, далі змерджити в `master`.

### Завантаження релізу вручну

Останній реліз:

```bash
# окремий скрипт
curl -fsSLO https://github.com/slucheninov/gmr/releases/latest/download/gmr

# архів tar.gz
curl -fsSLO https://github.com/slucheninov/gmr/releases/latest/download/gmr-$(curl -fsSL https://api.github.com/repos/slucheninov/gmr/releases/latest | jq -r .tag_name | sed 's/^v//').tar.gz
```

Конкретна версія (теги в форматі `vX.Y.Z`):

```bash
curl -fsSLO https://github.com/slucheninov/gmr/releases/download/v0.5.0/gmr-0.5.0.tar.gz
tar -xzf gmr-0.5.0.tar.gz
cd gmr-0.5.0 && ./install.sh
```

Перевірка контрольної суми:

```bash
curl -fsSLO https://github.com/slucheninov/gmr/releases/download/v0.5.0/gmr-0.5.0.sha256
sha256sum -c gmr-0.5.0.sha256
```

### Як працює `install.sh`

За замовчуванням інсталятор тягне `gmr` із **останнього GitHub Release** (`releases/latest/download/gmr`). Якщо реліз недоступний — фолбек на raw-файл із гілки `master` (далі `main`).

Поведінкою керують env vars:

| Змінна | Опис | Default |
|---|---|---|
| `GMR_INSTALL_FROM` | Джерело: `release` або `branch` | `release` |
| `GMR_INSTALL_VERSION` | Тег релізу (`latest` або `vX.Y.Z`) | `latest` |
| `GMR_INSTALL_BRANCH` | Гілка для branch-mode / фолбеку | `master` (фолбек: `main`) |
| `GMR_INSTALL_DIR` | Куди класти symlink | `/usr/local/bin` |

Приклади:

```bash
# Конкретна версія з релізу
GMR_INSTALL_VERSION=v0.5.0 bash <(curl -fsSL https://raw.githubusercontent.com/slucheninov/gmr/master/install.sh)

# Поточний код із master (без релізу)
GMR_INSTALL_FROM=branch bash <(curl -fsSL https://raw.githubusercontent.com/slucheninov/gmr/master/install.sh)
```

Що відбувається крок за кроком:

1. Парсяться аргументи (`-f` / `--force`) та env vars.
2. Якщо `~/.gmr/bin/gmr` уже існує і немає `--force` — вихід.
3. Завантажується файл `gmr` з обраного джерела (`curl` або `wget`).
4. Кладеться у `~/.gmr/bin/gmr` із прапорцем виконання.
5. Створюється symlink `${GMR_INSTALL_DIR}/gmr → ~/.gmr/bin/gmr` (через `sudo`, якщо немає прав запису).

## License

[MIT](LICENSE)
