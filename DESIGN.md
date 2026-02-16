



You said:
I want to build an app that is essentially a terminal of terminals.

Essentially as a tool to organize and track work.

There is like a top level view, which is basically a view like “finder”, but just the line item view, no fancy graphics, and there’s different views to group items in different ways.

E.g. it could be grouped by jira item or grouped by repository or grouped by category (where I am author vs where I am a reviewer) or nested views of these, open sessions only, and pull request state. Kind of like a pivot table sort of view.

And then each one of these groupings (“collections” we can call them) will relate to one or more coding task. Each coding task will have one or zero Claude code CLI or codex CLI sessions associated with it. A session can be open or closed. If it’s open, the terminal can be made to fill this app window except for a navigation pane more or less.

Tmux or hosted iTerm or whatever — or a brand new window that these clue can run in no problem. I need to be able to type in it, use left and right arrows, copy / paste, etc… use normal terminal navigation keyboard shortcuts.

What language and what framework would you build this on and why? Desktop app, macOS only requirements. No portability required.


ChatGPT said:

Thought for 26s
If it’s macOS-only and the core of the product is lots of real terminal I/O with “it just works” keyboard behavior, I’d build it native:

My pick: Swift + AppKit (with SwiftUI where it helps)
Why this is the best fit
Terminal input correctness (arrow keys, option/alt combos, IME, copy/paste, focus, accessibility) is much easier to get “native-feeling” on macOS with AppKit controlling the windowing and event system.

You can integrate the stuff macOS users expect: menu bar shortcuts, global shortcuts, Services, drag & drop, “Open Recent,” Keychain, sandbox entitlements, etc.

You won’t be fighting WebKit edge cases for key handling or text input when your app’s whole value is being a “terminal of terminals”.

How to do the terminal part
You basically have two solid implementation paths:

A) Embed a native terminal emulator view (recommended)

Use SwiftTerm as the terminal emulator widget/engine. It’s specifically “terminal emulator in Swift” and includes an AppKit front-end.

Under the hood, you still spawn real processes via a PTY (pseudo-terminal) and feed bytes to/from the emulator.

B) Use a WebKit view + xterm.js

You run xterm.js in a WKWebView and connect it to a local PTY (Swift side) over a message bridge.

This can be great if you want to reuse web UI components for the “pivot-table-like” organizer view, but terminal key-handling can be a bit more fiddly than fully native.

A practical architecture that matches your mental model
Left pane: “Finder-style” list + grouping/pivot views (task collections)

Right pane: active terminal (fills the window except navigation)

Model: tasks ↔ sessions (0/1) + status (open/closed), plus metadata (repo, Atlassian Jira key, PR link/state, author/reviewer, etc.)

Storage: SQLite (direct) or Core Data on top of SQLite

Rough sketch:

+-------------------------------------------------------------+
| Collections (group/pivot) | Terminal tab / session view      |
|---------------------------|----------------------------------|
| Jira: PROJ-123            | [PTY] <-> [Terminal emulator]    |
|   repo-a                  |  claude/codex/tmux/shell         |
|   repo-b                  |  bytes in/out, scrollback        |
| Reviews (me)              |  copy/paste, arrows, shortcuts   |
| Open sessions             |                                  |
+-------------------------------------------------------------+
Why I’m not recommending “control iTerm2”
iTerm2’s Python API is cool for automation, but it’s disabled by default for security and it’s fundamentally about controlling iTerm2, not embedding a terminal inside your app.

So it’s a neat prototype hack, not a great foundation for a product like this.

Strong alternative if you want maximum iteration speed: Electron + xterm.js + node-pty
Why people choose this

UI for your “pivot-table-ish” organizer is fast to build with React/table libs.

xterm.js is battle-tested (VS Code uses it).

node-pty is a common pairing for real PTYs; folks are actively building “agent terminal IDEs” this way right now.

Downsides

More memory/CPU overhead than native.

macOS input quirks can still happen (usually solvable, but you’ll spend time there).

