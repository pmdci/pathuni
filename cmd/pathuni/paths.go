package main

// Shared path resolution helpers to avoid duplication across commands.

import (
    "os"
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
