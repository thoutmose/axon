package digest

import (
	"hash/fnv"
	"slices"
	"strings"
)

// Quote is a line to start the day with, chosen for what the digest holds.
type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

type quote struct {
	en, fr, author string
	themes         []string // what the day holds: start, focus, persistence, learning, habits, rest, reflection, progress, planning, simplicity
	domains        []string // what the person works on: craft, writing, money, health
}

// quotes are short, well-sourced lines; doubtful attributions are left out.
// French versions are Axon's own translations.
var quotes = []quote{
	{"We are what we repeatedly do. Excellence, then, is not an act, but a habit.", "Nous sommes ce que nous faisons de manière répétée. L’excellence n’est donc pas un acte, mais une habitude.", "Will Durant", []string{"habits"}, nil},
	{"You do not rise to the level of your goals. You fall to the level of your systems.", "On ne s’élève pas au niveau de ses objectifs. On retombe au niveau de ses systèmes.", "James Clear", []string{"habits", "planning"}, nil},
	{"Every action you take is a vote for the type of person you wish to become.", "Chaque action est un vote pour la personne que vous voulez devenir.", "James Clear", []string{"habits", "progress"}, nil},
	{"Habit is a cable; we weave a thread of it each day, and at last we cannot break it.", "L’habitude est un câble ; chaque jour nous en tissons un fil, et à la fin nous ne pouvons plus le rompre.", "Horace Mann", []string{"habits"}, nil},
	{"The chains of habit are generally too small to be felt till they are too strong to be broken.", "Les chaînes de l’habitude sont d’ordinaire trop légères pour être senties, jusqu’à ce qu’elles soient trop fortes pour être brisées.", "Samuel Johnson", []string{"habits", "reflection"}, nil},
	{"Motivation is what gets you started. Habit is what keeps you going.", "La motivation vous fait commencer. L’habitude vous fait continuer.", "Jim Ryun", []string{"habits", "persistence"}, []string{"health"}},
	{"Well begun is half done.", "Bien commencé, c’est à moitié fait.", "Aristotle", []string{"start"}, nil},
	{"A journey of a thousand miles begins with a single step.", "Un voyage de mille lieues commence par un pas.", "Laozi", []string{"start", "progress"}, nil},
	{"Begin at once to live, and count each separate day as a separate life.", "Commence dès maintenant à vivre, et compte chaque jour comme une vie à part.", "Seneca", []string{"start", "rest"}, nil},
	{"While we are postponing, life speeds by.", "Pendant que nous remettons à plus tard, la vie passe.", "Seneca", []string{"start", "persistence"}, nil},
	{"It is not that we have a short time to live, but that we waste a lot of it.", "Ce n’est pas que nous ayons peu de temps, c’est que nous en perdons beaucoup.", "Seneca", []string{"focus", "reflection"}, nil},
	{"The impediment to action advances action. What stands in the way becomes the way.", "L’obstacle à l’action fait avancer l’action. Ce qui barre le chemin devient le chemin.", "Marcus Aurelius", []string{"persistence"}, nil},
	{"Waste no more time arguing about what a good man should be. Be one.", "Ne perds plus de temps à discuter de ce que doit être un homme de bien. Sois-le.", "Marcus Aurelius", []string{"start", "focus"}, nil},
	{"First say to yourself what you would be; and then do what you have to do.", "Dis-toi d’abord ce que tu veux être ; puis fais ce que tu as à faire.", "Epictetus", []string{"planning", "start"}, nil},
	{"No man is free who is not master of himself.", "Nul n’est libre s’il n’est maître de lui-même.", "Epictetus", []string{"focus", "habits"}, nil},
	{"The unexamined life is not worth living.", "Une vie sans examen ne vaut pas d’être vécue.", "Socrates", []string{"reflection"}, nil},
	{"Life can only be understood backwards; but it must be lived forwards.", "La vie ne peut être comprise qu’en regardant en arrière ; mais elle doit être vécue en regardant en avant.", "Søren Kierkegaard", []string{"reflection", "progress"}, nil},
	{"How we spend our days is, of course, how we spend our lives.", "La façon dont nous passons nos journées est, bien sûr, la façon dont nous passons nos vies.", "Annie Dillard", []string{"reflection", "focus"}, []string{"writing"}},
	{"Ever tried. Ever failed. No matter. Try again. Fail again. Fail better.", "Déjà essayé. Déjà échoué. Peu importe. Essaie encore. Échoue encore. Échoue mieux.", "Samuel Beckett", []string{"persistence"}, []string{"writing", "craft"}},
	{"Do what you can, with what you have, where you are.", "Fais ce que tu peux, avec ce que tu as, là où tu es.", "Theodore Roosevelt", []string{"persistence", "simplicity"}, nil},
	{"Far and away the best prize that life offers is the chance to work hard at work worth doing.", "Le plus beau prix que la vie offre est de loin la chance de travailler dur à un travail qui en vaut la peine.", "Theodore Roosevelt", []string{"focus", "progress"}, []string{"craft"}},
	{"Little by little, one travels far.", "Petit à petit, on va loin.", "Spanish proverb", []string{"persistence", "progress"}, nil},
	{"Small deeds done are better than great deeds planned.", "Mieux valent de petites actions faites que de grandes actions projetées.", "Peter Marshall", []string{"start", "simplicity"}, nil},
	{"Plans are worthless, but planning is everything.", "Les plans ne valent rien, mais la planification est tout.", "Dwight D. Eisenhower", []string{"planning"}, nil},
	{"What is important is seldom urgent and what is urgent is seldom important.", "Ce qui est important est rarement urgent, et ce qui est urgent est rarement important.", "Dwight D. Eisenhower", []string{"planning", "focus"}, nil},
	{"Focus is a matter of deciding what things you're not going to do.", "Se concentrer, c’est décider ce que l’on ne fera pas.", "John Carmack", []string{"focus", "simplicity"}, []string{"craft"}},
	{"Concentrate all your thoughts upon the work at hand. The sun's rays do not burn until brought to a focus.", "Concentrez toutes vos pensées sur la tâche du moment. Les rayons du soleil ne brûlent que réunis en un point.", "Alexander Graham Bell", []string{"focus"}, nil},
	{"Inspiration exists, but it has to find you working.", "L’inspiration existe, mais elle doit vous trouver au travail.", "Pablo Picasso", []string{"start", "focus"}, []string{"craft", "writing"}},
	{"Amateurs sit and wait for inspiration, the rest of us just get up and go to work.", "Les amateurs restent assis à espérer l’inspiration ; les autres se lèvent et vont travailler.", "Stephen King", []string{"start", "persistence"}, []string{"writing"}},
	{"Either write something worth reading or do something worth writing.", "Écrivez quelque chose qui vaille d’être lu, ou faites quelque chose qui vaille d’être écrit.", "Benjamin Franklin", []string{"progress"}, []string{"writing"}},
	{"You can't use up creativity. The more you use, the more you have.", "On n’épuise pas la créativité. Plus on s’en sert, plus on en a.", "Maya Angelou", []string{"start"}, []string{"writing", "craft"}},
	{"Knowing is not enough; we must apply. Willing is not enough; we must do.", "Savoir ne suffit pas, il faut appliquer. Vouloir ne suffit pas, il faut faire.", "Johann Wolfgang von Goethe", []string{"learning", "start"}, nil},
	{"If I have seen further it is by standing on the shoulders of Giants.", "Si j’ai vu plus loin, c’est en me tenant sur les épaules de géants.", "Isaac Newton", []string{"learning"}, nil},
	{"The first principle is that you must not fool yourself — and you are the easiest person to fool.", "Le premier principe est de ne pas vous tromper vous-même — et vous êtes la personne la plus facile à tromper.", "Richard Feynman", []string{"learning", "reflection"}, []string{"craft"}},
	{"Study hard what interests you the most in the most undisciplined, irreverent and original manner possible.", "Étudiez à fond ce qui vous intéresse le plus, de la manière la plus indisciplinée, irrévérencieuse et originale possible.", "Richard Feynman", []string{"learning"}, nil},
	{"The more that you read, the more things you will know. The more that you learn, the more places you'll go.", "Plus tu lis, plus tu sais de choses. Plus tu apprends, plus tu iras loin.", "Dr. Seuss", []string{"learning"}, nil},
	{"Repetition is the mother of learning.", "La répétition est la mère de l’apprentissage.", "Latin proverb", []string{"learning", "habits"}, nil},
	{"The palest ink is better than the best memory.", "L’encre la plus pâle vaut mieux que la meilleure mémoire.", "Chinese proverb", []string{"learning", "reflection"}, []string{"writing"}},
	{"Memory is the mother of all wisdom.", "La mémoire est la mère de toute sagesse.", "Aeschylus", []string{"learning"}, nil},
	{"Perfection is achieved, not when there is nothing more to add, but when there is nothing left to take away.", "La perfection est atteinte non quand il n’y a plus rien à ajouter, mais quand il n’y a plus rien à retrancher.", "Antoine de Saint-Exupéry", []string{"simplicity"}, []string{"craft", "writing"}},
	{"Programs must be written for people to read, and only incidentally for machines to execute.", "Les programmes doivent être écrits pour être lus par des humains, et seulement accessoirement exécutés par des machines.", "Harold Abelson and Gerald Jay Sussman", []string{"simplicity"}, []string{"craft"}},
	{"Make it work, make it right, make it fast.", "Faites que ça marche, que ce soit juste, puis que ce soit rapide.", "Kent Beck", []string{"progress", "simplicity"}, []string{"craft"}},
	{"Premature optimization is the root of all evil.", "L’optimisation prématurée est la racine de tous les maux.", "Donald Knuth", []string{"simplicity", "focus"}, []string{"craft"}},
	{"Rest is not idleness.", "Le repos n’est pas l’oisiveté.", "John Lubbock", []string{"rest"}, []string{"health"}},
	{"Almost everything will work again if you unplug it for a few minutes, including you.", "Presque tout remarche si on le débranche quelques minutes, vous y compris.", "Anne Lamott", []string{"rest"}, []string{"health"}},
	{"Those who think they have no time for bodily exercise will sooner or later have to find time for illness.", "Ceux qui pensent n’avoir pas le temps de faire de l’exercice devront tôt ou tard trouver le temps d’être malades.", "Edward Stanley, Earl of Derby", []string{"habits", "rest"}, []string{"health"}},
	{"Beware of little expenses; a small leak will sink a great ship.", "Méfiez-vous des petites dépenses : une petite voie d’eau coule un grand navire.", "Benjamin Franklin", []string{"planning"}, []string{"money"}},
	{"Do not save what is left after spending, but spend what is left after saving.", "N’épargnez pas ce qui reste après avoir dépensé : dépensez ce qui reste après avoir épargné.", "Warren Buffett", []string{"planning"}, []string{"money"}},
	{"The best time to plant a tree was twenty years ago. The second best time is now.", "Le meilleur moment pour planter un arbre, c’était il y a vingt ans. Le deuxième meilleur moment, c’est maintenant.", "Proverb", []string{"start", "persistence"}, []string{"money", "health"}},
}

