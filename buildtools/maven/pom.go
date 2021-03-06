package maven

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/fossas/fossa-cli/files"
	"github.com/fossas/fossa-cli/graph"
	"github.com/fossas/fossa-cli/pkg"
)

// A Manifest represents a POM manifest file.
type Manifest struct {
	Project      xml.Name     `xml:"project"`
	Parent       Parent       `xml:"parent"`
	Modules      []string     `xml:"modules>module"`
	ArtifactID   string       `xml:"artifactId"`
	GroupID      string       `xml:"groupId"`
	Version      string       `xml:"version"`
	Description  string       `xml:"description"`
	Name         string       `xml:"name"`
	URL          string       `xml:"url"`
	Dependencies []Dependency `xml:"dependencies>dependency"`
}

type Parent struct {
	ArtifactID string `xml:"artifactId"`
	GroupID    string `xml:"groupId"`
	Version    string `xml:"version"`
}

type Dependency struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`

	// Scope is where the dependency is used, such as "test" or "runtime".
	Scope string `xml:"scope"`

	Failed bool
}

// ID returns the dependency identifier as groupId:artifactId.
func (d Dependency) ID() string {
	return d.GroupId + ":" + d.ArtifactId
}

// GraphFromTarget returns simply the list of dependencies listed within the manifest file.
func GraphFromTarget(buildTarget string) (graph.Deps, error) {
	pom, err := ResolveManifestFromBuildTarget(buildTarget)
	if err != nil {
		return graph.Deps{}, err
	}
	deps := graph.Deps{
		Direct:     depsListToImports(pom.Dependencies),
		Transitive: make(map[pkg.ID]pkg.Package),
	}

	// From just a POM file we don't know what really depends on what, so list all imports in the graph.
	for _, dep := range pom.Dependencies {
		pack := pkg.Package{
			ID: pkg.ID{
				Type:     pkg.Maven,
				Name:     dep.ID(),
				Revision: dep.Version,
			},
		}
		deps.Transitive[pack.ID] = pack
	}

	return deps, nil
}

// ResolveManifestFromBuildTarget tries to determine what buildTarget is supposed to be and then reads the POM
// manifest file pointed to by buildTarget if it is a path to such a file or module.
func ResolveManifestFromBuildTarget(buildTarget string) (*Manifest, error) {
	var pomFile string
	stat, err := os.Stat(buildTarget)
	if err != nil {
		// buildTarget is not a path.
		if strings.Count(buildTarget, ":") == 1 {
			// This is likely a module ID.
			return nil, errors.Errorf("cannot identify POM file for module %q", buildTarget)
		}
		return nil, errors.Errorf("manifest file for %q cannot be read", buildTarget)
	}
	if stat.IsDir() {
		// We have the directory and will assume it uses the standard name for the manifest file.
		pomFile = filepath.Join(buildTarget, "pom.xml")
	} else {
		// We have the manifest file but still need its directory path.
		pomFile = buildTarget
	}

	var pom Manifest
	if err := files.ReadXML(&pom, pomFile); err != nil {
		return nil, err
	}
	return &pom, nil
}