“Lightweight Electron” alternative: Tauri (v2) + web UI + PTY
Tauri v2 is real and actively documented, and there are examples of building a terminal emulator in Tauri with xterm.js.

This is compelling if you want a web UI but don’t want Electron’s weight. The tradeoff is more complexity (Rust + frontend + IPC), and you still rely on a webview for the UI.

My bottom-line recommendation
If you’re serious about “normal terminal navigation keyboard shortcuts” being flawless and you don’t need portability:

Build it in native Swift:

UI: SwiftUI + AppKit (use AppKit for the list + window/shortcut plumbing where SwiftUI is annoying)

Terminal: SwiftTerm embedded view + your own PTY/session manager

Persistence: SQLite/Core Data

Integrations: GitHub + Jira APIs via OAuth, async/await

If you want, I can also give you a concrete “first milestone” plan (like: create a PTY-backed terminal tab, persist sessions, then build the grouping/pivot layer, then wire GitHub PR state, then Jira).


You said:
Also is there a framework or app already that I can configure to do this without building entirely from scratch? Or are these things you mentioned as close to that as I can get?

ChatGPT said:

Thought for 27s
You can get pretty close without building a bespoke macOS app, but there isn’t (yet) a single off-the-shelf thing that does “pivot-table-ish task dashboard + embedded full terminal sessions per task” exactly the way you described. What exists today tends to cover either the dashboard side or the terminal/session side really well—so the practical answer is usually “compose 2–3 tools.”

Here are the best “don’t build from scratch” paths, in descending order of “feels closest to your idea”:

1) Closest in one window: VS Code as the shell
If “one app window” matters most, this is the most realistic option.

Finder-ish navigation + grouping: VS Code’s Explorer + custom views (and you can add workspace-level structure).

PR state: GitHub Pull Requests & Issues extension is built for review/triage inside VS Code.

Jira items: Atlassian’s VS Code extension supports Jira work items in-editor.

Terminals: VS Code terminals are “real enough” for arrows, copy/paste, shortcuts.

Where it won’t match your vision: the “pivot table” / nested grouping (repo → jira → author/reviewer → PR state) will be approximate, unless you build a VS Code extension that provides that exact tree view. But you might not need to—many people get 80% here.

2) Best dashboard for Jira + PRs: GitKraken Launchpad (then jump to terminals)
If the top-level “work organizer” is the hard part, this is the best ready-made dashboard I know.

Unified view of PRs/issues/tasks across providers + filtering (repos, sprints/milestones, etc.).

Where it won’t match your vision: it’s not a “terminal of terminals.” It’s a dashboard that can open things, but it doesn’t embed/own your interactive terminals the way you described.

3) Terminal-first version of your idea: WezTerm or kitty + a small “launcher/TUI”
If you’re willing to make the organizer itself a text UI (inside a pane), you can get surprisingly close with almost no GUI work:

Option A: WezTerm
Has workspaces and a built-in multiplexer layer (tabs/windows/panes/workspaces).

You can script session behaviors and even use plugins for session save/restore.

Option B: kitty
Has session files (define tabs/windows/layouts + what to run).

Has remote control (so a script/app can create tabs, switch, send text, rename, etc.).

Then you write a small “task dashboard” program (TUI) that:

pulls Jira + PR data via CLIs/APIs,

shows your grouped views,

and on “enter” opens/switches to the right workspace/tab/session (via WezTerm mux calls or kitty remote control).

This gets you a “terminal of terminals” vibe fast, and the terminals are real.

Downside: your “Finder-like” left pane is now a terminal UI, not a native macOS list control.

4) Raycast as the command center (fastest to try)
Raycast is great as the “top-level launcher + quick dashboards”:

It has Jira and GitHub extensions and can show lists / dashboards.

But it’s not going to host your interactive terminals; it’s more “select item → open elsewhere.”

My opinionated take
If you want one window with embedded terminals and reasonable Jira/PR integration today: VS Code is the closest “configure it and go.”

If you want the best PR/Jira triage dashboard with minimal effort: GitKraken Launchpad.

