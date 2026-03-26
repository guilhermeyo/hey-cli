---
name: hey
description: |
  Interact with HEY email via the HEY CLI. Read and send emails, manage boxes,
  calendars, todos, habits, time tracking, and journal entries. Use for ANY
  HEY-related question or action.
triggers:
  # Direct invocations
  - hey
  - /hey
  # Email actions
  - hey boxes
  - hey box
  - hey topic
  - hey reply
  - hey compose
  - hey drafts
  # Calendar actions
  - hey calendars
  - hey recordings
  - hey event
  - hey event list
  - hey event create
  - hey event edit
  - hey event delete
  - create event
  - delete event
  # Move contacts
  - hey move
  - move contact
  - move to feedbox
  - move to imbox
  # Extensions
  - hey extenzion
  - hey ext
  - email extension
  - create extension
  - list extensions
  # Todos
  - hey todo
  # Seen/unseen
  - hey seen
  - hey unseen
  - mark as read
  - mark as seen
  - mark as unseen
  - mark as unread
  # Habits
  - hey habit
  # Time tracking
  - hey timetrack
  # Journal
  - hey journal
  # Auth
  - hey auth
  # Common actions
  - check my email
  - read email
  - send email
  - reply to email
  - compose email
  - list mailboxes
  - check calendar
  - add todo
  - complete todo
  - track time
  - write journal
  # Questions
  - can I hey
  - how do I hey
  - what's in hey
  - what hey
  - does hey
  # My work
  - my emails
  - my inbox
  - my imbox
  - my todos
  - my calendar
  - my journal
  # URLs
  - hey.com
invocable: true
argument-hint: "[command] [args...]"
---

# /hey - HEY Email Workflow Command

CLI for HEY email: mailboxes, email threads, replies, compose, calendars, todos, habits, time tracking, and journal entries.

## Agent Invariants

**MUST follow these rules:**

1. **Always use `--json`** for structured, predictable output
2. **Authentication required** for all data commands — run `hey auth login` first
3. **HTML output** is available via `--html` for commands that return HTML content

## Quick Reference

| Task | Command |
|------|---------|
| List mailboxes | `hey boxes --json` |
| List emails in a box | `hey box imbox --json` |
| Read email thread | `hey topic 123 --json` |
| Reply to email | `hey reply 123 -m "Thanks!"` |
| Compose email | `hey compose --to user@example.com --subject "Hello"` |
| List drafts | `hey drafts --json` |
| List calendars | `hey calendars --json` |
| List calendar events | `hey recordings 123 --json` |
| List events | `hey event list --json` |
| Create event | `hey event create "Meeting" --date 2026-04-06 --start 10:00 --end 11:00` |
| Create all-day event | `hey event create "Holiday" --date 2026-04-06 --all-day` |
| Edit event | `hey event edit 123 --title "New title" --start 14:00 --end 15:00` |
| Delete event | `hey event delete 123` |
| Move contact | `hey move 123 --box feedbox --yes` |
| Move by contact ID | `hey move --contact 456 --box trailbox --yes` |
| List extensions | `hey ext list --json` |
| Create extension | `hey ext create sales --member alice@example.com` |
| Edit extension | `hey ext edit 123 --name support --member bob@example.com` |
| Delete extension | `hey ext delete 123 --yes` |
| List todos | `hey todo list --json` |
| Add todo | `hey todo add "Buy milk"` |
| Complete todo | `hey todo complete 123` |
| Uncomplete todo | `hey todo uncomplete 123` |
| Delete todo | `hey todo delete 123` |
| Mark as seen | `hey seen 12345` |
| Mark as unseen | `hey unseen 12345` |
| Complete habit | `hey habit complete` |
| Uncomplete habit | `hey habit uncomplete` |
| Start time tracking | `hey timetrack start` |
| Stop time tracking | `hey timetrack stop` |
| Current timer | `hey timetrack current --json` |
| List time entries | `hey timetrack list --json` |
| List journal entries | `hey journal list --json` |
| Read journal entry | `hey journal read 2024-03-15 --json` |
| Write journal entry | `hey journal write "Today was great"` |
| Check auth status | `hey auth status` |
| Print access token | `hey auth token` |
| Launch TUI | `hey` |

## Decision Trees

### Reading Email

```
Want to read email?
├── Which mailbox? → hey boxes --json
├── List emails in box? → hey box <name|id> --json
├── Read full thread? → hey topic <id> --json
├── Mark as seen? → hey seen <posting-id>
├── Mark as unseen? → hey unseen <posting-id>
└── Launch interactive UI? → hey (no args, launches TUI)
```

### Sending Email

```
Want to send email?
├── Reply to thread? → hey reply <topic_id> -m "message"
│   └── Open editor? → hey reply <topic_id> (omit -m to open $EDITOR)
├── Compose new? → hey compose --to <email> --subject "Subject"
│   └── With body? → hey compose --to <email> --subject "Subject" -m "Body"
└── Check drafts? → hey drafts --json
```

### Managing Calendar Events

