package health

import "fmt"

// ImportMarkdownForUser loads mddata into the shared DB for one user (replace).
func ImportMarkdownForUser(dataRoot, mdDir, username string) (map[string]int, error) {
	db, err := OpenSharedDB(dataRoot)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	st, err := NewStore(db, dataRoot, username)
	if err != nil {
		return nil, err
	}
	w, f, d, err := ImportFromMDDir(mdDir)
	if err != nil {
		return nil, err
	}
	if len(w) == 0 && len(f) == 0 && len(d) == 0 {
		return nil, fmt.Errorf("no data found in %s", mdDir)
	}
	if err := st.ReplaceImported(w, f, d); err != nil {
		return nil, err
	}
	return map[string]int{
		"weights": len(w),
		"foods":   len(f),
		"days":    len(d),
	}, nil
}
