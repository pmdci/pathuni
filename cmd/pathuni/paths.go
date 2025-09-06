package main

// Shared path resolution helpers to avoid duplication across commands.

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

// resolvePathuniPaths returns config-derived paths for the current context, deduped.
func resolvePathuniPaths() ([]string, error) {
    paths, err := getPathUniPaths()
    if err != nil {
        return nil, err
    }
    return dedupePreserveOrder(paths), nil
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

