package main

// Shared path resolution helpers to avoid duplication across commands.

import (
    "os"
    "gopkg.in/yaml.v3"
)

// dedupePreserveOrder removes duplicate strings while preserving the first
// occurrence order.
func dedupePreserveOrder(in []string) []string {
    seen := make(map[string]struct{}, len(in))
    out := make([]string, 0, len(in))
    for _, s := range in {
        if _, ok := seen[s]; ok {
            continue
        }
        seen[s] = struct{}{}
        out = append(out, s)
    }
    return out
}

// resolveSystemPaths returns the current PATH entries, deduped.
func resolveSystemPaths() ([]string, error) {
    paths, err := getCurrentPath()
    if err != nil {
        return nil, err
    }
    return dedupePreserveOrder(paths), nil
}

// resolvePathuniPaths returns config-derived paths for the current context,
// respecting the global prune flag. When prune is "pathuni" or "all", only
// existing paths are included (current behavior). When prune is "none" or
// "system", include paths that pass tag filtering regardless of existence.
func resolvePathuniPaths() ([]string, error) {
    configPath := getConfigPath()
    osName, _ := getOSName()
    shellName, _ := getShellName()
    tagFilter, err := parseTagFlags(tagsInclude, tagsExclude)
    if err != nil { return nil, err }

    statuses, _, err := EvaluateConfigDetailed(configPath, osName, shellName, tagFilter)
    if err != nil { return nil, err }

    var out []string
    switch prune {
    case "pathuni", "all":
        for _, st := range statuses {
            if st.Included {
                out = append(out, st.Path)
            }
        }
    case "none", "system":
        for _, st := range statuses {
            if st.PassesFilter { // include even if not existing
                out = append(out, st.Path)
            }
        }
    default:
        // Fallback to safe behavior
        for _, st := range statuses { if st.Included { out = append(out, st.Path) } }
    }
    return dedupePreserveOrder(out), nil
}

// mergeFull merges pathuni and system lists according to the precedence flag
// and returns a deduped, order-preserving result. When pathuniFirst is true,
// pathuni entries come before system entries; otherwise system entries first.
func mergeFull(pathuni, system []string, pathuniFirst bool) []string {
    // Ensure each slice is internally deduped first to provide stable results.
    p := dedupePreserveOrder(pathuni)
    s := dedupePreserveOrder(system)

    var combined []string
    if pathuniFirst {
        combined = append(append([]string{}, p...), s...)
    } else {
        combined = append(append([]string{}, s...), p...)
    }
    return dedupePreserveOrder(combined)
}

// filterExisting returns only entries that exist and are directories.
func filterExisting(paths []string) []string {
    out := make([]string, 0, len(paths))
    for _, p := range paths {
        expanded := os.ExpandEnv(p)
        if info, err := os.Stat(expanded); err == nil && info.IsDir() {
            out = append(out, expanded)
        }
    }
    return out
}

// getPowerShellPathEntries returns system path files as PathEntry(ies) when
// configured to be injected on the pathuni side (as=pathuni). When tags are
// provided in the YAML, they are attached explicitly; otherwise Tags=nil so
// platform tag inheritance applies.
func getPowerShellPathEntries(shell string, platformConfig PlatformConfig) []PathEntry {
    var entries []PathEntry
    if shell != "powershell" || platformConfig.PowerShell == nil || !platformConfig.PowerShell.IncludeSystemPaths {
        return entries
    }
    as := platformConfig.PowerShell.IncludeSystemPathsAs
    if as == "" { as = "system" }
    if as != "pathuni" {
        return entries
    }
    sys, err := getSystemPaths()
    if err != nil {
        return entries
    }
    var tags []string
    if platformConfig.PowerShell.Tags != nil {
        // Explicit tags provided (possibly empty slice to break inheritance)
        tags = append([]string{}, platformConfig.PowerShell.Tags...)
    } else {
        // nil indicates inheritance
        tags = nil
    }
    for _, p := range sys {
        entries = append(entries, PathEntry{Path: p, Tags: tags})
    }
    return entries
}

// resolveSystemPathsContext returns system paths considering config context.
// When shell=powershell and the platform config has include_system_paths true
// with classification as "system" (default), also include macOS system path
// files from /etc/paths and /etc/paths.d/*.
func resolveSystemPathsContext(configPath, platform, shell string) ([]string, error) {
    // Start with current PATH entries (deduped)
    sys, err := getCurrentPath()
    if err != nil {
        return nil, err
    }
    sys = dedupePreserveOrder(sys)

    // Only enhance for PowerShell when configured
    if shell == "powershell" {
        data, err := os.ReadFile(configPath)
        if err == nil {
            var cfg Config
            if yaml.Unmarshal(data, &cfg) == nil {
                var p PlatformConfig
                switch platform {
                case "macOS":
                    p = cfg.MacOS
                case "Linux":
                    p = cfg.Linux
                }
                if p.PowerShell != nil && p.PowerShell.IncludeSystemPaths {
                    as := p.PowerShell.IncludeSystemPathsAs
                    if as == "" { as = "system" }
                    if as == "system" {
                        if extra, err := getSystemPaths(); err == nil {
                            sys = mergeFull([]string{}, append(sys, extra...), false) // system-first irrelevant; we just dedupe
                        }
                    }
                }
            }
        }
    }

    return dedupePreserveOrder(sys), nil
}
