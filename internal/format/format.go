package format

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ListFormat interface {
	Read(path string, missingOK bool) ([]string, error)
	Write(path string, values []string) error
}

type NewlineFormat struct{}

type JSONArrayFormat struct{}

func New(name string) (ListFormat, error) {
	switch strings.ToLower(name) {
	case "newline", "lines", "txt":
		return NewlineFormat{}, nil
	case "json", "json-array", "jsonarray":
		return JSONArrayFormat{}, nil
	default:
		return nil, fmt.Errorf("unknown format %q", name)
	}
}

func (f NewlineFormat) Read(path string, missingOK bool) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if missingOK && os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		v := strings.TrimSpace(line)
		if v == "" || strings.HasPrefix(v, "#") {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

func (f NewlineFormat) Write(path string, values []string) error {
	if err := ensureDir(path); err != nil {
		return err
	}
	content := strings.Join(values, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func (f JSONArrayFormat) Read(path string, missingOK bool) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if missingOK && os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (f JSONArrayFormat) Write(path string, values []string) error {
	if err := ensureDir(path); err != nil {
		return err
	}
	b, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func Normalize(values []string, doSort bool) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if doSort {
		sort.Strings(out)
	}
	return out
}

var codexRulePatternRe = regexp.MustCompile(`pattern\s*=\s*\[(.*?)\]`)
var codexRuleDecisionRe = regexp.MustCompile(`decision\s*=\s*\"(allow|forbidden|prompt)\"`)

const codexManagedMarker = "# syncd-managed"

func ReadCodexRules(path string, missingOK bool) (allow []string, deny []string, err error) {
	f, err := os.Open(path)
	if err != nil {
		if missingOK && os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	expectManaged := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), codexManagedMarker) {
			expectManaged = true
			continue
		}
		if !expectManaged {
			continue
		}
		expectManaged = false
		pattern, decision, ok := parseCodexRuleLine(line)
		if !ok {
			continue
		}
		joined := strings.Join(pattern, " ")
		switch decision {
		case "allow":
			allow = append(allow, joined)
		case "forbidden":
			deny = append(deny, joined)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return allow, deny, nil
}

func WriteCodexRules(path string, allow []string, deny []string) error {
	kept, err := readCodexRuleFileKeepingNonManaged(path)
	if err != nil {
		return err
	}
	lines := make([]string, 0, len(kept)+len(allow)+len(deny))
	lines = append(lines, kept...)
	for _, cmd := range allow {
		lines = append(lines, codexManagedMarker)
		lines = append(lines, codexRuleLine(cmd, "allow"))
	}
	for _, cmd := range deny {
		lines = append(lines, codexManagedMarker)
		lines = append(lines, codexRuleLine(cmd, "forbidden"))
	}
	if err := ensureDir(path); err != nil {
		return err
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func readCodexRuleFileKeepingNonManaged(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var kept []string
	skipNext := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), codexManagedMarker) {
			skipNext = true
			continue
		}
		kept = append(kept, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return kept, nil
}

func parseCodexRuleLine(line string) ([]string, string, bool) {
	if !strings.Contains(line, "prefix_rule") {
		return nil, "", false
	}
	patternMatch := codexRulePatternRe.FindStringSubmatch(line)
	decisionMatch := codexRuleDecisionRe.FindStringSubmatch(line)
	if len(patternMatch) < 2 || len(decisionMatch) < 2 {
		return nil, "", false
	}
	pattern, ok := parseQuotedList(patternMatch[1])
	if !ok {
		return nil, "", false
	}
	return pattern, decisionMatch[1], true
}

func parseQuotedList(raw string) ([]string, bool) {
	var out []string
	i := 0
	for i < len(raw) {
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == ',') {
			i++
		}
		if i >= len(raw) {
			break
		}
		if raw[i] != '"' {
			return nil, false
		}
		i++
		var sb strings.Builder
		for i < len(raw) {
			switch raw[i] {
			case '\\':
				if i+1 >= len(raw) {
					return nil, false
				}
				sb.WriteByte(raw[i+1])
				i += 2
			case '"':
				i++
				out = append(out, sb.String())
				goto nextItem
			default:
				sb.WriteByte(raw[i])
				i++
			}
		}
		return nil, false
	nextItem:
		for i < len(raw) && raw[i] != '"' {
			if raw[i] == ',' {
				i++
				break
			}
			i++
		}
	}
	return out, true
}

func codexRuleLine(cmd string, decision string) string {
	parts := splitCommandPattern(cmd)
	quoted := make([]string, 0, len(parts))
	for _, p := range parts {
		quoted = append(quoted, "\""+escapeCodexString(p)+"\"")
	}
	return fmt.Sprintf("prefix_rule(pattern=[%s], decision=\"%s\")", strings.Join(quoted, ", "), decision)
}

func escapeCodexString(value string) string {
	value = strings.ReplaceAll(value, "\\\\", "\\\\\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\\\"")
	return value
}

func splitCommandPattern(value string) []string {
	var out []string
	var sb strings.Builder
	inQuote := false
	escape := false
	flush := func() {
		if sb.Len() == 0 {
			return
		}
		out = append(out, sb.String())
		sb.Reset()
	}
	for _, r := range value {
		switch {
		case escape:
			sb.WriteRune(r)
			escape = false
		case r == '\\':
			escape = true
		case r == '"' || r == '\'':
			inQuote = !inQuote
		case !inQuote && (r == ' ' || r == '\t' || r == '\n'):
			flush()
		default:
			sb.WriteRune(r)
		}
	}
	flush()
	return out
}

func ReadJSONKey(path string, missingOK bool, key string) ([]string, error) {
	root, err := readJSONObject(path, missingOK)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, nil
	}
	val, ok, err := getJSONPath(root, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	list, ok := toStringSlice(val)
	if !ok {
		return nil, fmt.Errorf("key %q is not an array of strings", key)
	}
	return list, nil
}

func ReadJSONBoolMap(path string, missingOK bool, key string) ([]string, []string, error) {
	root, err := readJSONObject(path, missingOK)
	if err != nil {
		return nil, nil, err
	}
	if root == nil {
		return nil, nil, nil
	}
	val, ok, err := getJSONPath(root, key)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, nil
	}
	obj, ok := val.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("key %q is not an object", key)
	}
	var allow []string
	var deny []string
	for k, v := range obj {
		switch vv := v.(type) {
		case bool:
			if vv {
				allow = append(allow, k)
			} else {
				deny = append(deny, k)
			}
		default:
			continue
		}
	}
	return allow, deny, nil
}

func WriteJSONBoolMap(path string, key string, allow []string, deny []string) error {
	root, err := readJSONObject(path, true)
	if err != nil {
		return err
	}
	if root == nil {
		root = map[string]any{}
	}
	out := make(map[string]any, len(allow)+len(deny))
	for _, v := range allow {
		out[v] = true
	}
	for _, v := range deny {
		out[v] = false
	}
	if err := setJSONPath(root, key, out); err != nil {
		return err
	}
	if err := ensureDir(path); err != nil {
		return err
	}
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func WriteJSONKey(path string, key string, values []string) error {
	root, err := readJSONObject(path, true)
	if err != nil {
		return err
	}
	if root == nil {
		root = map[string]any{}
	}
	if err := setJSONPath(root, key, values); err != nil {
		return err
	}
	if err := ensureDir(path); err != nil {
		return err
	}
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func readJSONObject(path string, missingOK bool) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if missingOK && os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	if root == nil {
		root = map[string]any{}
	}
	return root, nil
}

func getJSONPath(root map[string]any, key string) (any, bool, error) {
	if key == "" {
		return nil, false, fmt.Errorf("json key is empty")
	}
	parts := strings.Split(key, ".")
	var cur any = root
	for i, part := range parts {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("path %q is not an object at %q", key, strings.Join(parts[:i], "."))
		}
		next, ok := obj[part]
		if !ok {
			return nil, false, nil
		}
		cur = next
	}
	return cur, true, nil
}

func setJSONPath(root map[string]any, key string, value any) error {
	if key == "" {
		return fmt.Errorf("json key is empty")
	}
	parts := strings.Split(key, ".")
	cur := root
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, ok := cur[part]
		if !ok {
			child := map[string]any{}
			cur[part] = child
			cur = child
			continue
		}
		obj, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path %q is not an object at %q", key, strings.Join(parts[:i+1], "."))
		}
		cur = obj
	}
	cur[parts[len(parts)-1]] = value
	return nil
}

func toStringSlice(val any) ([]string, bool) {
	switch v := val.(type) {
	case []string:
		return v, true
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}