If your “secret sauce” is really session orchestration and “every task has a terminal brain attached”: go WezTerm/kitty + a small TUI, and only build the full native app once the workflow is proven.

If you tell me which PR host(s) you’re on (GitHub/GitLab/Bitbucket/Azure DevOps) and whether Jira is Cloud, I’ll point you to the most “drop-in” combo and the minimum glue needed.


You said:
GitHub only support needed. I don’t really care about jira. I’m curious about how vs code I can have different workspaces view that sorta like this across disparate repos and work trees.

I’m totally down with a TUI view for navigation rather than finder specifically. I maybe would like to use a mouse but I’m not overly concerned with keyboard only nav.


ChatGPT said:
If it’s GitHub-only and you’re fine with a TUI-style navigator, you actually don’t need to build anything big. You can get very close with one of these two paths:

Path 1 — VS Code as the “terminal of terminals” (zero custom build)
VS Code can absolutely span disparate repos and worktrees in one mental workspace.

1️⃣ Use a multi-root workspace
VS Code supports a single workspace containing multiple folders (even completely unrelated repos).

File → Add Folder to Workspace…

Save as my-tasks.code-workspace

That file is just JSON. Example:

{
  "folders": [
    { "path": "~/code/repo-a" },
    { "path": "~/code/repo-b/worktrees/feature-x" },
    { "path": "~/code/repo-c" }
  ]
}
Now you have one Explorer tree spanning multiple repos and worktrees.

You can:

Collapse by repo

Filter open editors only

Search across all repos at once

Open per-repo terminals

2️⃣ Use GitHub PR extension for grouping by state
Install:

GitHub Pull Requests & Issues extension

It gives you:

“Assigned to me”

“Created by me”

“Review requested”

PR status

Inline review + checkout PR into a branch

So that covers your:

author vs reviewer grouping

open PRs

PR state

It’s not pivot-table style, but the PR panel already groups by role and status.

3️⃣ Use Worktrees for per-task isolation
You mentioned “one coding task → one session”.

Git worktrees are perfect here.

Example:

repo-a/
repo-a/worktrees/feature-123
repo-a/worktrees/bugfix-42
Each worktree becomes a folder in the VS Code workspace.

Then:

Each task gets its own terminal tab

Each has its own branch

No context bleed

You basically get “session per task” without building anything.

4️⃣ Terminal behavior
VS Code’s terminal:

Handles arrows, Ctrl-A/E, Option keys

Copy/paste works

tmux inside works

Claude / Codex CLI run fine

You can split terminals per folder.

Where VS Code falls short
It won’t give you:

Arbitrary pivot views (e.g., “group by PR state → then by repo → then by reviewer”)

A custom top-level “collection” concept

For that, you’d need to write a small VS Code extension that renders a custom TreeView panel.

That’s not huge. A VS Code extension is mostly:

Register a TreeDataProvider

Populate it from GitHub API

On click → run a command (open folder, open terminal, checkout PR, etc.)

That’s 1–2 weeks of work max if you’re focused.

Path 2 — Pure TUI Orchestrator (honestly closer to your original idea)
Since you’re fine with a TUI:

Combine:
WezTerm (workspace-aware terminal multiplexer)

GitHub CLI (gh)

A small TUI dashboard

WezTerm supports workspaces and scripting.

GitHub CLI already supports filtering:

gh pr list --author @me
gh pr list --review-requested @me
gh pr list --state open
You could write a TUI in:

Go (Bubble Tea)

Rust (ratatui)

Python (Textual)

And have it:

Left pane:

My PRs

Review PRs

Open PRs

Drafts

Right pane:

Launch/switch to WezTerm workspace

Spawn shell in correct repo

Attach Claude/Codex

WezTerm can be controlled via CLI to switch workspaces.

That gives you:

✔ Real terminals
✔ Real keyboard behavior
✔ Arbitrary grouping logic
✔ No Electron
✔ No heavy GUI

This might actually be the cleanest solution.

My honest take
If you want to prototype fast:

Start with:

VS Code multi-root workspace

GitHub PR extension

Worktrees per task

