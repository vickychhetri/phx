package pm

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Version struct {
	Major, Minor, Patch int
	Raw                 string
}

func parseVersion(v string) (Version, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version format: %s", v)
	}
	var major, minor, patch int
	_, err1 := fmt.Sscanf(parts[0], "%d", &major)
	_, err2 := fmt.Sscanf(parts[1], "%d", &minor)
	_, err3 := fmt.Sscanf(parts[2], "%d", &patch)
	if err1 != nil || err2 != nil || err3 != nil {
		return Version{}, fmt.Errorf("invalid version format: %s", v)
	}
	return Version{Major: major, Minor: minor, Patch: patch, Raw: v}, nil
}

func versionMatch(constraint, versionStr string) bool {
	if constraint == "*" || constraint == "latest" || constraint == "" {
		return true
	}
	
	if strings.HasPrefix(constraint, "^") {
		reqVer, err := parseVersion(constraint[1:])
		if err != nil {
			return false
		}
		curVer, err := parseVersion(versionStr)
		if err != nil {
			return false
		}
		if curVer.Major != reqVer.Major {
			return false
		}
		if curVer.Minor < reqVer.Minor {
			return false
		}
		if curVer.Minor == reqVer.Minor && curVer.Patch < reqVer.Patch {
			return false
		}
		return true
	}
	
	return constraint == versionStr
}

func compareVersions(v1, v2 Version) int {
	if v1.Major != v2.Major {
		return v1.Major - v2.Major
	}
	if v1.Minor != v2.Minor {
		return v1.Minor - v2.Minor
	}
	return v1.Patch - v2.Patch
}

func getRegistryURL() string {
	url := os.Getenv("PHX_REGISTRY_URL")
	return strings.TrimSuffix(url, "/")
}

func downloadFile(url string, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func getPackageIndex(registryDir string, name string) ([]byte, error) {
	remoteURL := getRegistryURL()
	if remoteURL != "" {
		url := fmt.Sprintf("%s/%s/index.json", remoteURL, name)
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			return io.ReadAll(resp.Body)
		}
	}

	indexFile := filepath.Join(registryDir, name, "index.json")
	if _, err := os.Stat(indexFile); err == nil {
		return os.ReadFile(indexFile)
	}

	return nil, fmt.Errorf("package %q not found in remote or local registry", name)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func getGitHubZipURL(urlStr string) (string, bool) {
	if strings.Contains(urlStr, "github.com/") && !strings.HasSuffix(urlStr, ".zip") {
		urlStr = strings.TrimSuffix(urlStr, ".git")
		urlStr = strings.TrimSuffix(urlStr, "/")
		return urlStr + "/archive/refs/heads/main.zip", true
	}
	return urlStr, false
}

func downloadURLToTemp(urlStr string) (string, error) {
	tempFile, err := os.CreateTemp("", "phx-pkg-*.zip")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	var resp *http.Response
	var fetchErr error

	if githubURL, isGitHub := getGitHubZipURL(urlStr); isGitHub {
		resp, fetchErr = http.Get(githubURL)
		if fetchErr != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			masterURL := strings.Replace(githubURL, "/main.zip", "/master.zip", 1)
			resp, fetchErr = http.Get(masterURL)
		}
	} else {
		resp, fetchErr = http.Get(urlStr)
	}

	if fetchErr != nil {
		return "", fetchErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download from %s: HTTP status %s", urlStr, resp.Status)
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func inspectDownloadedPackage(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == "phx.json" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			var pkgMeta struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(rc).Decode(&pkgMeta); err != nil {
				return "", fmt.Errorf("failed to parse phx.json in package: %w", err)
			}
			if pkgMeta.Name == "" {
				return "", fmt.Errorf("package phx.json is missing name field")
			}
			return pkgMeta.Name, nil
		}
	}

	return "", fmt.Errorf("no phx.json found in package zip")
}