// domainWords spot what the person works on, from their goals and areas.
var domainWords = map[string][]string{
	"craft":   {"code", "coding", "program", "software", "app", "build", "engineer", "develop", "design", "linux", "rust", "web", "data", "machine learning", "projet", "logiciel"},
	"writing": {"write", "writing", "book", "blog", "essay", "novel", "article", "écrire", "écriture", "livre", "roman"},
	"money":   {"money", "save", "saving", "budget", "invest", "debt", "finance", "épargne", "argent", "dette", "investir"},
	"health":  {"run", "running", "marathon", "gym", "fitness", "sleep", "weight", "health", "yoga", "walk", "sport", "santé", "sommeil", "marche"},
}

// Domains guesses what a person works on from words in their goals and areas.
func Domains(titles []string) []string {
	all := " " + strings.ToLower(strings.Join(titles, " ")) + " "
	var out []string
	for _, d := range []string{"craft", "writing", "money", "health"} {
		for _, w := range domainWords[d] {
			if strings.Contains(all, w) {
				out = append(out, d)
				break
			}
		}
	}
	return out
}

// pick chooses a quote for a theme and the person's domains: the best
// matches first, then one of them by the day, so it changes daily and is
// the same on every channel.
func pick(theme string, domains []string, seed string, lang string) Quote {
	best, top := []quote{}, -1
	for _, q := range quotes {
		score := 0
		if slices.Contains(q.themes, theme) {
			score += 2
		}
		for _, d := range domains {
			if slices.Contains(q.domains, d) {
				score++
			}
		}
		switch {
		case score > top:
			best, top = []quote{q}, score
		case score == top:
			best = append(best, q)
		}
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	q := best[int(h.Sum32()%uint32(len(best)))] //nolint:gosec // a handful of quotes
	if lang == "fr" {
		return Quote{Text: q.fr, Author: q.author}
	}
	return Quote{Text: q.en, Author: q.author}
}