See if it already scratches the itch.

If you feel constrained by the grouping model after a week, then build a small TUI orchestrator that sits above WezTerm.

I would not build a native macOS app yet. You’ll spend months solving windowing and event plumbing before you validate the workflow.

If you want, I can sketch:

A concrete VS Code workflow setup
or

A concrete “WezTerm + Bubble Tea + gh” architecture with command flow

Which direction feels more interesting to you?


You said:
Yeah why wezTerm
Over kitty though?

ChatGPT said:

Thought for 5s
If your goal is a “TUI navigator that can jump you into the right live terminal session”, the real deciding factor is: how cleanly can you model and switch “task workspaces” and keep them isolated?

Both kitty and WezTerm are excellent. My bias toward WezTerm for your use case is mostly about the “task workspace” abstraction and how naturally it maps to what you described.

Why WezTerm over kitty (for this exact idea)
1) Workspaces feel like “task contexts”
WezTerm has a first-class notion of workspaces where only the active workspace’s windows are shown in the GUI, and switching workspaces swaps what’s visible. That maps almost 1:1 to “I have a task collection, and the associated terminals should appear when I select it.”

kitty has “sessions” (which are great), but it’s more like “load a layout / set of tabs/windows,” not “swap the entire visible universe of windows” in the same way.

2) Built-in multiplexer model (tmux-ish without tmux)
WezTerm’s docs are very explicit that multiplexing panes/tabs/windows is core, and it can persist local tabs/windows until closed.

That makes it natural to treat WezTerm as the “host runtime” for your sessions without layering tmux unless you want it.

kitty is also a great multiplexer in practice, but the “control plane” story tends to be “remote control + launch + sessions,” which is powerful but a little more DIY.

3) A CLI that returns stable IDs is handy for a TUI controller
WezTerm has wezterm cli spawn which can create a new tab/window and returns a pane-id on success. That’s super convenient if your TUI wants to keep a map like task_id -> pane_id.

kitty remote control is very capable (open windows, send text, change titles, etc.).

But in my experience, you end up doing a bit more “query state + target selection” logic yourself (totally doable, just more glue).

Where kitty is actually better
If any of these are your priorities, kitty may win:

You want a dead-simple reproducible config and session files as the “source of truth” (kitty sessions are intentionally just text files).

You want to lean hard on remote control (JSON protocol, lots of commands, easy to script).

You prefer kitty’s ecosystem / “power keyboard user” philosophy, and you’re happy to wire the “workspace concept” yourself from sessions + focus rules.

My actual recommendation (opinionated)
If you want the cleanest “task → workspace → terminals appear” mental model: WezTerm.

If you want the most scriptable control surface and you’re fine implementing your own “workspace switching semantics” from sessions: kitty is equally viable.

What I’d build first (either way)
A tiny TUI that does only this:

gh pr list --review-requested @me / --author @me etc.

Select an item

“Open or focus” the corresponding terminal context

WezTerm: spawn --workspace <task> + store pane id

kitty: kitty @ launch ... + tag windows by title + session file per task

If you tell me which vibe you want:

“swap entire workspace when I switch tasks” (WezTerm-ish), or

“lots of windows/tabs, I’ll just focus the right one quickly” (kitty-ish),

…I’ll give you a concrete command-level plan for the first prototype.


You said:
Ok let’s go with wezTerm but why did you choose bubble tea

ChatGPT said:
Bubble Tea is just the nicest “get a legit TUI fast” option if you want the navigator to feel like a real app, not a bash script with fzf.

Why I picked it for your WezTerm + gh orchestrator:

Why Bubble Tea (Go) fits this project
1) It’s built for the exact UI you’re imagining
A list/tree/pivot-ish navigator with:

multiple panes

live filtering/search

status badges (open/draft/merged)

keybindings + mouse support (optional)

…is basically Bubble Tea’s sweet spot. The ecosystem (especially bubbles) has ready-made list/table/textinput components so you’re not hand-rolling rendering and input handling.

2) Concurrency is clean for “refresh GitHub state while UI stays snappy”
You’ll likely want:

