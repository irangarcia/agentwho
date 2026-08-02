package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const Start = "# >>> agentwho >>>"
const End = "# <<< agentwho <<<"

func Init(shell, binDir string) (string, error) {
	switch shell {
	case "zsh", "bash":
		return fmt.Sprintf(`_agentwho_bin=%s
case ":$PATH:" in
  *:"$_agentwho_bin":*) ;;
  *) export PATH="$_agentwho_bin:$PATH" ;;
esac
agentwho_prompt() { command agentwho prompt --plain; }
agentwho() {
  if [ "$1" = "use" ] && [ "$2" != "--help" ] && [ "$2" != "-h" ]; then
    local _agentwho_use
    _agentwho_use="$(command agentwho internal shell-use %s "${@:2}")" || return $?
    eval "$_agentwho_use"
  else
    command agentwho "$@"
  fi
}
`, quote(binDir), shell), nil
	case "fish":
		return fmt.Sprintf(`set -l agentwho_bin %s
if not contains -- $agentwho_bin $PATH
    set -gx PATH $agentwho_bin $PATH
end
function agentwho_prompt
    command agentwho prompt --plain
end
function agentwho
    if test (count $argv) -ge 1; and test "$argv[1]" = use
        if test (count $argv) -ge 2; and contains -- "$argv[2]" -h --help
            command agentwho $argv
            return $status
        end
        set -l _agentwho_use (command agentwho internal shell-use fish $argv[2..-1])
        or return $status
        eval $_agentwho_use
    else
        command agentwho $argv
    end
end
`, fishQuote(binDir)), nil
	default:
		return "", fmt.Errorf("shell %q is not supported\n\nSupported shells:\n  zsh\n  bash\n  fish", shell)
	}
}

func UseProfile(shell, profile string) (string, error) {
	switch shell {
	case "zsh", "bash":
		return "export AGENTWHO_PROFILE=" + quote(profile), nil
	case "fish":
		return "set -gx AGENTWHO_PROFILE " + fishQuote(profile), nil
	default:
		return "", fmt.Errorf("shell %q is not supported\n\nSupported shells:\n  zsh\n  bash\n  fish", shell)
	}
}

func UseAutomatic(shell string) (string, error) {
	switch shell {
	case "zsh", "bash":
		return "unset AGENTWHO_PROFILE", nil
	case "fish":
		return "set -e AGENTWHO_PROFILE", nil
	default:
		return "", fmt.Errorf("shell %q is not supported\n\nSupported shells:\n  zsh\n  bash\n  fish", shell)
	}
}

func EvalLine(shell string) string {
	if shell == "fish" {
		return "agentwho shell init fish | source"
	}
	return fmt.Sprintf("eval \"$(agentwho shell init %s)\"", shell)
}

// PromptSetup returns an optional, composable snippet that prefixes the
// user's existing command prompt with the active AgentWho profile.
func PromptSetup(shell string) (string, error) {
	switch shell {
	case "zsh":
		return `setopt PROMPT_SUBST
PROMPT='$(agentwho prompt --plain) '"$PROMPT"`, nil
	case "bash":
		return `PS1='$(agentwho prompt --plain) '"$PS1"`, nil
	case "fish":
		return "# Add inside your existing fish_prompt function:\nagentwho prompt --plain", nil
	default:
		return "", fmt.Errorf("shell %q is not supported\n\nSupported shells:\n  zsh\n  bash\n  fish", shell)
	}
}

func Block(shell string) string      { return Start + "\n" + EvalLine(shell) + "\n" + End + "\n" }
func CountBlocks(content string) int { return strings.Count(content, Start) }

// IsConfigured reports whether a shell config already contains either an
// AgentWho-managed block or the documented manual initialization line.
func IsConfigured(path, shellName string) (bool, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	content := string(b)
	return CountBlocks(content) > 0 || strings.Contains(content, EvalLine(shellName)), nil
}

func AddBlock(path, shell string) (backup string, changed bool, err error) {
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	if CountBlocks(string(b)) > 0 {
		return "", false, nil
	}
	if len(b) > 0 {
		backup = path + ".agentwho.bak." + time.Now().Format("20060102-150405.000000000")
		if err := os.WriteFile(backup, b, 0o600); err != nil {
			return "", false, fmt.Errorf("create shell backup: %w", err)
		}
	}
	content := string(b)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += Block(shell)
	if err := atomic(path, []byte(content)); err != nil {
		return backup, false, err
	}
	return backup, true, nil
}

func RemoveBlocks(path string) (backup string, changed bool, err error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	content := string(b)
	if CountBlocks(content) == 0 {
		return "", false, nil
	}
	backup = path + ".agentwho.bak." + time.Now().Format("20060102-150405.000000000")
	if err := os.WriteFile(backup, b, 0o600); err != nil {
		return "", false, err
	}
	for {
		start := strings.Index(content, Start)
		if start < 0 {
			break
		}
		endRel := strings.Index(content[start:], End)
		if endRel < 0 {
			return backup, false, fmt.Errorf("unterminated AgentWho block in %s", path)
		}
		end := start + endRel + len(End)
		if end < len(content) && content[end] == '\n' {
			end++
		}
		content = content[:start] + content[end:]
	}
	if err := atomic(path, []byte(content)); err != nil {
		return backup, false, err
	}
	return backup, true, nil
}

func DefaultConfig(shellName, home string) string {
	switch shellName {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		if _, err := os.Stat(filepath.Join(home, ".bashrc")); err == nil {
			return filepath.Join(home, ".bashrc")
		}
		return filepath.Join(home, ".bash_profile")
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return ""
	}
}

func Detect(value string) string { return filepath.Base(value) }

func atomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".agentwho-shell-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func quote(s string) string     { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
func fishQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "\\'") + "'" }