func downloadAndInspectPackage(urlStr string) (string, string, error) {
	tempZip, err := downloadURLToTemp(urlStr)
	if err != nil {
		return "", "", err
	}

	name, err := inspectDownloadedPackage(tempZip)
	if err != nil {
		_ = os.Remove(tempZip)
		return "", "", err
	}

	return name, tempZip, nil
}

func readPhxJsonFromZip(zipPath string) ([]byte, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == "phx.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("phx.json not found in zip")
}

func resolveDependencies(registryDir string, dependencies map[string]string, resolved map[string]string) error {
	for name, constraint := range dependencies {
		if existingVersion, found := resolved[name]; found {
			if !versionMatch(constraint, existingVersion) {
				return fmt.Errorf("conflict: package %s already resolved to %s, but require %s", name, existingVersion, constraint)
			}
			continue
		}

		var subDeps map[string]string

		if strings.HasPrefix(constraint, "http://") || strings.HasPrefix(constraint, "https://") {
			resolved[name] = constraint

			localCacheDir := filepath.Join(registryDir, "cache", name)
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(constraint)))[:8]
			archivePath := filepath.Join(localCacheDir, fmt.Sprintf("%s-%s.zip", name, hash))

			if _, err := os.Stat(archivePath); os.IsNotExist(err) {
				_ = os.MkdirAll(localCacheDir, 0755)
				tempZip, err := downloadURLToTemp(constraint)
				if err != nil {
					return fmt.Errorf("failed to download dependency %s from %s: %w", name, constraint, err)
				}
				_ = copyFile(tempZip, archivePath)
				_ = os.Remove(tempZip)
			}

			subPhxJson, err := readPhxJsonFromZip(archivePath)
			if err == nil {
				var subPkg struct {
					Dependencies map[string]string `json:"dependencies"`
				}
				if json.Unmarshal(subPhxJson, &subPkg) == nil {
					subDeps = subPkg.Dependencies
				}
			}
		} else {
			data, err := getPackageIndex(registryDir, name)
			if err != nil {
				return err
			}

			var index struct {
				Name     string `json:"name"`
				Versions map[string]struct {
					Name         string            `json:"name"`
					Version      string            `json:"version"`
					Dependencies map[string]string `json:"dependencies"`
				} `json:"versions"`
			}

			if err := json.Unmarshal(data, &index); err != nil {
				return fmt.Errorf("failed to parse package metadata for %s: %w", name, err)
			}

			var bestVer string
			var bestVerObj Version
			for verStr := range index.Versions {
				if versionMatch(constraint, verStr) {
					verObj, err := parseVersion(verStr)
					if err != nil {
						continue
					}
					if bestVer == "" || compareVersions(verObj, bestVerObj) > 0 {
						bestVer = verStr
						bestVerObj = verObj
					}
				}
			}

			if bestVer == "" {
				return fmt.Errorf("no matching version found for %s with constraint %s", name, constraint)
			}

			resolved[name] = bestVer
			subDeps = index.Versions[bestVer].Dependencies
		}

		if len(subDeps) > 0 {
			if err := resolveDependencies(registryDir, subDeps, resolved); err != nil {
				return err
			}
		}
	}
	return nil
}

func Init(dir string, yes bool) error {
	phxJsonPath := filepath.Join(dir, "phx.json")
	if _, err := os.Stat(phxJsonPath); err == nil {
		return fmt.Errorf("phx.json already exists")
	}

	defaultName := filepath.Base(dir)
	name := defaultName
	version := "1.0.0"
	description := ""
	mainFile := "index.php"

	if !yes {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Package Name (%s): ", defaultName)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			name = input
		}

		fmt.Print("Version (1.0.0): ")
		input, _ = reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			version = input
		}

		fmt.Print("Description: ")
		input, _ = reader.ReadString('\n')
		description = strings.TrimSpace(input)

		fmt.Print("Main entry file (index.php): ")
		input, _ = reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			mainFile = input
		}
	}

	pkg := map[string]interface{}{
		"name":         name,
		"version":      version,
		"description":  description,
		"main":         mainFile,
		"dependencies": map[string]string{},
	}

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(phxJsonPath, data, 0644)
	if err != nil {
		return err
	}

	fmt.Println("Successfully created phx.json")
	return nil
}