poll PRs every N seconds

update counts + statuses

keep selection stable

spawn/focus terminals instantly

Go’s concurrency makes it easy to do “background refresh” without the UI getting janky.

3) Shipping a single binary is delightful
For a tool like this, you want:

brew install yourtool / drop a binary in ~/bin

no Python env headaches

no node_modules

Go excels here.

4) Easy integration with gh and WezTerm CLI
You’ll be doing a lot of:

run gh pr list ... --json ...

parse JSON

run wezterm cli spawn ...

store IDs in a local sqlite/db/json file

Go is straightforward for subprocess + JSON + sqlite.

5) Mouse support exists (if you want it)
You said mouse is “nice but not required.” Bubble Tea/terminal UIs can support mouse events depending on terminal capabilities; it’s not a guarantee everywhere, but it’s doable enough for clicking list items / scrolling if you care later.

Why not Textual (Python)?
Textual is great and arguably prettier, but:

you’ll be managing Python packaging/envs on macOS

binary distribution is more annoying

subprocess + async can get fiddly depending on how you structure it

If you personally prefer Python, Textual is a totally legit choice—I just default to “this should install like a tool, not like an app.”

Why not ratatui (Rust)?
Also great. But Rust slows down iteration unless you’re already fluent, and you’ll spend more time on lifetimes/architecture than on “does this workflow slap?”

Why not just fzf?
You can prototype the selection part with fzf, but once you want:

multiple grouped views

persistent state (task → pane id)

counts and badges

background refresh

open/closed sessions list

…you quickly outgrow shell glue.

If you’re already Go-curious: Bubble Tea is the “fastest path to satisfying”
It’s the best “small amount of code → feels like a real product” ratio for TUIs.

If you want, I can outline a minimal v0 architecture like:

data model (PR → task)

views (Authored / Review Requested / By Repo / Open Sessions)

commands (spawn/focus/close)

persistence (sqlite or a single JSON file)

wezterm CLI calls (spawn + set tab title + workspace naming)

…and it’ll be immediately implementable.


You said:
Yeah outline a minimal v0 architecture for me

ChatGPT said:
Here’s a minimal v0 that already feels like “terminal of terminals,” but keeps scope tight.

v0 goal
A TUI that shows your GitHub PRs (authored + review-requested), lets you pick one, and then opens/focuses a WezTerm workspace + pane tied to that PR (or to a repo/branch you choose). It also tracks “open/closed sessions” locally.

Components
1) Data model
Keep it dead simple and durable. Store in a local SQLite DB (or JSON for first week).

Entities
PRItem (ephemeral, from GitHub)

id (GitHub PR node_id or number+repo)

repo (owner/name)

number

title

author

isDraft

state (OPEN/CLOSED/MERGED)

updatedAt

url

headRefName (branch)

baseRefName

TaskKey (your stable key)

v0: just repo#number (e.g. octo/repo#123)

later: could support “non-PR tasks” too

Session (persistent, your system of record)

task_key (PK)

workspace (string; e.g. pr-octo-repo-123)

pane_id (int64, nullable)

cwd (path, e.g. ~/code/repo or worktree path)

command (string array-ish; e.g. ["zsh"] or ["codex"])

status (open/closed/unknown)

last_seen_at

created_at

2) GitHub integration (v0 = gh)
Use GitHub CLI because it’s already authenticated and stable.

Fetch commands (two lists)
Authored open PRs:

gh pr list --author @me --state open --json ...

Review requested:

gh pr list --review-requested @me --state open --json ...

Fields to request:

number,title,isDraft,state,updatedAt,url,headRefName,baseRefName,repository

In Bubble Tea, do this as a background command and merge results.

Refresh strategy:

initial fetch on startup

manual refresh key (r)

optional: periodic refresh every 60–120s (nice but not required for v0)

3) WezTerm control plane
You want three operations:

A) “Open or focus” a session for a TaskKey
Algorithm:

Look up Session by task_key

If it has a pane_id, verify it still exists (optional v0)

