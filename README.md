# gmr — Git Merge Request automation

CLI-утиліта, яка автоматизує створення GitLab Merge Request: стейджить зміни, генерує commit message через AI (Gemini / Claude), створює гілку і відкриває MR — однією командою.

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
sudo cp gmr /usr/local/bin/gmr
```

За замовчуванням встановлюється в `/usr/local/bin`. Можна змінити через `GMR_INSTALL_DIR`:

```bash
GMR_INSTALL_DIR=~/.local/bin bash <(curl -fsSL https://raw.githubusercontent.com/slucheninov/gmr/master/install.sh)
```

## Requirements

- `glab` — [GitLab CLI](https://gitlab.com/gitlab-org/cli)
- `jq`
- `curl`
- `GEMINI_API_KEY` та/або `ANTHROPIC_API_KEY`

## Usage

```bash
gmr [branch-name]
```

Якщо `branch-name` не вказано, генерується автоматично: `auto/YYYYMMDD-HHMMSS`.

## How it works

1. Перевіряє, що ти на основній гілці і є зміни
2. Стейджить всі зміни (`git add -A`)
3. Генерує commit message через AI: Gemini (default) → Claude (fallback) → ручне введення
4. Створює гілку, комітить, відкриває MR через `glab`
5. Повертається на основну гілку

## Configuration

| Змінна | Опис | Default |
|---|---|---|
| `GEMINI_API_KEY` | API ключ Google Gemini | — |
| `ANTHROPIC_API_KEY` | API ключ Anthropic Claude | — |
| `GMR_MAIN_BRANCH` | Основна гілка | `master` |
| `GMR_GEMINI_MODEL` | Модель Gemini | `gemini-2.5-flash` |
| `GMR_ANTHROPIC_MODEL` | Модель Claude | `claude-sonnet-4-20250514` |
| `GMR_MAX_DIFF` | Макс. рядків diff для AI | `500` |
| `GMR_INSTALL_DIR` | Каталог встановлення | `/usr/local/bin` |

## License

[MIT](LICENSE)
