package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/slucheninov/gmr/internal/ai"
	"github.com/slucheninov/gmr/internal/git"
	"github.com/slucheninov/gmr/internal/platform"
	"github.com/slucheninov/gmr/internal/release"
	"github.com/slucheninov/gmr/internal/ui"
)

const deployHelp = `gmr deploy — cut the next release tag and publish it

Usage: gmr deploy [options] [tag]

Looks at the commits since the previous semver tag, asks AI to choose a
version bump (patch/minor/major) and write release notes, creates and pushes
an annotated tag, then creates a GitHub Release / GitLab Release from it.

With no existing semver tag, the first release is v0.0.1 (or "<GMR_TAG_PREFIX>0.0.1").

If [tag] is given explicitly (e.g. v1.2.3), it is used as-is, overriding any
bump flag or the AI's choice. It must look like <prefix>MAJOR.MINOR.PATCH.

Options:
  -h, --help        Show this help
      --patch       Force a patch bump, overriding the AI's choice
      --minor       Force a minor bump, overriding the AI's choice
      --major       Force a major bump, overriding the AI's choice
      --no-release  Create and push the tag but skip the GitHub/GitLab Release
  -y, --yes         Do not ask for confirmation
`

var errDeployShowHelp = errors.New("show deploy help")

type deployOptions struct {
	bump        release.Bump
	bumpForced  bool
	noRelease   bool
	yes         bool
	explicitTag string
}

// parseDeployArgs parses the arguments to `gmr deploy`.
func parseDeployArgs(args []string) (deployOptions, error) {
	var o deployOptions
	var positional []string
	bumpSet := false

	setBump := func(b release.Bump) error {
		if bumpSet {
			return errors.New("only one of --patch, --minor, --major may be given")
		}
		o.bump = b
		o.bumpForced = true
		bumpSet = true
		return nil
	}

	for _, a := range args {
		switch a {
		case "-h", "--help":
			return deployOptions{}, errDeployShowHelp
		case "--patch":
			if err := setBump(release.Patch); err != nil {
				return deployOptions{}, err
			}
		case "--minor":
			if err := setBump(release.Minor); err != nil {
				return deployOptions{}, err
			}
		case "--major":
			if err := setBump(release.Major); err != nil {
				return deployOptions{}, err
			}
		case "--no-release":
			o.noRelease = true
		case "-y", "--yes":
			o.yes = true
		default:
			if strings.HasPrefix(a, "-") {
				return deployOptions{}, fmt.Errorf("unknown option: %s", a)
			}
			positional = append(positional, a)
		}
	}

	if len(positional) > 1 {
		return deployOptions{}, fmt.Errorf("unexpected argument: %s", positional[1])
	}
	if len(positional) == 1 {
		tag := positional[0]
		if _, _, ok := release.ParseTag(tag); !ok {
			return deployOptions{}, fmt.Errorf("invalid tag %q: expected <prefix>MAJOR.MINOR.PATCH (e.g. v1.2.3)", tag)
		}
		o.explicitTag = tag
	}

	return o, nil
}

// deployMain is the entry point for `gmr deploy`, dispatched from main().
func deployMain(args []string) {
	opts, err := parseDeployArgs(args)
	if err != nil {
		if errors.Is(err, errDeployShowHelp) {
			fmt.Print(deployHelp)
			return
		}
		ui.Errf("%s", err.Error())
		return
	}
	if err := runDeploy(opts); err != nil {
		ui.Errf("%s", err.Error())
	}
}

