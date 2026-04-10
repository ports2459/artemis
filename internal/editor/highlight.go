package editor

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/eden/artemis/internal/app"
)

// Language represents a programming language for syntax highlighting.
type Language int

const (
	LangNone   Language = iota
	LangCSharp
	LangJSON
	LangTOML
	LangLua
)

// DetectLanguage returns the Language based on the file extension.
func DetectLanguage(path string) Language {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".cs":
		return LangCSharp
	case ".json":
		return LangJSON
	case ".toml":
		return LangTOML
	case ".lua":
		return LangLua
	default:
		return LangNone
	}
}

// Highlighting styles using theme colors.
var (
	styleKeyword = lipgloss.NewStyle().Foreground(app.ColorMauve)
	styleString  = lipgloss.NewStyle().Foreground(app.ColorGreen)
	styleComment = lipgloss.NewStyle().Foreground(app.ColorOverlay1)
	styleNumber  = lipgloss.NewStyle().Foreground(app.ColorPeach)
	styleType    = lipgloss.NewStyle().Foreground(app.ColorYellow)
	styleAttr    = lipgloss.NewStyle().Foreground(app.ColorTeal)
)

// highlightRule is a regex-based rule that maps matches to a style.
type highlightRule struct {
	pattern *regexp.Regexp
	style   lipgloss.Style
}

// styledToken tracks a styled region of text.
type styledToken struct {
	start int
	end   int
	style lipgloss.Style
}

// Pre-compiled regex rules for each language.
var (
	csharpRules = []highlightRule{
		{pattern: regexp.MustCompile(`//.*$`), style: styleComment},
		{pattern: regexp.MustCompile(`"(?:[^"\\]|\\.)*"`), style: styleString},
		{pattern: regexp.MustCompile(`\[[\w]+(?:\(.*?\))?\]`), style: styleAttr},
		{pattern: regexp.MustCompile(`\b\d+(?:\.\d+)?\b`), style: styleNumber},
		{pattern: regexp.MustCompile(`\b(?:public|private|protected|internal|static|class|struct|interface|enum|void|return|if|else|for|foreach|while|do|switch|case|break|continue|new|this|base|using|namespace|abstract|virtual|override|sealed|readonly|const|var|string|int|float|double|bool|true|false|null|async|await|try|catch|finally|throw|get|set)\b`), style: styleKeyword},
		{pattern: regexp.MustCompile(`\b(?:Harmony|HarmonyPatch|GameObject|Transform|Component|MonoBehaviour|Vector3|Vector2|Quaternion|Debug|Console|List|Dictionary|IEnumerable|Action|Func|Task|String|Int32|Boolean)\b`), style: styleType},
	}

	jsonRules = []highlightRule{
		{pattern: regexp.MustCompile(`"(?:[^"\\]|\\.)*"\s*:`), style: styleAttr},
		{pattern: regexp.MustCompile(`:\s*"(?:[^"\\]|\\.)*"`), style: styleString},
		{pattern: regexp.MustCompile(`"(?:[^"\\]|\\.)*"`), style: styleString},
		{pattern: regexp.MustCompile(`\b(?:true|false|null)\b`), style: styleKeyword},
		{pattern: regexp.MustCompile(`\b-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b`), style: styleNumber},
	}

	tomlRules = []highlightRule{
		{pattern: regexp.MustCompile(`#.*$`), style: styleComment},
		{pattern: regexp.MustCompile(`^\s*\[[\w.\-]+\]`), style: styleAttr},
		{pattern: regexp.MustCompile(`"(?:[^"\\]|\\.)*"`), style: styleString},
		{pattern: regexp.MustCompile(`'[^']*'`), style: styleString},
		{pattern: regexp.MustCompile(`^\s*[\w.\-]+\s*=`), style: styleType},
		{pattern: regexp.MustCompile(`\b(?:true|false)\b`), style: styleKeyword},
		{pattern: regexp.MustCompile(`\b\d+(?:\.\d+)?\b`), style: styleNumber},
	}

	luaRules = []highlightRule{
		{pattern: regexp.MustCompile(`--.*$`), style: styleComment},
		{pattern: regexp.MustCompile(`"(?:[^"\\]|\\.)*"`), style: styleString},
		{pattern: regexp.MustCompile(`'(?:[^'\\]|\\.)*'`), style: styleString},
		{pattern: regexp.MustCompile(`\b\d+(?:\.\d+)?\b`), style: styleNumber},
		{pattern: regexp.MustCompile(`\b(?:function|local|end|if|then|else|elseif|for|while|do|repeat|until|return|and|or|not|in|nil|true|false|require)\b`), style: styleKeyword},
	}
)

// tokenRegion tracks a region of text that has already been styled.
type tokenRegion struct {
	start int
	end   int
}

// HighlightLine applies syntax highlighting to a single line of text.
// It uses regex rules for the given language and returns a styled string.
// First match wins: once a region is claimed by a rule, later rules cannot override it.
func HighlightLine(line string, lang Language) string {
	if lang == LangNone {
		return line
	}

	rules := getRules(lang)
	if len(rules) == 0 {
		return line
	}

	// Collect all tokens with their regions
	var tokens []styledToken
	var claimed []tokenRegion

	for _, rule := range rules {
		matches := rule.pattern.FindAllStringIndex(line, -1)
		for _, m := range matches {
			start, end := m[0], m[1]
			if overlaps(claimed, start, end) {
				continue
			}
			tokens = append(tokens, styledToken{start: start, end: end, style: rule.style})
			claimed = append(claimed, tokenRegion{start: start, end: end})
		}
	}

	if len(tokens) == 0 {
		return line
	}

	// Sort tokens by start position
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].start < tokens[j].start
	})

	// Build the output string
	var result strings.Builder
	pos := 0
	for _, tok := range tokens {
		if tok.start > pos {
			result.WriteString(line[pos:tok.start])
		}
		result.WriteString(tok.style.Render(line[tok.start:tok.end]))
		pos = tok.end
	}
	if pos < len(line) {
		result.WriteString(line[pos:])
	}

	return result.String()
}

// HighlightContent highlights all lines in a multi-line string.
func HighlightContent(content string, lang Language) string {
	if lang == LangNone {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = HighlightLine(line, lang)
	}
	return strings.Join(lines, "\n")
}

func getRules(lang Language) []highlightRule {
	switch lang {
	case LangCSharp:
		return csharpRules
	case LangJSON:
		return jsonRules
	case LangTOML:
		return tomlRules
	case LangLua:
		return luaRules
	default:
		return nil
	}
}

// overlaps checks if a region [start, end) overlaps with any claimed region.
func overlaps(claimed []tokenRegion, start, end int) bool {
	for _, c := range claimed {
		if start < c.end && end > c.start {
			return true
		}
	}
	return false
}
