package utils

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const (
	obfuscationChars    = " .\\-/\\\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−"
	usernamePlaceholder = "клиент"
)

var (
	urlPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)https?://\S+`),
		regexp.MustCompile(`(?i)www\.\S+`),
		regexp.MustCompile(`(?i)tg://\S+`),
		regexp.MustCompile(`(?i)telegram\.me\S*`),
		regexp.MustCompile(`(?i)t\.me/\+\S*`),
		regexp.MustCompile(`(?i)joinchat\S*`),
	}

	obfuscatedDomainPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)[tт][\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*[\.\s\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*[\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*[mм][eе]`),
		regexp.MustCompile(`(?i)[tт][\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*[eе][\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*[lłl1i|][\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*[eе][\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*[gɢgqг][\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*[rр][\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*[aа][\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]*(?:[mм]|rn)`),
		regexp.MustCompile(`(?i)t\.me\S*`),
	}

	englishServicePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)telegram`),
		regexp.MustCompile(`(?i)teleqram`),
		regexp.MustCompile(`(?i)teiegram`),
		regexp.MustCompile(`(?i)teieqram`),
		regexp.MustCompile(`(?i)telegrarn`),
		regexp.MustCompile(`(?i)service`),
		regexp.MustCompile(`(?i)notif(?:ication)?`),
		regexp.MustCompile(`(?i)system`),
		regexp.MustCompile(`(?i)security`),
		regexp.MustCompile(`(?i)safety`),
		regexp.MustCompile(`(?i)support`),
		regexp.MustCompile(`(?i)moderation`),
		regexp.MustCompile(`(?i)review`),
		regexp.MustCompile(`(?i)compliance`),
		regexp.MustCompile(`(?i)abuse`),
		regexp.MustCompile(`(?i)spam`),
		regexp.MustCompile(`(?i)report`),
	}

	russianServicePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)телеграм\w*`),
		regexp.MustCompile(`(?i)служебн\w*`),
		regexp.MustCompile(`(?i)уведомлен\w*`),
		regexp.MustCompile(`(?i)поддержк\w*`),
		regexp.MustCompile(`(?i)безопасн\w*`),
		regexp.MustCompile(`(?i)модерац\w*`),
		regexp.MustCompile(`(?i)жалоб\w*`),
		regexp.MustCompile(`(?i)абуз\w*`),
	}

	normalizedBannedTokens = map[string]bool{
		"tme":          true,
		"telegram":     true,
		"teleqram":     true,
		"teiegram":     true,
		"teieqram":     true,
		"telegrarn":    true,
		"joinchat":     true,
		"notification": true,
		"moderation":   true,
		"review":       true,
		"compliance":   true,
		"abuse":        true,
		"spam":         true,
		"report":       true,
	}
	dangerousCombinations = [][]string{
		{"telegram", "support"},
		{"telegram", "admin"},
		{"service", "support"},
		{"system", "admin"},
		{"security", "admin"},
	}

	preLowerTranslation = map[rune]rune{
		'I': 'l',
		'İ': 'l',
		'Q': 'g',
		'＠': ' ',
	}

	postLowerTranslation = map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
		'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i",
		'і': "i", 'й': "i", 'к': "k", 'л': "l", 'м': "m",
		'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s",
		'т': "t", 'у': "u", 'ф': "f", 'х': "h", 'ц': "c",
		'ч': "ch", 'ш': "sh", 'щ': "sh", 'ъ': "", 'ы': "y",
		'ь': "", 'э': "e", 'ю': "yu", 'я': "ya", '＿': "_",
	}
)

func containsDangerousCombination(normalized string) bool {
	for _, combo := range dangerousCombinations {
		if strings.Contains(normalized, combo[0]) && strings.Contains(normalized, combo[1]) {
			return true
		}
	}
	return false
}

func normalizeForDetection(value string) string {
	if value == "" {
		return ""
	}

	normalized := norm.NFKD.String(value)

	var preLowerBuilder strings.Builder
	for _, r := range normalized {
		if replacement, ok := preLowerTranslation[r]; ok {
			preLowerBuilder.WriteRune(replacement)
		} else {
			preLowerBuilder.WriteRune(r)
		}
	}
	normalized = strings.ToLower(preLowerBuilder.String())

	var builder strings.Builder
	for _, r := range normalized {
		if unicode.In(r, unicode.Mn) {
			continue
		}
		if replacement, ok := postLowerTranslation[r]; ok {
			builder.WriteString(replacement)
		} else {
			builder.WriteRune(r)
		}
	}
	normalized = builder.String()

	normalized = strings.ReplaceAll(normalized, "rn", "m")

	obfuscationRegex := regexp.MustCompile(`[\s\.\-/\\•﹒٫＿․·∙‧ꞏ‒–—﹘﹣⁻−]+`)
	normalized = obfuscationRegex.ReplaceAllString(normalized, "")

	alphanumericRegex := regexp.MustCompile(`[^a-z0-9]+`)
	normalized = alphanumericRegex.ReplaceAllString(normalized, "")

	return normalized
}

func removePatterns(value string) string {
	updated := value
	allPatterns := append([]*regexp.Regexp{}, urlPatterns...)
	allPatterns = append(allPatterns, obfuscatedDomainPatterns...)
	allPatterns = append(allPatterns, englishServicePatterns...)
	allPatterns = append(allPatterns, russianServicePatterns...)

	for _, pattern := range allPatterns {
		updated = pattern.ReplaceAllString(updated, " ")
	}
	return updated
}

func finalize(value string, originalValue string) *string {
	spaceRegex := regexp.MustCompile(`\s+`)
	compacted := spaceRegex.ReplaceAllString(value, " ")
	compacted = strings.Trim(compacted, " \t\r\n-_.,/\\")
	compacted = strings.TrimSpace(compacted)

	if compacted == "" {
		return nil
	}

	normalizedOriginal := normalizeForDetection(originalValue)
	for token := range normalizedBannedTokens {
		if strings.Contains(normalizedOriginal, token) {
			return nil
		}
	}

	if containsDangerousCombination(normalizedOriginal) {
		return nil
	}

	normalized := normalizeForDetection(compacted)
	for token := range normalizedBannedTokens {
		if strings.Contains(normalized, token) {
			return nil
		}
	}

	if containsDangerousCombination(normalized) {
		return nil
	}

	return &compacted
}

// SanitizeDisplayName cleans and validates display names (first name, last name)
func SanitizeDisplayName(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	original := *value
	clean := strings.ReplaceAll(*value, "@", " ")
	clean = removePatterns(clean)
	return finalize(clean, original)
}

// SanitizeUsername cleans and validates usernames
func SanitizeUsername(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	original := *value
	clean := strings.TrimSpace(*value)
	clean = strings.TrimPrefix(clean, "@")
	clean = removePatterns(clean)
	return finalize(clean, original)
}

// UsernameForDisplay returns a safe display version of username with optional @ prefix
func UsernameForDisplay(username *string, withAt bool) string {
	sanitized := SanitizeUsername(username)
	if sanitized == nil || *sanitized == "" {
		return usernamePlaceholder
	}
	if withAt {
		return "@" + *sanitized
	}
	return *sanitized
}

// DisplayNameOrFallback returns sanitized display name or fallback if empty
func DisplayNameOrFallback(firstName *string, fallback string) string {
	sanitized := SanitizeDisplayName(firstName)
	if sanitized != nil && *sanitized != "" {
		return *sanitized
	}
	if fallback != "" {
		return fallback
	}
	return usernamePlaceholder
}

// IsSuspiciousUser checks if user has suspicious username or display name
// containsAlphanumeric checks if string contains any letter or digit
func containsAlphanumeric(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || (r >= 'а' && r <= 'я') ||
			(r >= 'А' && r <= 'Я') {
			return true
		}
	}
	return false
}

func IsSuspiciousUser(username *string, firstName *string, lastName *string) bool {
	if username != nil && *username != "" {
		if containsAlphanumeric(*username) && SanitizeUsername(username) == nil {
			return true
		}
	}
	if firstName != nil && *firstName != "" {
		if containsAlphanumeric(*firstName) && SanitizeDisplayName(firstName) == nil {
			return true
		}
	}
	if lastName != nil && *lastName != "" {
		if containsAlphanumeric(*lastName) && SanitizeDisplayName(lastName) == nil {
			return true
		}
	}
	return false
}