If it exists → activate/focus it

Else → spawn a new pane/tab in a named workspace, store the new pane_id

B) Spawn
Use wezterm cli spawn ... and capture returned pane_id.
Put it in a workspace named from TaskKey.

Workspace naming convention (stable and reversible):

pr/<owner>/<repo>/<number> or slug it: pr-octo-repo-123

Spawn details:

--cwd <cwd>

--workspace <workspace>

optionally set a tab title to repo#123 (WezTerm lets you set titles via config/escape sequences; v0 you can just printf '\033]0;...\007' in shell)

C) Focus / activate
WezTerm has wezterm cli activate-pane --pane-id <id> and similar focus commands. (Exact command depends on version; in practice you can always “activate pane” or “switch workspace” then activate pane.)

v0 cheat if focusing is annoying:

Always spawn in the desired workspace and rely on WezTerm’s “show that workspace” behavior when a new pane appears.

Later: do proper focus.

4) Repo/worktree mapping (super minimal)
You need a cwd to drop into.

v0 approach: a config file mapping repo → local path.

~/.ttt/config.toml:

[repos]
"owner/repo-a" = "/Users/you/code/repo-a"
"owner/repo-b" = "/Users/you/code/repo-b"
Later you can auto-discover via:

gh repo view --json nameWithOwner + ghq / gh repo clone conventions

or scan a root directory for git repos

Worktrees (v0 optional):

If you want: store cwd per session as a worktree path you created for that PR.

You can add a key: w = “create/worktree checkout for this PR” (later).

TUI layout (Bubble Tea)
Views (tabs)
Top row: tabs (keys 1–3)

Review Requested

Authored

Open Sessions (from your DB, independent of GitHub state)

Each view is basically a list with:

left: repo#num

title

badges: DRAFT, updated age

maybe an icon for “has session open”

Keybindings (v0)
↑/↓ move

enter open/focus session (WezTerm)

o open PR in browser (gh pr view --web)

r refresh GitHub lists

s toggle “show only with sessions”

q quit

Optional:

/ filter/search (bubble textinput)

c close session (mark closed + optionally kill pane)

Mouse:

ignore in v0

Persistence (choose one)
Option A: JSON (fastest)
~/.ttt/state.json with sessions[].

Pros: 30 minutes to implement
Cons: concurrency + evolution later

Option B: SQLite (still easy, better)
Use modernc.org/sqlite (pure Go) or mattn/go-sqlite3 (cgo).

Schema:

create table if not exists sessions (
  task_key text primary key,
  workspace text not null,
  pane_id integer,
  cwd text not null,
  command text,
  status text not null,
  last_seen_at text,
  created_at text
);
Command flow (the important part)
On startup
Load sessions from DB

Kick off GitHub refresh commands (async)

Render UI immediately with “loading…”

On selecting PR + Enter
Derive task_key = repo#number

Determine cwd from config mapping

session = getOrCreateSession(task_key)

If session.pane_id present:

try wezterm activate-pane

if fails, clear pane_id and spawn new

If no pane_id:

pane_id = wezterm spawn --workspace session.workspace --cwd session.cwd

update session with pane_id, status=open, last_seen_at=now

Return to UI, mark item as “open session”

Close session (optional v0)
x: mark closed locally (don’t kill the pane yet)

v0: “closed” just means “don’t try to focus, spawn a fresh one next time”

later: add kill support via wezterm CLI

Suggested v0 milestones (in order)
Bubble Tea app with 3 tabs + dummy data

gh fetch + list rendering + refresh

Config mapping repo → path

wezterm cli spawn on Enter, store pane_id in JSON/SQLite

“Open Sessions” tab from DB

Focus existing pane_id (activate-pane) + fallback spawn

That’s enough to feel like your concept.

If you want, I can also give you:

a concrete config.toml format + example

a suggested TaskKey/workspace naming scheme that won’t bite you later

a skeleton Go project structure (cmd/ttt, internal/github, internal/wezterm, internal/store, internal/ui) and the exact JSON fields to request from gh so parsing is painless.