func Add(dir string, pkgName string, constraint string) error {
	phxJsonPath := filepath.Join(dir, "phx.json")
	if _, err := os.Stat(phxJsonPath); os.IsNotExist(err) {
		return fmt.Errorf("phx.json not found. Run 'phx pkg init' first")
	}

	data, err := os.ReadFile(phxJsonPath)
	if err != nil {
		return err
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	registryDir := filepath.Join(home, ".phx_registry")

	deps, ok := pkg["dependencies"].(map[string]interface{})
	if !ok {
		deps = make(map[string]interface{})
		pkg["dependencies"] = deps
	}

	if strings.HasPrefix(pkgName, "http://") || strings.HasPrefix(pkgName, "https://") {
		realName, tempZip, err := downloadAndInspectPackage(pkgName)
		if err != nil {
			return fmt.Errorf("failed to add package from URL: %w", err)
		}
		defer os.Remove(tempZip)

		deps[realName] = pkgName

		localCacheDir := filepath.Join(registryDir, "cache", realName)
		if err := os.MkdirAll(localCacheDir, 0755); err == nil {
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(pkgName)))[:8]
			cachedZip := filepath.Join(localCacheDir, fmt.Sprintf("%s-%s.zip", realName, hash))
			_ = copyFile(tempZip, cachedZip)
		}
		pkgName = realName
		constraint = pkgName
	} else {
		indexFile := filepath.Join(registryDir, pkgName, "index.json")
		if _, err := os.Stat(indexFile); os.IsNotExist(err) {
			return fmt.Errorf("package %q not found in registry", pkgName)
		}

		if constraint == "" {
			idxData, err := os.ReadFile(indexFile)
			if err != nil {
				return err
			}
			var index struct {
				Versions map[string]interface{} `json:"versions"`
			}
			if err := json.Unmarshal(idxData, &index); err == nil && len(index.Versions) > 0 {
				var latestVersion string
				var latestVersionObj Version
				for verStr := range index.Versions {
					verObj, err := parseVersion(verStr)
					if err != nil {
						continue
					}
					if latestVersion == "" || compareVersions(verObj, latestVersionObj) > 0 {
						latestVersion = verStr
						latestVersionObj = verObj
					}
				}
				if latestVersion != "" {
					constraint = "^" + latestVersion
				}
			}
			if constraint == "" {
				constraint = "*"
			}
		}

		deps[pkgName] = constraint
	}

	newData, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(phxJsonPath, newData, 0644); err != nil {
		return err
	}

	fmt.Printf("Added dependency %s to phx.json\n", pkgName)
	return Install(dir, true)
}

