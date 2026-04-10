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

## Usage

```bash
gmr [branch-name]
```

Якщо `branch-name` не вказано, генерується автоматично: `auto/YYYYMMDD-HHMMSS`.

## How it works

1. Перевіряє, що ти на основній гілці і є зміни
2. Визначає платформу (GitLab / GitHub) за URL `origin` remote
3. Стейджить всі зміни (`git add -A`)
4. Генерує commit message через AI: Gemini → Claude → OpenAI → ручне введення
5. Створює гілку, комітить, відкриває MR (`glab`) або PR (`gh`)
6. Для GitHub — вмикає auto-merge зі squash
7. Повертається на основну гілку

## Configuration

| Змінна | Опис | Default |
|---|---|---|
| `GEMINI_API_KEY` | API ключ Google Gemini | — |
| `ANTHROPIC_API_KEY` | API ключ Anthropic Claude | — |
| `OPENAI_API_KEY` | API ключ OpenAI | — |
| `GMR_MAIN_BRANCH` | Основна гілка | `master` |
| `GMR_GEMINI_MODEL` | Модель Gemini | `gemini-flash-latest` |
| `GMR_ANTHROPIC_MODEL` | Модель Claude | `claude-sonnet-4-20250514` |
| `GMR_OPENAI_MODEL` | Модель OpenAI | `gpt-4o-mini` |
| `GMR_MAX_DIFF` | Макс. рядків diff для AI | `500` |
| `GMR_INSTALL_DIR` | Symlink directory | `/usr/local/bin` |

## License

[MIT](LICENSE)
