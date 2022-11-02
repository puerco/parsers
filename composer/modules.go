// SPDX-License-Identifier: Apache-2.0

package composer

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/opensbom-generator/parsers/meta"
)

func (m *Composer) getRootProjectInfo(path string) (meta.Package, error) {
	if err := m.buildCmd(projectInfoCmd, path); err != nil {
		return meta.Package{}, err
	}

	buffer := new(bytes.Buffer)
	if err := m.command.Execute(buffer); err != nil {
		return meta.Package{}, err
	}
	defer buffer.Reset()

	var projectInfo ProjectInfo

	err := json.NewDecoder(buffer).Decode(&projectInfo)
	if err != nil {
		return meta.Package{}, err
	}
	if projectInfo.Name == "" {
		return meta.Package{}, errRootProject
	}

	module := convertProjectInfoToModule(projectInfo, path)
	return module, nil
}

func convertProjectInfoToModule(project ProjectInfo, path string) meta.Package {
	version := normalizePackageVersion(project.Versions[0])
	packageURL := genComposerURL(project.Name)

	if packageURL == "" {
		composerJSON, _ := getComposerJSONFileData()
		packageURL = composerJSON.Homepage
	}

	packageDownloadLocation := rootPackageDownloadLocation(packageURL)

	checkSumValue := readCheckSum(packageURL)
	name := getName(project.Name)
	supplier := rootProjectSupplier(name)

	module := meta.Package{
		Name:       name,
		Version:    version,
		Root:       true,
		PackageURL: packageURL,
		Checksum: meta.Checksum{
			Algorithm: meta.HashAlgoSHA1,
			Value:     checkSumValue,
		},
		PackageDownloadLocation: packageDownloadLocation,
		Supplier:                supplier,
	}

	licensePkg, err := helper.GetLicenses(path)
	if err == nil {
		module.LicenseDeclared = helper.BuildLicenseDeclared(licensePkg.ID)
		module.LicenseConcluded = helper.BuildLicenseConcluded(licensePkg.ID)
		module.Copyright = helper.GetCopyright(licensePkg.ExtractedText)
		module.CommentsLicense = licensePkg.Comments
	}

	return module
}

func rootPackageDownloadLocation(defaultValue string) string {
	packageJSON, _ := getPackageJSONFileData()
	packageDownloadLocation := packageJSON.Repository.URL

	if packageDownloadLocation == "" {
		packageDownloadLocation = defaultValue
	}

	hasProtocol := strings.Contains(packageDownloadLocation, "http")
	isGithub := strings.Contains(packageDownloadLocation, "github.com/")
	hasGitSuffix := strings.Contains(packageDownloadLocation, ".git")

	if !hasProtocol {
		packageDownloadLocation = "https://" + packageDownloadLocation
	}
	if isGithub && !hasGitSuffix {
		packageDownloadLocation += ".git"
	}

	return packageDownloadLocation
}

func rootProjectSupplier(projectName string) meta.Supplier {
	composerJSON, _ := getComposerJSONFileData()
	if len(composerJSON.Authors) > 0 {
		author := composerJSON.Authors[0]
		return meta.Supplier{
			Name:  author.Name,
			Email: author.Email,
			Type:  meta.Person,
		}
	}

	return meta.Supplier{
		Name:  projectName,
		Email: "",
	}
}

func (m *Composer) getTreeListFromComposerShowTree(path string) (TreeList, error) {
	if err := m.buildCmd(ShowModulesCmd, path); err != nil {
		return TreeList{}, err
	}

	buffer := new(bytes.Buffer)
	if err := m.command.Execute(buffer); err != nil {
		return TreeList{}, err
	}
	defer buffer.Reset()

	var tree TreeList
	err := json.NewDecoder(buffer).Decode(&tree)
	if err != nil {
		return TreeList{}, err
	}

	return tree, nil
}

func addTreeComponentsToModule(treeComponent TreeComponent, modules []meta.Package) bool { //nolint: unparam
	moduleMap := map[string]meta.Package{}
	moduleIndex := map[string]int{}
	for idx, module := range modules {
		moduleMap[module.Name] = module
		moduleIndex[module.Name] = idx
	}

	rootLevelName := getName(treeComponent.Name)
	_, ok := moduleMap[rootLevelName]
	if !ok {
		return false
	}

	requires := treeComponent.Requires

	if requires == nil {
		return false
	}

	if len(requires) == 0 {
		return false
	}

	for _, subTreeComponent := range requires {
		childLevelName := getName(subTreeComponent.Name)
		childLevelModule, ok := moduleMap[childLevelName]
		if !ok {
			continue
		}

		addSubModuleToAModule(modules, moduleIndex[rootLevelName], childLevelModule)
		addTreeComponentsToModule(subTreeComponent, modules)
	}

	return true
}

func addSubModuleToAModule(modules []meta.Package, moduleIndex int, subModule meta.Package) {
	modules[moduleIndex].Packages[subModule.Name] = &meta.Package{
		Name:             subModule.Name,
		Version:          subModule.Version,
		Path:             subModule.Path,
		LocalPath:        subModule.LocalPath,
		Supplier:         subModule.Supplier,
		PackageURL:       subModule.PackageURL,
		Checksum:         subModule.Checksum,
		PackageHomePage:  subModule.PackageHomePage,
		LicenseConcluded: subModule.LicenseConcluded,
		LicenseDeclared:  subModule.LicenseDeclared,
		CommentsLicense:  subModule.CommentsLicense,
		OtherLicense:     subModule.OtherLicense,
		Copyright:        subModule.Copyright,
		PackageComment:   subModule.PackageComment,
		Root:             subModule.Root,
	}
}

