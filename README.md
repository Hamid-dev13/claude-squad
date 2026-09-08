# Claude Squad

> **Personal fork** of [smtg-ai/claude-squad](https://github.com/smtg-ai/claude-squad), based on upstream `v1.0.20`.
> It adds a configurable color theme (see [Themes](#themes)) — upstream hardcodes every color, and
> [issue #300](https://github.com/smtg-ai/claude-squad/issues/300) asking for themes has been open since May 2026.
> Not intended to be contributed back. `cs version` reports `1.0.20-hamid.1` to distinguish it from an upstream build.
>
> Build and install from source:
>
> ```sh
> go build -o ~/.local/bin/cs .
> ```

[Claude Squad](https://smtg-ai.github.io/claude-squad/) is a terminal app that manages multiple [Claude Code](https://github.com/anthropics/claude-code), [Codex](https://github.com/openai/codex), [Gemini](https://github.com/google-gemini/gemini-cli) (and other local agents including [Aider](https://github.com/Aider-AI/aider)) in separate workspaces, allowing you to work on multiple tasks simultaneously.


![Claude Squad Screenshot](assets/screenshot.png)

### Highlights
- Complete tasks in the background (including yolo / auto-accept mode!)
- Manage instances and tasks in one terminal window
- Review changes before applying them, checkout changes before pushing them
- Each task gets its own isolated git workspace, so no conflicts

<br />

https://github.com/user-attachments/assets/aef18253-e58f-4525-9032-f5a3d66c975a

<br />

### Installation

Both Homebrew and manual installation will install Claude Squad as `cs` on your system.

#### Homebrew

```bash
brew install claude-squad
ln -s "$(brew --prefix)/bin/claude-squad" "$(brew --prefix)/bin/cs"
```

#### Manual

Claude Squad can also be installed by running the following command:

```bash
curl -fsSL https://raw.githubusercontent.com/smtg-ai/claude-squad/main/install.sh | bash
```

This puts the `cs` binary in `~/.local/bin`.

To use a custom name for the binary:

```bash
curl -fsSL https://raw.githubusercontent.com/smtg-ai/claude-squad/main/install.sh | bash -s -- --name <your-binary-name>
```

### Prerequisites

- [tmux](https://github.com/tmux/tmux/wiki/Installing)
- [gh](https://cli.github.com/)

### Usage

```
Usage:
  cs [flags]
  cs [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  debug       Print debug information like config paths
  help        Help about any command
  reset       Reset all stored instances
  version     Print the version number of claude-squad

Flags:
  -y, --autoyes          [experimental] If enabled, all instances will automatically accept prompts for claude code & aider
  -h, --help             help for claude-squad
  -p, --program string   Program to run in new instances (e.g. 'aider --model ollama_chat/gemma3:1b')
```

Run the application with:

```bash
cs
```
NOTE: The default program is `claude` and we recommend using the latest version.

<br />

<b>Using Claude Squad with other AI assistants:</b>
- For [Codex](https://github.com/openai/codex): Set your API key with `export OPENAI_API_KEY=<your_key>`
- Launch with specific assistants:
   - Codex: `cs -p "codex"`
   - Aider: `cs -p "aider ..."`
   - Gemini: `cs -p "gemini"`
- Make this the default, by modifying the config file (locate with `cs debug`)

<br />

#### Menu
The menu at the bottom of the screen shows available commands: 

##### Instance/Session Management
- `n` - Create a new session
- `N` - Create a new session with a prompt
- `D` - Kill (delete) the selected session
- `↑/j`, `↓/k` - Navigate between sessions

##### Actions
- `↵/o` - Attach to the selected session to reprompt
- `ctrl-q` - Detach from session
- `s` - Commit and push branch to github
- `c` - Checkout. Commits changes and pauses the session
- `r` - Resume a paused session
- `?` - Show help menu

##### Navigation
- `tab` - Switch between preview tab and diff tab
- `q` - Quit the application
- `shift-↓/↑` - scroll in diff view

### Configuration

Claude Squad stores its configuration in `~/.claude-squad/config.json`. You can find the exact path by running `cs debug`.

#### Profiles

Profiles let you define multiple named program configurations and switch between them when creating a new session. When more than one profile is defined, the session creation overlay shows a profile picker that you can navigate with `←`/`→`.

To configure profiles, add a `profiles` array to your config file and set `default_program` to the name of the profile to select by default:

```json
{
  "default_program": "claude",
  "profiles": [
    { "name": "claude", "program": "claude" },
    { "name": "codex", "program": "codex" },
    { "name": "aider", "program": "aider --model ollama_chat/gemma3:1b" }
  ]
}
```

Each profile has two fields:

| Field     | Description                                              |
|-----------|----------------------------------------------------------|
| `name`    | Display name shown in the profile picker                 |
| `program` | Shell command used to launch the agent for that profile  |

If no profiles are defined, Claude Squad uses `default_program` directly as the launch command (the default is `claude`).

#### Themes

Every color in the UI can be overridden from a `theme` object in the config file. Run `cs theme` to print the full default palette, ready to paste, and copy only the keys you want to change — anything you leave out keeps its default.

```json
{
  "theme": {
    "tab_active": "#ff8800",
    "tab_inactive": { "light": "#cccccc", "dark": "#444444" },
    "menu_text": "34"
  }
}
```

A color is either:

- a hex value, `"#RRGGBB"` or `"#RGB"`;
- an ANSI 256 index, `"0"` through `"255"`;
- a `{"light": ..., "dark": ...}` pair, applied according to the terminal background. A single side is enough — the other one mirrors it.

The keys, grouped by the area they affect:

| Group     | Keys                                                                                                                                     |
|-----------|------------------------------------------------------------------------------------------------------------------------------------------|
| Tab bar   | `tab_active`, `tab_inactive`, `tab_border`                                                                                               |
| Session list | `status_ready`, `status_paused`, `stat_added`, `stat_removed`, `text_primary`, `text_muted`, `text_footer`, `selection_bg`, `selection_fg`, `app_title_bg`, `app_title_fg` |
| Menu bar  | `menu_key`, `menu_desc`, `menu_separator`, `menu_action_group`, `menu_text`                                                               |
| Diff tab  | `diff_added`, `diff_removed`, `diff_hunk`                                                                                                |
| Feedback  | `error`, `paused_hint`                                                                                                                   |
| Overlays  | `overlay_accent`, `overlay_accent_fg`, `overlay_text`, `overlay_dim`, `overlay_shadow`, `confirm_border`                                  |
| Help screen | `help_title`, `help_header`, `help_key`, `help_desc`                                                                                   |

Note that `tab_active` and `tab_inactive` share the same default, so the tab bar looks unchanged until you set them to different values.

An unparseable color is ignored and falls back to its default rather than aborting the launch. `cs debug` lists every rejected key.

`border_style` sets the line weight of the tab bar and the pane frame — `"rounded"` (the default, light lines with rounded corners) or `"thick"`. Unicode box drawing has no heavy rounded corner, so thick borders are necessarily square; the junction glyphs switch weight along with the borders so the seams still line up.

```json
{ "theme": { "border_style": "thick" } }
```

#### Session colors

Press `C` on a session to tag it with a color. The picker lists the palette with a swatch of each actual color; `↑`/`↓` moves, `enter` confirms, `esc` cancels, and the first entry clears the tag.

A tagged session shows a solid bar of its color down the left edge of its row, and while it is selected its highlight band is tinted with the color instead of the theme's `selection_bg`. The active tab is **filled** with the color and the pane frame takes the same tint — so the session you are looking at is identifiable from the right-hand pane alone. Inactive tabs stay unfilled and keep their theme color, otherwise the active/inactive distinction would be lost.

The band uses a dimmed version of the color, blended 30% toward the terminal background: painted at full strength across a whole row it would swallow the title and wreck the green and red of the diff stats. The edge bar and the tab keep the raw color. This is the same relationship the default theme already has by hand between its `#A78BFA` accent and its `#3B2F5C` band.

The tab label switches to black or white depending on the fill's relative luminance, so a bright `amber` and a deep `blue` are both readable without configuring anything. ANSI indices are the exception: the terminal's palette decides what `"62"` actually looks like, so they cannot be measured and get light text.

The tag is stored in `~/.claude-squad/state.json` as the color's **name**, not its value, so re-theming a color updates every session already tagged with it. Untagged sessions write no extra key, and state files written before this feature existed load unchanged.

The eight default colors are `violet`, `blue`, `cyan`, `green`, `amber`, `orange`, `rose` and `slate`. Replace them with `instance_colors`, which unlike the keys above is substituted wholesale rather than merged — so you can offer fewer than eight:

```json
{
  "theme": {
    "instance_colors": [
      { "name": "urgent",  "color": "#ef4444" },
      { "name": "review",  "color": { "light": "#B45309", "dark": "#FBBF24" } },
      { "name": "backlog", "color": "#64748B" }
    ]
  }
}
```

A tag naming a color the theme no longer defines renders as untagged rather than failing.

### FAQs

#### Failed to start new session

If you get an error like `failed to start new session: timed out waiting for tmux session`, update the
underlying program (ex. `claude`) to the latest version.

### How It Works

1. **tmux** to create isolated terminal sessions for each agent
2. **git worktrees** to isolate codebases so each session works on its own branch
3. A simple TUI interface for easy navigation and management

### License

[AGPL-3.0](LICENSE.md)

### Star History

[![Star History Chart](https://api.star-history.com/svg?repos=smtg-ai/claude-squad&type=Date)](https://www.star-history.com/#smtg-ai/claude-squad&Date)
