package indra

import (
	"fmt"
)

var (
	// URL is the git URL for the repository.
	URL = ""
	// GitRef is the gitref, as in refs/heads/branchname.
	GitRef = "refs/heads/main"
	// ParentGitCommit is the commit hash of the parent HEAD.
	ParentGitCommit = "4c044ca4db090eb38d44d2b42998b1a16e87fd98"
	// BuildTime stores the time when the current binary was built.
	BuildTime = "2023-01-29T11:46:42Z"
	// SemVer lists the (latest) git tag on the release.
	SemVer = "v0.0.6"
	// PathBase is the path base returned from runtime caller.
	PathBase = "/opt/indra-labs/indra/"
	// Major is the major number from the tag.
	Major = 0
	// Minor is the minor number from the tag.
	Minor = 0
	// Patch is the patch version number from the tag.
	Patch = 6
)

// Version returns a pretty printed version information string.
func Version() string {
	return fmt.Sprint(
		"\nRepository Information\n",
		"\tGit repository: "+URL+"\n",
		"\tBranch: "+GitRef+"\n",
		"\tParentGitCommit: "+ParentGitCommit+"\n",
		"\tBuilt: "+BuildTime+"\n",
		"\tSemVer: "+SemVer+"\n",
	)
}
