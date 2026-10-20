package service

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

type changeSourceReader interface {
	Commit(context.Context, string, string) error
	Read(context.Context, string, types.ChangeReference) (string, error)
}
type gitChangeSources struct{}

func (gitChangeSources) Commit(ctx context.Context, repo, commit string) error {
	if repo == "" {
		return fmt.Errorf("repository path is not configured")
	}
	out, err := exec.CommandContext(ctx, "git", "--no-pager", "-C", repo, "cat-file", "-t", commit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git cat-file: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if strings.TrimSpace(string(out)) != "commit" {
		return fmt.Errorf("object is not a commit")
	}
	return nil
}
func (g gitChangeSources) Read(ctx context.Context, repo string, ref types.ChangeReference) (string, error) {
	if ref.Path == "" || path.IsAbs(ref.Path) || strings.ContainsAny(ref.Path, "\\:\x00") || path.Clean(ref.Path) != ref.Path || ref.Path == ".." || strings.HasPrefix(ref.Path, "../") {
		return "", fmt.Errorf("source path must be repository-relative and normalized")
	}
	if ref.StartLine < 0 || ref.EndLine < 0 || (ref.EndLine > 0 && (ref.StartLine < 1 || ref.EndLine < ref.StartLine)) {
		return "", fmt.Errorf("invalid source line range")
	}
	if err := g.Commit(ctx, repo, ref.Commit); err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, "git", "--no-pager", "-C", repo, "show", ref.Commit+":"+ref.Path).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show: %w: %s", err, strings.TrimSpace(string(out)))
	}
	content := string(out)
	if strings.ContainsRune(content, 0) {
		return "", fmt.Errorf("binary sources are not supported")
	}
	if ref.StartLine > 0 {
		lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
		end := ref.EndLine
		if end == 0 {
			end = len(lines)
		}
		if ref.StartLine > len(lines) || end > len(lines) {
			return "", fmt.Errorf("source line range is outside the file")
		}
		content = strings.Join(lines[ref.StartLine-1:end], "\n")
	}
	return content, nil
}