func Install(dir string, forceResolve bool) error {
	phxJsonPath := filepath.Join(dir, "phx.json")
	if _, err := os.Stat(phxJsonPath); os.IsNotExist(err) {
		return fmt.Errorf("phx.json not found. Run 'phx pkg init' first")
	}

	data, err := os.ReadFile(phxJsonPath)
	if err != nil {
		return err
	}

	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	registryDir := filepath.Join(home, ".phx_registry")

	phxLockPath := filepath.Join(dir, "phx.lock")
	resolved := make(map[string]string)
	lockResolved := make(map[string]string)

	if !forceResolve {
		if _, err := os.Stat(phxLockPath); err == nil {
			fmt.Println("Installing dependencies from phx.lock...")
			lockData, err := os.ReadFile(phxLockPath)
			if err == nil {
				var lockfile struct {
					Dependencies map[string]struct {
						Version  string `json:"version"`
						Resolved string `json:"resolved"`
					} `json:"dependencies"`
				}
				if json.Unmarshal(lockData, &lockfile) == nil {
					for name, dep := range lockfile.Dependencies {
						resolved[name] = dep.Version
						if dep.Resolved != "" {
							lockResolved[name] = dep.Resolved
						}
					}
				}
			}
		}
	}

	if len(resolved) == 0 {
		fmt.Println("Resolving dependencies from phx.json...")
		if len(pkg.Dependencies) > 0 {
			if err := resolveDependencies(registryDir, pkg.Dependencies, resolved); err != nil {
				return err
			}
		}
	}

	phxPkgsDir := filepath.Join(dir, "phx_packages")
	if err := os.RemoveAll(phxPkgsDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean phx_packages directory: %w", err)
	}
	if err := os.MkdirAll(phxPkgsDir, 0755); err != nil {
		return fmt.Errorf("failed to create phx_packages directory: %w", err)
	}

	type LockDependency struct {
		Version  string `json:"version"`
		Resolved string `json:"resolved"`
	}
	lockfileData := struct {
		Dependencies map[string]LockDependency `json:"dependencies"`
	}{
		Dependencies: make(map[string]LockDependency),
	}

	var autoloaderLines []string

	for name, ver := range resolved {
		localCacheDir := filepath.Join(registryDir, "cache", name)
		if err := os.MkdirAll(localCacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}

		var archivePath string
		resolvedPathFromLock := lockResolved[name]
		if resolvedPathFromLock != "" {
			if _, err := os.Stat(resolvedPathFromLock); err == nil {
				archivePath = resolvedPathFromLock
			}
		}

		if archivePath == "" {
			if strings.HasPrefix(ver, "http://") || strings.HasPrefix(ver, "https://") {
				hash := fmt.Sprintf("%x", sha256.Sum256([]byte(ver)))[:8]
				archivePath = filepath.Join(localCacheDir, fmt.Sprintf("%s-%s.zip", name, hash))

				if _, err := os.Stat(archivePath); os.IsNotExist(err) {
					fmt.Printf("Downloading %s (%s)...\n", name, ver)
					tempZip, err := downloadURLToTemp(ver)
					if err != nil {
						return fmt.Errorf("failed to download package %s: %w", name, err)
					}
					_ = copyFile(tempZip, archivePath)
					_ = os.Remove(tempZip)
				}
			} else {
				archivePath = filepath.Join(localCacheDir, fmt.Sprintf("%s-%s.zip", name, ver))

				if _, err := os.Stat(archivePath); os.IsNotExist(err) {
					localRegPath := filepath.Join(registryDir, name, fmt.Sprintf("%s-%s.zip", name, ver))
					if _, errLocal := os.Stat(localRegPath); errLocal == nil {
						archivePath = localRegPath
					} else {
						remoteURL := getRegistryURL()
						if remoteURL == "" {
							return fmt.Errorf("package archive %q not found locally and no remote registry configured", name)
						}
						downloadURL := fmt.Sprintf("%s/%s/%s-%s.zip", remoteURL, name, name, ver)
						fmt.Printf("Downloading %s (%s)...\n", name, ver)
						if err := downloadFile(downloadURL, archivePath); err != nil {
							return fmt.Errorf("failed to download package %s: %w", name, err)
						}
					}
				}
			}
		}

		destDir := filepath.Join(phxPkgsDir, name)
		fmt.Printf("Installing %s (%s)...\n", name, ver)
		if err := unzip(archivePath, destDir); err != nil {
			return fmt.Errorf("failed to extract package %s: %w", name, err)
		}

		lockfileData.Dependencies[name] = LockDependency{
			Version:  ver,
			Resolved: archivePath,
		}

		subPhxJsonPath := filepath.Join(destDir, "phx.json")
		mainFile := "index.php"
		if subData, err := os.ReadFile(subPhxJsonPath); err == nil {
			var subPkg struct {
				Main string `json:"main"`
			}
			if json.Unmarshal(subData, &subPkg) == nil && subPkg.Main != "" {
				mainFile = subPkg.Main
			}
		}
		
		autoloaderLines = append(autoloaderLines, fmt.Sprintf("include \"phx_packages/%s/%s\";", name, mainFile))
	}

	autoloadPath := filepath.Join(phxPkgsDir, "autoload.php")
	var autoloadContent strings.Builder
	autoloadContent.WriteString("<?php\n")
	autoloadContent.WriteString("// PHX Autoloader - generated by phx package manager\n")
	for _, line := range autoloaderLines {
		autoloadContent.WriteString(line + "\n")
	}
	autoloadContent.WriteString("?>\n")

	if err := os.WriteFile(autoloadPath, []byte(autoloadContent.String()), 0644); err != nil {
		return fmt.Errorf("failed to write autoload.php: %w", err)
	}

	lockBytes, err := json.MarshalIndent(lockfileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal phx.lock: %w", err)
	}

	if err := os.WriteFile(phxLockPath, lockBytes, 0644); err != nil {
		return fmt.Errorf("failed to write phx.lock: %w", err)
	}

	fmt.Println("Package installation complete.")
	return nil
}

