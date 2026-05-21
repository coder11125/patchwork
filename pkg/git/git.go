package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Git struct {
	dir string
}

func New(dir string) *Git {
	return &Git{dir: dir}
}

func (g *Git) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *Git) Init(ctx context.Context) error {
	_, err := g.run(ctx, "init")
	return err
}

func (g *Git) Clone(ctx context.Context, repoURL, targetDir string) error {
	_, err := g.run(ctx, "clone", repoURL, targetDir)
	return err
}

func (g *Git) Checkout(ctx context.Context, ref string) error {
	_, err := g.run(ctx, "checkout", ref)
	return err
}

func (g *Git) CreateBranch(ctx context.Context, name string) error {
	_, err := g.run(ctx, "checkout", "-b", name)
	return err
}

func (g *Git) Commit(ctx context.Context, message string) error {
	_, err := g.run(ctx, "commit", "-m", message)
	return err
}

func (g *Git) Push(ctx context.Context, remote, branch string) error {
	_, err := g.run(ctx, "push", remote, branch)
	return err
}

func (g *Git) Diff(ctx context.Context) (string, error) {
	return g.run(ctx, "diff")
}

func (g *Git) CurrentBranch(ctx context.Context) (string, error) {
	return g.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
}

func (g *Git) IsClean(ctx context.Context) (bool, error) {
	out, err := g.run(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(out) == 0, nil
}

func (g *Git) Add(ctx context.Context, paths ...string) error {
	args := append([]string{"add"}, paths...)
	_, err := g.run(ctx, args...)
	return err
}

func (g *Git) AddAll(ctx context.Context) error {
	return g.Add(ctx, ".")
}

func (g *Git) Log(ctx context.Context, format string, n int) (string, error) {
	return g.run(ctx, "log", "-n", fmt.Sprintf("%d", n), "--format="+format)
}

func (g *Git) RemoteURL(ctx context.Context, remote string) (string, error) {
	return g.run(ctx, "remote", "get-url", remote)
}

func (g *Git) Fetch(ctx context.Context, remote string) error {
	_, err := g.run(ctx, "fetch", remote)
	return err
}

func (g *Git) Pull(ctx context.Context, remote, branch string) error {
	_, err := g.run(ctx, "pull", remote, branch)
	return err
}
