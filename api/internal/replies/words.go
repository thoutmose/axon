package replies

// words are the answers in each language, keyed by the English.
var words = map[string]map[string]string{"en": en, "fr": fr}

var en = map[string]string{
	"help": "Answer the digest here:\n" +
		"• done 2 — the second item is done (done 1 3, or just 2)\n" +
		"• done passport — I find it among your open tasks and habits\n" +
		"• add call the bank tomorrow at 3pm — a new task\n" +
		"• note … — a fleeting note in your inbox\n" +
		"• snooze 2 monday — move a task\n" +
		"• list — today’s items again, numbered\n" +
		"• undo — take back what I just did",
	"This bot answers one person. To use it with your Axon, open Preferences → Chat apps and send the code shown there: link ABCD-EFGH": "This bot answers one person. To use it with your Axon, open Preferences → Chat apps and send the code shown there: link ABCD-EFGH",
	"That code is wrong or has expired. Make a new one in Axon: Preferences → Chat apps.":                                               "That code is wrong or has expired. Make a new one in Axon: Preferences → Chat apps.",
	"Linked. Your digests arrive here, and I do what you answer.":                                                                       "Linked. Your digests arrive here, and I do what you answer.",
	"Not sure what you mean. Say help to see what I understand.":                                                                        "Not sure what you mean. Say help to see what I understand.",
	"I found nothing open like “%s”. Say list to see today’s items with their numbers.":                                                 "I found nothing open like “%s”. Say list to see today’s items with their numbers.",
	"No item %d in your latest list: say list to see it again.":                                                                         "No item %d in your latest list: say list to see it again.",
	"“%s” is gone from Axon.":                                 "“%s” is gone from Axon.",
	"“%s” was already done.":                                  "“%s” was already done.",
	"(the next one is planned)":                               "(the next one is planned)",
	"Which one?":                                              "Which one?",
	"Answer done and the number.":                             "Answer done and the number.",
	"in your inbox":                                           "in your inbox",
	"Saved to your inbox: “%s”":                               "Saved to your inbox: “%s”",
	"Say which item to move, for example: snooze 2 tomorrow.": "Say which item to move, for example: snooze 2 tomorrow.",
	"Only tasks can be moved; “%s” stays.":                    "Only tasks can be moved; “%s” stays.",
	"Nothing to undo.":                                        "Nothing to undo.",
	"Undone:":                                                 "Undone:",
	"Nice work.":                                              "Nice work.",
	"One less thing on your mind.":                            "One less thing on your mind.",
	"Keep going.":                                             "Keep going.",
	"Well done.":                                              "Well done.",
	"That is a productive moment.":                            "That is a productive moment.",
	"Great progress.":                                         "Great progress.",
}

var fr = map[string]string{
	"help": "Répondez au récapitulatif ici :\n" +
		"• fait 2 — le deuxième élément est fait (fait 1 3, ou juste 2)\n" +
		"• fait passeport — je le trouve parmi vos tâches et habitudes ouvertes\n" +
		"• ajoute appeler la banque demain à 15h — une nouvelle tâche\n" +
		"• note … — une note éphémère dans votre boîte de réception\n" +
		"• reporte 2 lundi — déplacer une tâche\n" +
		"• liste — les éléments du jour, numérotés\n" +
		"• annule — défaire ce que je viens de faire",
	"This bot answers one person. To use it with your Axon, open Preferences → Chat apps and send the code shown there: link ABCD-EFGH": "Ce bot répond à une seule personne. Pour l’utiliser avec votre Axon, ouvrez Préférences → Applications de discussion et envoyez le code affiché : link ABCD-EFGH",
	"That code is wrong or has expired. Make a new one in Axon: Preferences → Chat apps.":                                               "Ce code est faux ou a expiré. Créez-en un nouveau dans Axon : Préférences → Applications de discussion.",
	"Linked. Your digests arrive here, and I do what you answer.":                                                                       "C’est lié. Vos récapitulatifs arrivent ici, et je fais ce que vous répondez.",
	"Not sure what you mean. Say help to see what I understand.":                                                                        "Je ne suis pas sûr de comprendre. Dites aide pour voir ce que je comprends.",
	"I found nothing open like “%s”. Say list to see today’s items with their numbers.":                                                 "Je ne trouve rien d’ouvert qui ressemble à « %s ». Dites liste pour voir les éléments du jour et leurs numéros.",
	"No item %d in your latest list: say list to see it again.":                                                                         "Pas d’élément %d dans votre dernière liste : dites liste pour la revoir.",
	"“%s” is gone from Axon.":                                 "« %s » n’existe plus dans Axon.",
	"“%s” was already done.":                                  "« %s » était déjà fait.",
	"(the next one is planned)":                               "(la prochaine est planifiée)",
	"Which one?":                                              "Lequel ?",
	"Answer done and the number.":                             "Répondez fait et le numéro.",
	"in your inbox":                                           "dans votre boîte de réception",
	"Saved to your inbox: “%s”":                               "Enregistré dans votre boîte de réception : « %s »",
	"Say which item to move, for example: snooze 2 tomorrow.": "Dites quel élément déplacer, par exemple : reporte 2 demain.",
	"Only tasks can be moved; “%s” stays.":                    "Seules les tâches se déplacent ; « %s » reste.",
	"Nothing to undo.":                                        "Rien à défaire.",
	"Undone:":                                                 "Défait :",
	"Nice work.":                                              "Beau travail.",
	"One less thing on your mind.":                            "Une chose de moins en tête.",
	"Keep going.":                                             "Continuez comme ça.",
	"Well done.":                                              "Bien joué.",
	"That is a productive moment.":                            "Voilà un moment productif.",
	"Great progress.":                                         "Beaux progrès.",
}
