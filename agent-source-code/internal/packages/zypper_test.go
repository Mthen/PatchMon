package packages

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZypperParsesInstalledAndUpdates(t *testing.T) {
	m := NewZypperManager(nil)
	installed := m.parseInstalledPackages(`bash	5.2.21-1.1	GNU Bourne-Again Shell
curl	8.12.1-1.1	Tool`)
	require.Equal(t, "5.2.21-1.1", installed["bash"].CurrentVersion)

	updates := m.parseUpgradablePackages(`Loading repository data...
Reading installed packages...
S | Repository | Name | Current Version | Available Version | Arch
--+------------+------+-----------------+-------------------+------
v | repo-oss   | bash | 5.2.21-1.1      | 5.2.21-1.2        | x86_64
v | repo-oss   | curl | 8.12.1-1.1      | 8.12.1-1.2        | x86_64`, installed)
	require.Len(t, updates, 2)
	require.Equal(t, "bash", updates[0].Name)
	require.Equal(t, "5.2.21-1.2", updates[0].AvailableVersion)
}