func Update(dir string) error {
	return Install(dir, true)
}

func Publish(dir string) error {
	phxJsonPath := filepath.Join(dir, "phx.json")
	if _, err := os.Stat(phxJsonPath); os.IsNotExist(err) {
		return fmt.Errorf("phx.json not found. Run 'phx pkg init' first")
	}

	data, err := os.ReadFile(phxJsonPath)
	if err != nil {
		return fmt.Errorf("failed to read phx.json: %w", err)
	}

	var pkg struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		Description  string            `json:"description"`
		Main         string            `json:"main"`
		Dependencies map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("failed to parse phx.json: %w", err)
	}

	if pkg.Name == "" || pkg.Version == "" {
		return fmt.Errorf("package name and version must be defined in phx.json")
	}

	if _, err := parseVersion(pkg.Version); err != nil {
		return fmt.Errorf("invalid package version format %q (must be x.y.z): %w", pkg.Version, err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	registryDir := filepath.Join(home, ".phx_registry")
	pkgRegistryDir := filepath.Join(registryDir, pkg.Name)

	if err := os.MkdirAll(pkgRegistryDir, 0755); err != nil {
		return fmt.Errorf("failed to create registry directory for package: %w", err)
	}

	indexFile := filepath.Join(pkgRegistryDir, "index.json")
	var index struct {
		Name     string `json:"name"`
		Versions map[string]struct {
			Name         string            `json:"name"`
			Version      string            `json:"version"`
			Description  string            `json:"description"`
			Main         string            `json:"main"`
			Dependencies map[string]string `json:"dependencies"`
			Archive      string            `json:"archive"`
		} `json:"versions"`
	}
	index.Versions = make(map[string]struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		Description  string            `json:"description"`
		Main         string            `json:"main"`
		Dependencies map[string]string `json:"dependencies"`
		Archive      string            `json:"archive"`
	})

	if _, err := os.Stat(indexFile); err == nil {
		idxData, err := os.ReadFile(indexFile)
		if err == nil {
			_ = json.Unmarshal(idxData, &index)
		}
	}

	index.Name = pkg.Name
	if _, exists := index.Versions[pkg.Version]; exists {
		return fmt.Errorf("version %s is already published", pkg.Version)
	}

	archivePath := filepath.Join(pkgRegistryDir, fmt.Sprintf("%s-%s.zip", pkg.Name, pkg.Version))
	fmt.Printf("Archiving package files to %s...\n", archivePath)
	if err := zipDir(dir, archivePath); err != nil {
		return fmt.Errorf("failed to create package archive: %w", err)
	}

	index.Versions[pkg.Version] = struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		Description  string            `json:"description"`
		Main         string            `json:"main"`
		Dependencies map[string]string `json:"dependencies"`
		Archive      string            `json:"archive"`
	}{
		Name:         pkg.Name,
		Version:      pkg.Version,
		Description:  pkg.Description,
		Main:         pkg.Main,
		Dependencies: pkg.Dependencies,
		Archive:      archivePath,
	}

	newIdxData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry index: %w", err)
	}

	if err := os.WriteFile(indexFile, newIdxData, 0644); err != nil {
		return fmt.Errorf("failed to write registry index: %w", err)
	}

	globalRegistryFile := filepath.Join(registryDir, "registry.json")
	globalRegistry := make(map[string]string)
	if globalRegistryData, err := os.ReadFile(globalRegistryFile); err == nil {
		_ = json.Unmarshal(globalRegistryData, &globalRegistry)
	}
	globalRegistry[pkg.Name] = pkg.Description

	if newGlobalData, err := json.MarshalIndent(globalRegistry, "", "  "); err == nil {
		_ = os.WriteFile(globalRegistryFile, newGlobalData, 0644)
	}

	fmt.Printf("Successfully published package %s@%s to registry\n", pkg.Name, pkg.Version)
	return nil
}

