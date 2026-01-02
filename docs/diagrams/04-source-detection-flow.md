# Source Detection Flow

This diagram shows how witr determines what started a process.

```mermaid
flowchart TD
    Start[Process Ancestry Chain] --> Detect[source.Detect]

    Detect --> CheckContainer{Container?}
    CheckContainer -->|Yes| ContainerSource[Container Source]
    CheckContainer -->|No| CheckSupervisor{Supervisor?}

    CheckSupervisor -->|Yes| SupervisorSource[Supervisor Source]
    CheckSupervisor -->|No| CheckSystemd{Systemd?}

    CheckSystemd -->|Yes| SystemdSource[Systemd Source]
    CheckSystemd -->|No| CheckLaunchd{Launchd?}

    CheckLaunchd -->|Yes| LaunchdSource[Launchd Source]
    CheckLaunchd -->|No| CheckCron{Cron?}

    CheckCron -->|Yes| CronSource[Cron Source]
    CheckCron -->|No| CheckShell{Shell?}

    CheckShell -->|Yes| ShellSource[Shell Source]
    CheckShell -->|No| UnknownSource[Unknown Source]

    subgraph "Container Detection"
        ContainerSource --> ReadCgroup["/proc/PID/cgroup"]
        ReadCgroup --> CgroupMatch{Content Contains?}
        CgroupMatch -->|docker| Docker["docker<br/>Confidence: 0.9"]
        CgroupMatch -->|podman/libpod| Podman["podman<br/>Confidence: 0.9"]
        CgroupMatch -->|kubepods| K8s["kubernetes<br/>Confidence: 0.9"]
        CgroupMatch -->|colima| Colima["colima<br/>Confidence: 0.9"]
        CgroupMatch -->|containerd| Containerd["containerd<br/>Confidence: 0.8"]
    end

    subgraph "Supervisor Detection"
        SupervisorSource --> CheckAncestry1[Check Ancestry Commands]
        CheckAncestry1 --> SupMatch{Command/Cmdline Contains?}
        SupMatch -->|pm2| PM2["pm2<br/>Confidence: 0.9"]
        SupMatch -->|supervisord| Supervisord["supervisord<br/>Confidence: 0.7"]
        SupMatch -->|gunicorn| Gunicorn["gunicorn<br/>Confidence: 0.7"]
        SupMatch -->|uwsgi| Uwsgi["uwsgi<br/>Confidence: 0.7"]
        SupMatch -->|s6/runsv| S6Runit["s6/runit<br/>Confidence: 0.7"]
        SupMatch -->|monit| Monit["monit<br/>Confidence: 0.7"]
        SupMatch -->|circus| Circus["circus<br/>Confidence: 0.7"]
        SupMatch -->|forever| Forever["forever<br/>Confidence: 0.7"]
        SupMatch -->|tini| Tini["tini<br/>Confidence: 0.7"]
    end

    subgraph "Systemd Detection (Linux)"
        SystemdSource --> CheckPID1Sys{PID 1 == systemd?}
        CheckPID1Sys -->|Yes| SysD["systemd<br/>Confidence: 0.8"]
    end

    subgraph "Launchd Detection (macOS)"
        LaunchdSource --> CheckPID1Launch{PID 1 == launchd?}
        CheckPID1Launch -->|Yes| GetLaunchdInfo[Get Launchd Info]
        GetLaunchdInfo --> LaunchctlBlame["launchctl blame PID"]
        LaunchctlBlame --> FindPlist[Find Plist File]
        FindPlist --> ParsePlist[Parse Plist XML]
        ParsePlist --> ExtractDetails["Extract:<br/>- Label<br/>- Domain<br/>- Triggers<br/>- KeepAlive"]
        ExtractDetails --> LaunchD["launchd service<br/>Confidence: 0.95"]
    end

    subgraph "Cron Detection"
        CronSource --> CheckAncestry2{Command == cron/crond?}
        CheckAncestry2 -->|Yes| CronD["cron<br/>Confidence: 0.6"]
    end

    subgraph "Shell Detection"
        ShellSource --> CheckAncestry3{Command in shells?}
        CheckAncestry3 --> ShellList["bash, zsh, sh, fish"]
        ShellList --> ShellD["shell<br/>Confidence: 0.5"]
    end

    subgraph "Unknown"
        UnknownSource --> UnknownD["unknown<br/>Confidence: 0.2"]
    end

    Docker --> Final[Source Object]
    Podman --> Final
    K8s --> Final
    Colima --> Final
    Containerd --> Final
    PM2 --> Final
    Supervisord --> Final
    Gunicorn --> Final
    Uwsgi --> Final
    S6Runit --> Final
    Monit --> Final
    Circus --> Final
    Forever --> Final
    Tini --> Final
    SysD --> Final
    LaunchD --> Final
    CronD --> Final
    ShellD --> Final
    UnknownD --> Final

    style Start fill:#e3f2fd
    style Final fill:#c8e6c9
    style ContainerSource fill:#ffebee
    style SupervisorSource fill:#fff3e0
    style SystemdSource fill:#e8f5e9
    style LaunchdSource fill:#f3e5f5
    style CronSource fill:#e0f7fa
    style ShellSource fill:#fce4ec
```

## Detection Priority

Sources are checked in priority order. The first match wins:

```mermaid
graph TD
    subgraph "Priority Order (High to Low)"
        direction TB
        P1["1. Container<br/>(docker, podman, k8s, colima, containerd)"] --> P2
        P2["2. Supervisor<br/>(pm2, supervisord, gunicorn, uwsgi, etc.)"] --> P3
        P3["3. Systemd<br/>(Linux only)"] --> P4
        P4["4. Launchd<br/>(macOS only)"] --> P5
        P5["5. Cron"] --> P6
        P6["6. Shell<br/>(bash, zsh, sh, fish)"] --> P7
        P7["7. Unknown"]
    end

    style P1 fill:#ffcdd2
    style P2 fill:#ffe0b2
    style P3 fill:#c8e6c9
    style P4 fill:#e1bee7
    style P5 fill:#b2ebf2
    style P6 fill:#f8bbd9
    style P7 fill:#cfd8dc
```

## Launchd Detail Extraction

```mermaid
graph LR
    subgraph "Launchd Info"
        Label["Label<br/>(Service Name)"]
        Domain["Domain<br/>(system/gui/user)"]
        PlistPath["Plist Path"]
        Triggers["Triggers"]
    end

    subgraph "Trigger Types"
        RunAtLoad["RunAtLoad<br/>(starts at login/boot)"]
        StartInterval["StartInterval<br/>(periodic)"]
        StartCalendar["StartCalendarInterval<br/>(scheduled)"]
        WatchPaths["WatchPaths<br/>(file changes)"]
        QueueDirs["QueueDirectories<br/>(directory watch)"]
        KeepAlive["KeepAlive<br/>(auto-restart)"]
    end

    Triggers --> RunAtLoad
    Triggers --> StartInterval
    Triggers --> StartCalendar
    Triggers --> WatchPaths
    Triggers --> QueueDirs
    Triggers --> KeepAlive

    style Label fill:#e8eaf6
    style Triggers fill:#fff3e0
```
