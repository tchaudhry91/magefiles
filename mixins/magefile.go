package mixins

import (
	"fmt"
	"os"
	"path/filepath"

	"get.porter.sh/magefiles/porter"
	"get.porter.sh/magefiles/releases"
	"github.com/carolynvs/magex/mgx"
	"github.com/carolynvs/magex/shx"
	"github.com/carolynvs/magex/xplat"
	"github.com/magefile/mage/mg"
)

type Magefile struct {
	Pkg       string
	MixinName string
	BinDir    string
}

// Create a magefile helper for a mixin
func NewMagefile(pkg, mixinName, binDir string) Magefile {
	return Magefile{Pkg: pkg, MixinName: mixinName, BinDir: binDir}
}

var must = shx.CommandBuilder{StopOnError: true}

// Build the mixin
func (m Magefile) Build() {
	must.RunV("go", "mod", "tidy")
	releases.BuildAll(m.Pkg, m.MixinName, m.BinDir)
}

// Cross-compile the mixin before a release
func (m Magefile) XBuildAll() {
	releases.XBuildAll(m.Pkg, m.MixinName, m.BinDir)
	releases.PrepareMixinForPublish(m.MixinName)
}

// Run unit tests
func (m Magefile) TestUnit() {
	v := ""
	if mg.Verbose() {
		v = "-v"
	}
	must.Command("go", "test", v, "./pkg/...").CollapseArgs().RunV()
}

// Run all tests
func (m Magefile) Test() {
	m.TestUnit()

	// Check that we can call `mixin version`
	m.Build()
	must.RunV(filepath.Join(m.BinDir, m.MixinName+xplat.FileExt()), "version")
}

// Publish the mixin and its mixin feed
func (m Magefile) Publish() {
	mg.SerialDeps(m.PublishBinaries, m.PublishMixinFeed)
}

// Publish binaries to a github release
// Requires PORTER_RELEASE_REPOSITORY to be set to github.com/USERNAME/REPO
func (m Magefile) PublishBinaries() {
	mg.SerialDeps(porter.UseBinForPorterHome, porter.EnsurePorter)
	releases.PrepareMixinForPublish(m.MixinName)
	releases.PublishMixin(m.MixinName)
}

// Publish a mixin feed
// Requires PORTER_PACKAGES_REMOTE to be set to git@github.com:USERNAME/REPO.git
func (m Magefile) PublishMixinFeed() {
	mg.SerialDeps(porter.UseBinForPorterHome, porter.EnsurePorter)
	releases.PublishMixinFeed(m.MixinName)
}

// Test out publish locally, with your github forks
// Assumes that you forked and kept the repository name unchanged.
func (m Magefile) TestPublish(username string) {
	mixinRepo := fmt.Sprintf("github.com/%s/%s-mixin", username, m.MixinName)
	pkgRepo := fmt.Sprintf("https://github.com/%s/packages.git", username)
	fmt.Printf("Publishing a release to %s and committing a mixin feed to %s\n", mixinRepo, pkgRepo)
	fmt.Printf("If you use different repository names, set %s and %s then call mage Publish instead.\n", releases.ReleaseRepository, releases.PackagesRemote)
	os.Setenv(releases.ReleaseRepository, mixinRepo)
	os.Setenv(releases.PackagesRemote, pkgRepo)

	m.Publish()
}

// Install the mixin
func (m Magefile) Install() {
	porterHome := porter.GetPorterHome()
	fmt.Printf("Installing the %s mixin into %s\n", m.MixinName, porterHome)

	os.MkdirAll(filepath.Join(porterHome, "mixins", m.MixinName, "runtimes"), 0770)
	mgx.Must(shx.Copy(filepath.Join(m.BinDir, m.MixinName+xplat.FileExt()), filepath.Join(porterHome, "mixins", m.MixinName)))
	mgx.Must(shx.Copy(filepath.Join(m.BinDir, "runtimes", m.MixinName+"-runtime"+xplat.FileExt()), filepath.Join(porterHome, "mixins", m.MixinName, "runtimes")))
}

// Remove generated build files
func (m Magefile) Clean() {
	os.RemoveAll("bin")
}