func Search(query string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	registryDir := filepath.Join(home, ".phx_registry")

	packages := make(map[string]string)
	remoteURL := getRegistryURL()
	if remoteURL != "" {
		url := fmt.Sprintf("%s/registry.json", remoteURL)
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var remoteRegistry map[string]string
			if data, err := io.ReadAll(resp.Body); err == nil {
				_ = json.Unmarshal(data, &remoteRegistry)
				for k, v := range remoteRegistry {
					packages[k] = v
				}
			}
		}
	}

	globalRegistryFile := filepath.Join(registryDir, "registry.json")
	if data, err := os.ReadFile(globalRegistryFile); err == nil {
		var localRegistry map[string]string
		if json.Unmarshal(data, &localRegistry) == nil {
			for k, v := range localRegistry {
				packages[k] = v
			}
		}
	}

	if len(packages) == 0 {
		if _, err := os.Stat(registryDir); err == nil {
			files, _ := os.ReadDir(registryDir)
			for _, f := range files {
				if f.IsDir() && f.Name() != "cache" {
					packages[f.Name()] = ""
				}
			}
		}
	}

	fmt.Println("Search results:")
	fmt.Printf("%-20s %s\n", "Package Name", "Description")
	fmt.Println(strings.Repeat("-", 60))

	found := false
	for pkgName, desc := range packages {
		if query != "" && !strings.Contains(strings.ToLower(pkgName), strings.ToLower(query)) && !strings.Contains(strings.ToLower(desc), strings.ToLower(query)) {
			continue
		}
		fmt.Printf("%-20s %s\n", pkgName, desc)
		found = true
	}

	if !found {
		fmt.Println("No packages matched the query.")
	}
	return nil
}

func zipDir(srcDir, destZip string) error {
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		parts := strings.Split(rel, string(filepath.Separator))
		for _, part := range parts {
			if part == "phx_packages" || part == ".git" || part == "node_modules" {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func unzip(srcZip, destDir string) error {
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer r.Close()

	commonPrefix := ""
	if len(r.File) > 0 {
		firstFile := r.File[0].Name
		idx := strings.Index(firstFile, "/")
		if idx != -1 {
			prefix := firstFile[:idx+1]
			hasPrefix := true
			for _, f := range r.File {
				if !strings.HasPrefix(f.Name, prefix) {
					hasPrefix = false
					break
				}
			}
			if hasPrefix {
				commonPrefix = prefix
			}
		}
	}

	for _, f := range r.File {
		relName := f.Name
		if commonPrefix != "" {
			relName = strings.TrimPrefix(relName, commonPrefix)
		}
		if relName == "" {
			continue
		}

		fpath := filepath.Join(destDir, relName)

		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(filepath.Separator)) && fpath != destDir {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
