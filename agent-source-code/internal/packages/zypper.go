package packages

import (
	"bufio"
	"os"
	"strings"

	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
)

// ZypperManager handles zypper package information collection.
type ZypperManager struct{ logger *logrus.Logger }

func NewZypperManager(logger *logrus.Logger) *ZypperManager {
	return &ZypperManager{logger: logger}
}

// GetPackages uses rpm for the installed inventory and zypper for available updates.
func (m *ZypperManager) GetPackages() ([]models.Package, error) {
	installedCmd, cancelInstalled := boundedCommand(collectorTimeout, "rpm", "-qa", "--qf", "%{NAME}\t%{EVR}\t%{SUMMARY}\n")
	defer cancelInstalled()
	installedCmd.Env = append(os.Environ(), "LANG=C")
	installedOutput, err := installedCmd.Output()
	if err != nil {
		return nil, commandError("rpm -qa", err)
	}

	updatesCmd, cancelUpdates := boundedCommand(networkCollectorTimeout, "zypper", "--non-interactive", "list-updates")
	defer cancelUpdates()
	updatesCmd.Env = append(os.Environ(), "LANG=C")
	updatesOutput, err := updatesCmd.Output()
	if err != nil && len(updatesOutput) == 0 && !isExitCode(err, 100) {
		return nil, commandError("zypper list-updates", err)
	}

	installed := m.parseInstalledPackages(string(installedOutput))
	return CombinePackageData(installed, m.parseUpgradablePackages(string(updatesOutput), installed)), nil
}

func (m *ZypperManager) parseInstalledPackages(output string) map[string]models.Package {
	packages := make(map[string]models.Package)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if fields := zypperTableFields(line); len(fields) >= 4 && fields[0] == "i" {
			if fields[2] != "" && fields[3] != "" && fields[2] != "Name" {
				packages[fields[2]] = models.Package{Name: fields[2], CurrentVersion: fields[3]}
			}
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 2 || strings.TrimSpace(fields[0]) == "" || strings.TrimSpace(fields[1]) == "" {
			continue
		}
		name := strings.TrimSpace(fields[0])
		pkg := models.Package{Name: name, CurrentVersion: strings.TrimSpace(fields[1])}
		if len(fields) == 3 {
			pkg.Description = strings.TrimSpace(fields[2])
		}
		packages[name] = pkg
	}
	return packages
}

func (m *ZypperManager) parseUpgradablePackages(output string, installed map[string]models.Package) []models.Package {
	var packages []models.Package
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := zypperTableFields(scanner.Text())
		if len(fields) < 6 || fields[0] != "v" {
			continue
		}
		name := fields[2]
		current, ok := installed[name]
		if !ok || name == "" || fields[4] == "" {
			continue
		}
		packages = append(packages, models.Package{
			Name: name, CurrentVersion: current.CurrentVersion,
			AvailableVersion: fields[4], NeedsUpdate: true,
			SourceRepository: fields[1],
		})
	}
	return packages
}

func zypperTableFields(line string) []string {
	if !strings.Contains(line, "|") {
		return nil
	}
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
