# Output
Output demo.gif
Set Shell "powershell"

# Terminal appearance
Set FontSize 18
Set Width 1400
Set Height 900
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms
Set Framerate 24
Set Padding 30

# ─────────────────────────────────────────
# 1. Show stats first
# ─────────────────────────────────────────
Type "repowalk stats"
Sleep 300ms
Enter
Sleep 4s

# ─────────────────────────────────────────
# 2. Launch TUI
# ─────────────────────────────────────────
Type "repowalk ui"
Sleep 300ms
Enter
Sleep 3s

# ─────────────────────────────────────────
# 3. Navigate to cmd/ (row 12, 0-indexed)
#    0:.agents  1:.git  2:.github  3:.gitignore
#    4:.goreleaser.yaml  5:.repowalk.json
#    6:CODE_OF_CONDUCT.md  7:CONTRIBUTING.md
#    8:LICENSE  9:Makefile  10:README.md
#    11:_config.yml  12:cmd/   ← target
#    Need 13 Down presses (start at row 0, press to reach row 12... wait
#    cursor starts AT row 0, Down 1 = row 1, Down 13 = row 13... hmm)
#    Screenshot showed off-by-1: landed on _config.yml with 12 Downs.
#    So cmd/ needs 13 Downs.
# ─────────────────────────────────────────
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 150ms
Down 1
Sleep 400ms

# Now on cmd/ — Exclusive Select (double-tap Space within 500ms)
Space
Sleep 80ms
Space
Sleep 2.5s

# ─────────────────────────────────────────
# 4. Navigate to internal/ — cmd/ stays collapsed after exclusive select
#    From cmd/ (row 12), go down:
#    13:demo.gif  14:demo.tape  15:go.mod
#    16:go.sum    17:go_rewrite_roadmap.md  18:internal/
#    That's 6 downs.
# ─────────────────────────────────────────
Down 1
Sleep 200ms
Down 1
Sleep 200ms
Down 1
Sleep 200ms
Down 1
Sleep 200ms
Down 1
Sleep 200ms
Down 1
Sleep 400ms

# Now on internal/ — normal select
Space
Sleep 2.5s

# ─────────────────────────────────────────
# 5. Save with `s` — two-step dialog
#    Step 1: clear default and type custom filename
#    Step 2: accept "." (project root) as-is
# ─────────────────────────────────────────
Type "s"
Sleep 1.5s

# Step 1: clear default filename and type our custom name
Ctrl+E
Sleep 100ms
Ctrl+U
Sleep 200ms
Type "demo_super_context.md"
Sleep 600ms
Enter
Sleep 1s

# Type the text
Backspace 1
Type "your/fav/dir"
Sleep 1.5s

# Erase it letter by letter (5 characters = Backspace 5)
Backspace 12
Type "."
Enter
Sleep 2.5s

# ─────────────────────────────────────────
# 6. Search for the newly saved file in the TUI
# ─────────────────────────────────────────
Type "/"
Sleep 500ms
Type "demo_super"
Sleep 2s

# The right pane now shows the file content — pause to let viewers read it
Sleep 3s
# ─────────────────────────────────────────
# 7. Quit
# ─────────────────────────────────────────
Escape
Type "q"
Sleep 2s