// runDeploy implements `gmr deploy`: tag the next release from the commit log
// since the previous tag and publish it.
func runDeploy(opts deployOptions) error {
	r := git.NewRunner()

	if err := git.IsRepo(r); err != nil {
		return err
	}

	pc, err := detectPlatform(r)
	if err != nil {
		return err
	}

	mainBranch := git.DetectMainBranch(r)
	current, err := git.CurrentBranch(r)
	if err != nil {
		return err
	}
	if current != mainBranch {
		ui.Warn("Releases are normally cut from '%s', not '%s'.", mainBranch, current)
		if !opts.yes && stdinIsTTY() {
			fmt.Fprint(ui.Out, ui.Prompt(fmt.Sprintf("Continue from '%s'? [y/N]: ", current)))
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			if !isYes(line) {
				return errors.New("aborted")
			}
		}
	}

	hasChanges, err := git.HasChanges(r)
	if err != nil {
		return err
	}
	if hasChanges {
		return errors.New("uncommitted changes — commit or stash them before deploying")
	}

	if err := git.FetchTags(r); err != nil {
		ui.Warn("git fetch --tags failed: %s (continuing with local tags)", err)
	}

	tags, err := git.Tags(r)
	if err != nil {
		return err
	}
	latestTag, prefix, _, hasTag := release.Latest(tags)
	if !hasTag {
		prefix = "v"
		if v, ok := os.LookupEnv("GMR_TAG_PREFIX"); ok {
			prefix = v
		}
	}

	fromTag := ""
	if hasTag {
		fromTag = latestTag
	}
	log, err := git.LogRange(r, fromTag)
	if err != nil {
		return err
	}
	if log == "" {
		if hasTag {
			return fmt.Errorf("no commits since %s — nothing to release", latestTag)
		}
		return errors.New("repository has no commits")
	}

	limit := maxDiffLines()
	limited, truncated := git.LimitLines(log, limit)
	logContent := limited
	if truncated {
		logContent += fmt.Sprintf("\n... (log truncated at %d lines)", limit)
	}

	out := generate(ai.ReleasePrompt + logContent)
	var bump release.Bump
	var notes string
	if out == "" {
		ui.Warn("All AI providers unavailable — using the raw commit log as release notes. Review it before publishing.")
		bump = release.Patch
		notes = log
	} else {
		bump, notes = release.ParseAIResponse(out)
	}

	if opts.bumpForced {
		bump = opts.bump
	}

	tag := opts.explicitTag
	if tag == "" {
		tag = release.NextTag(tags, bump, prefix)
	}
	if git.TagExists(r, tag) {
		return fmt.Errorf("tag %s already exists", tag)
	}

	if !opts.yes {
		fmt.Fprintln(ui.Out)
		fmt.Fprintln(ui.Out, ui.Highlight(fmt.Sprintf("Release %s (%s bump):", tag, bump)))
		ui.Banner()
		fmt.Fprintln(ui.Out, notes)
		ui.Banner()
		fmt.Fprintln(ui.Out)

		if stdinIsTTY() {
			fmt.Fprint(ui.Out, ui.Prompt(fmt.Sprintf("Create tag %s and release? [Y/n/e(edit)]: ", tag)))
			reader := bufio.NewReader(os.Stdin)
			choice, _ := reader.ReadString('\n')
			switch strings.ToLower(strings.TrimSpace(choice)) {
			case "n":
				ui.Log("Aborted.")
				return nil
			case "e":
				edited, err := editInEditor(notes)
				if err != nil {
					return err
				}
				edited = strings.TrimSpace(edited)
				if edited == "" {
					return errors.New("release notes are empty. Aborted")
				}
				notes = edited
			}
		}
	}

	if err := git.CreateTag(r, tag, notes); err != nil {
		return err
	}
	ui.Log("Pushing tag %s...", tag)
	if err := git.PushTag(r, tag); err != nil {
		_ = git.DeleteLocalTag(r, tag)
		return fmt.Errorf("failed to push tag %s (local tag rolled back): %w", tag, err)
	}

	if !opts.noRelease {
		ui.Log("Creating release...")
		var c *exec.Cmd
		if pc.kind == platform.GitLab {
			c = exec.Command("glab", "release", "create", tag, "-R", pc.gitlabPath, "--name", tag, "--notes", notes)
		} else {
			c = exec.Command("gh", "release", "create", tag, "--title", tag, "--notes", notes)
		}
		c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := c.Run(); err != nil {
			ui.Warn("Failed to create the release (tag %s is already pushed): %s", tag, err)
			ui.Warn("Create it manually, e.g. `gh release create %s` or `glab release create %s`.", tag, tag)
		}
	}

	ui.OK("Released %s", tag)
	ui.Log("Run `gmr status` to watch the pipeline.")
	return nil
}

// isYes reports whether line (after trimming) is "y" or "yes", case-insensitive.
func isYes(line string) bool {
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