func getComposerLockFileData() (LockFile, error) {
	raw, err := os.ReadFile(ComposerLockFileName)
	if err != nil {
		return LockFile{}, err
	}

	var fileData LockFile
	err = json.Unmarshal(raw, &fileData)
	if err != nil {
		return LockFile{}, err
	}

	return fileData, nil
}

func getComposerJSONFileData() (JSONObject, error) {
	raw, err := os.ReadFile(ComposerJSONFileName)
	if err != nil {
		return JSONObject{}, err
	}

	var fileData JSONObject
	err = json.Unmarshal(raw, &fileData)
	if err != nil {
		return JSONObject{}, err
	}

	return fileData, nil
}

func getPackageJSONFileData() (PackageJSONObject, error) {
	raw, err := os.ReadFile(PackageJSON)
	if err != nil {
		return PackageJSONObject{}, err
	}

	var fileData PackageJSONObject
	err = json.Unmarshal(raw, &fileData)
	if err != nil {
		return PackageJSONObject{}, err
	}

	return fileData, nil
}

func (m *Composer) getModulesFromComposerLockFile(path string) ([]meta.Package, error) {
	modules := make([]meta.Package, 0)

	info, err := getComposerLockFileData()
	if err != nil {
		return nil, err
	}

	mainMod, err := m.getRootProjectInfo(path)
	if err != nil {
		return nil, err
	}

	modules = append(modules, mainMod)

	if len(info.Packages) > 0 {
		for _, pckg := range info.Packages {
			mod := convertLockPackageToModule(pckg)
			modules = append(modules, mod)
		}
	}

	if len(info.PackagesDev) > 0 {
		for _, pckg := range info.PackagesDev {
			mod := convertLockPackageToModule(pckg)
			modules = append(modules, mod)
		}
	}

	return modules, nil
}

func convertLockPackageToModule(dep LockPackage) meta.Package {
	module := meta.Package{
		Version:                 normalizePackageVersion(dep.Version),
		Name:                    getName(dep.Name),
		Root:                    false,
		PackageURL:              genURLFromComposerPackage(dep),
		PackageDownloadLocation: dep.Source.URL,
		Checksum: meta.Checksum{
			Algorithm: meta.HashAlgoSHA1,
			Value:     getCheckSumValue(dep),
		},
		Supplier:  getAuthorFromComposerLockFileDep(dep),
		LocalPath: getLocalPath(dep),
		Packages:  map[string]*meta.Package{},
	}
	path := getLocalPath(dep)
	licensePkg, err := helper.GetLicenses(path)
	if err == nil {
		module.LicenseDeclared = helper.BuildLicenseDeclared(licensePkg.ID)
		module.LicenseConcluded = helper.BuildLicenseConcluded(licensePkg.ID)
		module.Copyright = helper.GetCopyright(licensePkg.ExtractedText)
		module.CommentsLicense = licensePkg.Comments
	} else if len(dep.License) > 0 {
		licenseValue := dep.License[0]
		module.LicenseDeclared = licenseValue
		module.LicenseConcluded = licenseValue
	}

	return module
}

func getAuthorFromComposerLockFileDep(dep LockPackage) meta.Supplier {
	authors := dep.Authors
	if len(authors) == 0 {
		return meta.Supplier{
			Name: getName(dep.Name),
			Type: meta.Organization,
		}
	}
	author := authors[0]
	pckAuthor := meta.Supplier{
		Name:  author.Name,
		Email: author.Email,
		Type:  meta.Person,
	}

	if pckAuthor.Email == "" {
		pckAuthor.Type = meta.Organization
	}

	return pckAuthor
}

func getName(moduleName string) string {
	groupNames := strings.Split(moduleName, "/")

	if len(groupNames) > 1 {
		return groupNames[1]
	}
	return groupNames[0]
}

func genURLFromComposerPackage(dep LockPackage) string {
	homePage := dep.Homepage
	if homePage != "" {
		return removeURLProtocol(homePage)
	}

	gitURL := removeURLProtocol(dep.Source.URL)
	gitURL = strings.ReplaceAll(gitURL, ".git", "")
	if gitURL != "" {
		return gitURL
	}

	createdURL := genComposerURL(dep.Name)
	return createdURL
}

func genComposerURL(name string) string {
	return "github.com/" + name
}

func normalizePackageVersion(version string) string {
	parts := strings.Split(version, "v")

	if parts[0] != "" {
		return version
	}

	if len(parts) > 1 {
		return parts[1]
	}

	return parts[0]
}

func getCheckSumValue(module LockPackage) string {
	value := module.Dist.Shasum
	if value != "" {
		return value
	}

	return readCheckSum(genURLFromComposerPackage(module))
}

func readCheckSum(content string) string {
	if content == "" {
		return ""
	}
	h := sha1.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

func getLocalPath(module LockPackage) string {
	path := "./vendor/" + module.Name
	return path
}
