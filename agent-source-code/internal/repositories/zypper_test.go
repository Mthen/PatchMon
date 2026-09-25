package repositories

import (
	"os"
	"path/filepath"
	"testing"

	"patchmon-agent/internal/constants"

	"github.com/stretchr/testify/require"
)

func TestZypperParsesRepositoryFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "openSUSE.repo")
	require.NoError(t, os.WriteFile(file, []byte(`[repo-oss]
name=Main Repository
enabled=1
baseurl=https://download.opensuse.org/distribution/leap/15.6/repo/oss/

[repo-debug]
name=Debug
enabled=0
baseurl=https://download.opensuse.org/debug/`), 0600))

	repos, err := NewZypperManager(nil).parseRepoFile(file)
	require.NoError(t, err)
	require.Len(t, repos, 1)
	require.Equal(t, "repo-oss", repos[0].Name)
	require.Equal(t, constants.RepoTypeZypper, repos[0].RepoType)
	require.True(t, repos[0].IsSecure)
}
