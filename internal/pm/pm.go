package pm

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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

func resolveDependencies(registryDir string, dependencies map[string]string, resolved map[string]string) error {
	for name, constraint := range dependencies {
		if existingVersion, found := resolved[name]; found {
			if !versionMatch(constraint, existingVersion) {
				return fmt.Errorf("conflict: package %s already resolved to %s, but require %s", name, existingVersion, constraint)
			}
			continue
		}

		indexFile := filepath.Join(registryDir, name, "index.json")
		if _, err := os.Stat(indexFile); os.IsNotExist(err) {
			return fmt.Errorf("package %s not found in registry", name)
		}

		data, err := os.ReadFile(indexFile)
		if err != nil {
			return fmt.Errorf("failed to read package metadata for %s: %w", name, err)
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

		pkgVerData := index.Versions[bestVer]
		if len(pkgVerData.Dependencies) > 0 {
			if err := resolveDependencies(registryDir, pkgVerData.Dependencies, resolved); err != nil {
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

	deps, ok := pkg["dependencies"].(map[string]interface{})
	if !ok {
		deps = make(map[string]interface{})
		pkg["dependencies"] = deps
	}

	deps[pkgName] = constraint

	newData, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(phxJsonPath, newData, 0644); err != nil {
		return err
	}

	fmt.Printf("Added dependency %s: %s to phx.json\n", pkgName, constraint)
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

	if !forceResolve {
		if _, err := os.Stat(phxLockPath); err == nil {
			fmt.Println("Installing dependencies from phx.lock...")
			lockData, err := os.ReadFile(phxLockPath)
			if err == nil {
				var lockfile struct {
					Dependencies map[string]struct {
						Version string `json:"version"`
					} `json:"dependencies"`
				}
				if json.Unmarshal(lockData, &lockfile) == nil {
					for name, dep := range lockfile.Dependencies {
						resolved[name] = dep.Version
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
		archivePath := filepath.Join(registryDir, name, fmt.Sprintf("%s-%s.zip", name, ver))
		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			return fmt.Errorf("package archive not found in registry: %s", archivePath)
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

	fmt.Printf("Successfully published package %s@%s to registry\n", pkg.Name, pkg.Version)
	return nil
}

func Search(query string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	registryDir := filepath.Join(home, ".phx_registry")
	if _, err := os.Stat(registryDir); os.IsNotExist(err) {
		fmt.Println("No packages published yet.")
		return nil
	}

	files, err := os.ReadDir(registryDir)
	if err != nil {
		return err
	}

	fmt.Println("Search results:")
	fmt.Printf("%-20s %-10s %s\n", "Package Name", "Version", "Description")
	fmt.Println(strings.Repeat("-", 60))

	found := false
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		pkgName := f.Name()
		if query != "" && !strings.Contains(strings.ToLower(pkgName), strings.ToLower(query)) {
			continue
		}

		indexFile := filepath.Join(registryDir, pkgName, "index.json")
		if _, err := os.Stat(indexFile); err != nil {
			continue
		}

		idxData, err := os.ReadFile(indexFile)
		if err != nil {
			continue
		}

		var index struct {
			Versions map[string]struct {
				Version     string `json:"version"`
				Description string `json:"description"`
			} `json:"versions"`
		}

		if err := json.Unmarshal(idxData, &index); err == nil && len(index.Versions) > 0 {
			var latestVer string
			var latestVerObj Version
			for verStr := range index.Versions {
				verObj, err := parseVersion(verStr)
				if err != nil {
					continue
				}
				if latestVer == "" || compareVersions(verObj, latestVerObj) > 0 {
					latestVer = verStr
					latestVerObj = verObj
				}
			}

			if latestVer != "" {
				desc := index.Versions[latestVer].Description
				fmt.Printf("%-20s %-10s %s\n", pkgName, latestVer, desc)
				found = true
			}
		}
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

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

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