```
Want to manage events?
├── List events? → hey event list --json
├── Create event? → hey event create "Title" --date YYYY-MM-DD --start HH:MM --end HH:MM
│   ├── All-day? → add --all-day (omit --start/--end)
│   ├── With reminder? → add --reminder 30m (or 1h, 1d)
│   └── Specific calendar? → add --calendar <id>
├── Edit event? → hey event edit <id> --title "New" --start 14:00 --end 15:00
└── Delete event? → hey event delete <id>
```

### Moving Contacts

```
Want to move a contact to a different box?
├── By topic ID? → hey move <topic-id> --box feedbox --yes
├── By contact ID? → hey move --contact <id> --box trailbox --yes
└── Valid boxes: imbox, feedbox, asidebox, laterbox, trailbox, bubblebox
```

### Managing Extensions

```
Want to manage email extensions?
├── List? → hey ext list --json
├── Create? → hey ext create <name> --member <email>
├── Edit? → hey ext edit <id> --name <new-name> --member <email>
└── Delete? → hey ext delete <id> --yes
```

### Managing Todos

```
Want to manage todos?
├── List todos? → hey todo list --json
├── Add todo? → hey todo add "Task description"
├── Complete? → hey todo complete <id>
├── Uncomplete? → hey todo uncomplete <id>
└── Delete? → hey todo delete <id>
```

## Resource Reference

### Email - Boxes

```bash
hey boxes --json                              # List all mailboxes
hey box imbox --json                          # List emails in Imbox (by name)
hey box 123 --json                            # List emails in box (by ID)
```

Box names: `imbox`, `the_feed`, `paper_trail`, `set_aside`, `reply_later`, `screened_out`

**Response format:** `hey box` returns `{"box": {...}, "postings": [...]}`. Each posting has: `id`, `name` (subject), `seen` (read status), `created_at`, `contacts`, `summary`, `app_url`.

### Email - Topics

```bash
hey topic 123 --json                          # Read full email thread
hey topic 123 --html                          # Read with raw HTML content
```

### Email - Reply & Compose

```bash
hey reply 123 -m "Thanks!"                   # Reply with inline message
hey reply 123                                 # Reply via $EDITOR
hey compose --to user@example.com --subject "Hello"         # Compose new
hey compose --to user@example.com --subject "Hi" -m "Body"  # With body
```

### Email - Seen/Unseen

```bash
hey seen 12345                                # Mark posting as seen
hey seen 12345 67890                          # Mark multiple postings as seen
hey unseen 12345                              # Mark posting as unseen
hey unseen 12345 67890                        # Mark multiple postings as unseen
```

Takes posting IDs (the `id` field from `hey box` output).

### Drafts

```bash
hey drafts --json                             # List drafts
```

### Calendars

```bash
hey calendars --json                          # List calendars (returns array of {id, name, kind})
hey recordings 123 --json                     # List events in calendar
```

**Response format:** `hey recordings` returns `{"Calendar::Event": [...]}`. Each event has: `id`, `title`, `starts_at`, `ends_at`, `all_day`, `recurring`, `starts_at_time_zone`. Access events via `.["Calendar::Event"]` in jq.

### Calendar Events

```bash
hey event list --json                         # List all events
hey event list --limit 10 --json              # List with limit
hey event create "Meeting" --date 2026-04-06 --start 10:00 --end 11:00  # Create event
hey event create "Holiday" --date 2026-04-06 --all-day                  # All-day event
hey event create "Standup" --date 2026-04-06 --start 09:00 --end 09:30 --reminder 30m --reminder 1d
hey event edit 123 --title "New title" --start 14:00 --end 15:00  # Edit event
hey event delete 123                          # Delete event
```

### Move Contacts

```bash
hey move 123 --box feedbox --yes              # Move contact by topic ID
hey move --contact 456 --box trailbox --yes   # Move by contact ID
```

Box kinds: `imbox`, `feedbox`, `asidebox`, `laterbox`, `trailbox`, `bubblebox`

### Extensions

```bash
hey ext list --json                           # List email extensions
hey ext create sales --member alice@example.com  # Create extension
hey ext create support --member a@ex.com --member b@ex.com  # Multiple members
hey ext edit 123 --name new-name --member a@ex.com  # Edit extension
hey ext delete 123 --yes                      # Delete extension
```

### Todos

```bash
hey todo list --json                          # List all todos
hey todo add "Task description"                        # Add a todo
hey todo complete 123                         # Mark complete
hey todo uncomplete 123                       # Mark incomplete
hey todo delete 123                           # Delete a todo
```

### Habits

```bash
hey habit complete                            # Mark habit complete for today
hey habit uncomplete                          # Unmark habit for today
```

### Time Tracking

```bash
hey timetrack start                           # Start timer
hey timetrack stop                            # Stop timer
hey timetrack current --json                  # Show current timer
hey timetrack list --json                     # List time entries
```

### Journal

```bash
hey journal list --json                       # List journal entries
hey journal read 2024-03-15 --json            # Read entry by date
hey journal write "Today's entry"                     # Write entry inline
hey journal write                             # Write entry via $EDITOR
```

### Authentication

```bash
hey auth login                                # Log in (browser-based OAuth)
hey auth status                               # Check if authenticated
hey auth logout                               # Log out
```

If a command fails with an auth error, run `hey auth status` to check, then `hey auth login` to re-authenticate.
