package repositories

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"patchmon-agent/internal/constants"
	"patchmon-agent/pkg/models"

	"github.com/sirupsen/logrus"
)

type ZypperManager struct{ logger *logrus.Logger }

func NewZypperManager(logger *logrus.Logger) *ZypperManager {
	return &ZypperManager{logger: logger}
}

func (m *ZypperManager) GetRepositories() ([]models.Repository, error) {
	files, err := filepath.Glob("/etc/zypp/repos.d/*.repo")
	if err != nil {
		return nil, err
	}
	var repositories []models.Repository
	for _, file := range files {
		repos, err := m.parseRepoFile(file)
		if err != nil {
			m.logger.WithError(err).WithField("file", file).Warn("Failed to parse zypper repository file")
			continue
		}
		repositories = append(repositories, repos...)
	}
	return repositories, nil
}

type zypperRepoEntry struct {
	id, name, baseURL string
	enabled           bool
}

func (m *ZypperManager) parseRepoFile(filename string) ([]models.Repository, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var repositories []models.Repository
	var entry *zypperRepoEntry
	save := func() {
		if entry == nil || !entry.enabled || !isValidRepoURL(entry.baseURL) {
			return
		}
		repositories = append(repositories, models.Repository{
			Name: entry.id, URL: entry.baseURL, Distribution: entry.name,
			RepoType: constants.RepoTypeZypper, IsEnabled: true,
			IsSecure: strings.HasPrefix(entry.baseURL, "https://"),
		})
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			save()
			entry = &zypperRepoEntry{id: strings.Trim(line, "[]"), enabled: true}
			continue
		}
		if entry == nil {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			entry.name = strings.TrimSpace(value)
		case "baseurl":
			entry.baseURL = strings.TrimSpace(value)
		case "enabled":
			value = strings.ToLower(strings.TrimSpace(value))
			entry.enabled = value != "0" && value != "false"
		}
	}
	save()
	return repositories, scanner.Err()
}
