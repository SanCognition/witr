# Output Modes

This diagram shows the different output formats available in witr.

```mermaid
graph TB
    Result[Result Object] --> OutputSelect{Output Mode}

    OutputSelect -->|default| Standard["Standard Output<br/>standard.go"]
    OutputSelect -->|--short| Short["Short Output<br/>short.go"]
    OutputSelect -->|--tree| Tree["Tree Output<br/>tree.go"]
    OutputSelect -->|--json| JSON["JSON Output<br/>json.go"]
    OutputSelect -->|--warnings| Warnings["Warnings Only<br/>standard.go"]
    OutputSelect -->|--env| Env["Env Only<br/>envonly.go"]

    subgraph "Standard Output"
        S1["Target: process_name"]
        S2["Process: name (pid N) [health] {forked}"]
        S3["User: username"]
        S4["Container: docker/podman/k8s"]
        S5["Service: service_name"]
        S6["Command: full command line"]
        S7["Started: relative time (absolute time)"]
        S8["Restarts: count"]
        S9["Why It Exists: init -> ... -> target"]
        S10["Source: source_type (name)"]
        S11["Source Details: type, plist, triggers, keepalive"]
        S12["Working Dir: /path"]
        S13["Git Repo: name (branch)"]
        S14["Listening: address:port"]
        S15["Socket: state + explanation"]
        S16["Energy: sleep prevention"]
        S17["Thermal: throttling state"]
        S18["Open Files: N of M (%)"]
        S19["Locks: locked files"]
        S20["Warnings: list of warnings"]
    end

    subgraph "Short Output"
        SH1["process1 (pid N) -> process2 (pid M) -> ... -> target (pid X)"]
    end

    subgraph "Tree Output"
        T1["init (pid 1)"]
        T2["  └─ parent (pid N)"]
        T3["    └─ child (pid M)"]
        T4["      └─ target (pid X)"]
    end

    subgraph "JSON Output"
        J1["{"]
        J2["  Target: {...}"]
        J3["  ResolvedTarget: string"]
        J4["  Process: {...}"]
        J5["  RestartCount: number"]
        J6["  Ancestry: [...]"]
        J7["  Source: {...}"]
        J8["  Warnings: [...]"]
        J9["  SocketInfo: {...}"]
        J10["  ResourceContext: {...}"]
        J11["  FileContext: {...}"]
        J12["}"]
    end

    subgraph "Env Output"
        E1["Command: full command line"]
        E2["Environment:"]
        E3["  VAR1=value1"]
        E4["  VAR2=value2"]
    end

    Standard --> S1 --> S2 --> S3 --> S4 --> S5 --> S6 --> S7 --> S8 --> S9 --> S10 --> S11 --> S12 --> S13 --> S14 --> S15 --> S16 --> S17 --> S18 --> S19 --> S20
    Short --> SH1
    Tree --> T1 --> T2 --> T3 --> T4
    JSON --> J1 --> J2 --> J3 --> J4 --> J5 --> J6 --> J7 --> J8 --> J9 --> J10 --> J11 --> J12
    Env --> E1 --> E2 --> E3 --> E4

    style Result fill:#e3f2fd
    style Standard fill:#c8e6c9
    style Short fill:#fff3e0
    style Tree fill:#f3e5f5
    style JSON fill:#e0f7fa
    style Env fill:#fce4ec
```

## Color Scheme

```mermaid
graph LR
    subgraph "ANSI Color Codes"
        Reset["Reset: \\033[0m"]
        Red["Red: \\033[31m<br/>(Warnings, Errors)"]
        Green["Green: \\033[32m<br/>(Commands, Dirs)"]
        Blue["Blue: \\033[34m<br/>(Target, Container, Service)"]
        Cyan["Cyan: \\033[36m<br/>(User, Source, Files)"]
        Magenta["Magenta: \\033[35m<br/>(Started, Ancestry, Tree)"]
        DimYellow["Dim Yellow: \\033[2;33m<br/>(Restarts, Forked, Workarounds)"]
        Bold["Bold/Dim: \\033[2m<br/>(PID numbers)"]
    end

    style Red fill:#ffcdd2
    style Green fill:#c8e6c9
    style Blue fill:#bbdefb
    style Cyan fill:#b2ebf2
    style Magenta fill:#e1bee7
    style DimYellow fill:#fff9c4
```

## Output Examples

### Standard Output
```
Target      : nginx

Process     : nginx (pid 1234) [healthy]
User        : www-data
Command     : nginx -g daemon off;
Started     : 2 days ago (Mon 2025-01-01 10:00:00 +0000)

Why It Exists :
  launchd (pid 1) -> nginx (pid 1234)

Source      : com.example.nginx (launchd)
              Type : Launch Daemon
              Plist : /Library/LaunchDaemons/com.example.nginx.plist
              Trigger : RunAtLoad (starts at login/boot)
              KeepAlive : Yes (restarts if killed)

Working Dir : /var/www
Listening   : 0.0.0.0:80
              0.0.0.0:443

Warnings    :
  - Process is listening on a public interface
```

### Short Output
```
launchd (pid 1) -> nginx (pid 1234)
```

### Tree Output
```
launchd (pid 1)
  └─ nginx (pid 1234)
```
