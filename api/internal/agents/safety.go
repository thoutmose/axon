package agents

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Health safety (EP-44). Skills with "safety: health" in their front matter
// get three guards that do not depend on the model behaving:
//   - a red-flag topic in the request gets a referral to a professional and
//     no model call at all, so nothing is sent;
//   - a reply that sets a daily energy target under 1,200 kcal or a fast of
//     two days or more is replaced by a safety message;
//   - a write tool whose input does either is refused.

// NeedsConsent means a skill needs a consent the person has not given.
type NeedsConsent struct{ Purpose string }

func (e NeedsConsent) Error() string { return "consent " + e.Purpose + " not granted" }

var redFlags = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(` + strings.Join([]string{
	// English
	`chest pain`, `chest tightness`, `faint(ed|ing)?`, `pass(ed)? out`, `black(ed)? out`, `short(ness)? of breath`, `can'?t breathe`,
	`heart (attack|palpitations?)`, `palpitations?`, `eating disorders?`, `anorexi\w*`, `bulimi\w*`, `binge(ing|s)?`, `purg(e|ing)`,
	`laxatives?`, `vomit\w*`, `starv\w*`, `not eating`, `suicid\w*`, `self[- ]harm\w*`, `kill myself`, `hurt myself`, `pregnan\w*`,
	`broken bone`, `fracture[ds]?`, `torn (ligament|muscle|tendon)`, `insulin`, `diabet\w*`, `medications?`, `blood pressure`,
	// French
	`douleurs? (thoracique|à la poitrine)`, `évanou\w*`, `malaises?`, `essouffl\w*`, `troubles? (alimentaires?|du comportement alimentaire)`,
	`anorexi\w*`, `boulimi\w*`, `vomi\w*`, `laxatifs?`, `suicid\w*`, `me faire du mal`, `enceinte`, `grossesse`, `médicaments?`,
	`tension artérielle`, `diabète`,
}, "|") + `)(?:$|[^\p{L}])`)

// RedFlag reports whether a request touches a topic a coach must hand to a professional.
func RedFlag(text string) bool { return redFlags.MatchString(text) }

// Referral answers a red-flag request instead of the model.
const Referral = "This sounds like something to talk about with a health professional, such as your doctor, " +
	"a registered dietitian, or a physiotherapist, rather than an app. If you are in danger or feel unwell right now, " +
	"call your local emergency number (112 in Europe, 911 in North America). If you are struggling with food, " +
	"your body, or thoughts of harming yourself, a support line can help today. I have not sent anything to the AI provider."

// UnsafeReply replaces a reply that crossed the safety line.
const UnsafeReply = "I won't suggest that: very low daily energy targets (under 1,200 kcal) or multi-day fasts can be harmful " +
	"without medical supervision. A registered dietitian can help you set targets that are right for you."

var (
	kcalRe = regexp.MustCompile(`(?i)(\d[\d ,.]{1,6})\s*(k?cal(ories)?|kcals)\b`)
	dayRe  = regexp.MustCompile(`(?i)\b(a|per|each|every|par|chaque)\s+(day|jour)\b|\bdaily\b|\bquotidien`)
	fastRe = regexp.MustCompile(`(?i)\b(fast|fasting|jeûne[rz]?|jeûner)\b[^.]{0,40}?\b(\d+)\s*(days?|jours?)\b|\b(\d+)[- ](day|jour)s?\s+(water )?(fast|jeûne)`)
)

// Unsafe reports whether text proposes a daily target under 1,200 kcal or a
// fast of two days or more. A figure is a daily target when "a day", "daily",
// or the like sits next to it on its line: a meal plan's kcal per meal is not.
func Unsafe(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if !dayRe.MatchString(line) {
			continue
		}
		for _, loc := range kcalRe.FindAllStringSubmatchIndex(line, -1) {
			n, err := strconv.Atoi(strings.NewReplacer(" ", "", ",", "", ".", "").Replace(line[loc[2]:loc[3]]))
			near := line[max(0, loc[0]-40):min(len(line), loc[1]+40)]
			if err == nil && n > 0 && n < 1200 && dayRe.MatchString(near) {
				return true
			}
		}
	}
	for _, m := range fastRe.FindAllStringSubmatch(text, -1) {
		for _, g := range []string{m[2], m[4]} {
			if days, err := strconv.Atoi(g); err == nil && days >= 2 {
				return true
			}
		}
	}
	return false
}

func unsafeToolError() error {
	return toolErrorf("%s", fmt.Sprintf("refused: %s", UnsafeReply))
}
