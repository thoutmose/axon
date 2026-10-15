<p align="center">
  <img src="https://img.shields.io/badge/statut-d%C3%A9veloppement%20actif-blue" alt="Développement actif">
  <img src="https://img.shields.io/badge/licence-AGPL--3.0-purple" alt="AGPL-3.0">
  <img src="https://img.shields.io/badge/API-OpenAPI%203.1-6BA539" alt="OpenAPI 3.1">
</p>

<h1 align="center">Axon</h1>

<p align="center"><strong>Votre vie, auto-hébergée et connectée.</strong><br>Notes, action, apprentissage, santé, projets et réflexion dans un seul graphe, à taille humaine.</p>

<p align="center"><a href="README.md">English</a> · <strong>Français</strong></p>

> [!NOTE]
> Axon est en développement actif. Les fonctionnalités marquées ✅ marchent aujourd’hui ; les autres sont prévues, dans l’ordre donné par la [feuille de route](#feuille-de-route). Axon n’est ni un dispositif médical, ni un thérapeute, ni une promesse de mise en production.

## Sommaire

- [Pourquoi Axon ?](#pourquoi-axon-)
- [Fonctionnalités](#fonctionnalités)
- [Architecture](#architecture)
- [Pile technique](#pile-technique)
- [Matériel](#matériel-requis)
- [Démarrage rapide](#démarrage-rapide)
- [Configuration](#configuration)
- [Sécurité et RGPD](#sécurité-et-rgpd)
- [Agents IA](#agents-ia)
- [Gamification](#gamification)
- [Développement](#méthode-de-développement)
- [Tests et exploitation](#tests-et-exploitation)
- [Feuille de route](#feuille-de-route)
- [Contribuer](#contribuer)
- [Licence et remerciements](#licence-et-remerciements)

## Pourquoi Axon ?

Axon rassemble ce qui est d’habitude éparpillé entre applications de notes, listes de tâches, suivis, cours et tableurs. Son centre n’est ni un fil d’actualité ni un compteur de séries : c’est un graphe qui vous appartient, fait de vos engagements, de vos connaissances et de vos preuves. L’auto-hébergement, l’export, le consentement explicite et l’automatisation réversible sont des exigences de premier rang.

> [!WARNING]
> N’y saisissez ni urgence médicale, ni secret, ni données personnelles d’une autre personne. Les conseils des coachs sont informatifs et demandent le jugement d’un professionnel qualifié.

### Principes de fonctionnement

1. **Vos données vous appartiennent.** Tout s’exporte dans des formats ouverts ; l’effacement est vérifié, pas seulement mis en file d’attente.
2. **Privé par défaut.** Le consentement est explicite, par finalité, et révocable.
3. **Les agents proposent, les personnes décident.** Aucune modification produite par une IA ne s’applique sans approbation.
4. **Fonctionne sans IA.** Chaque fonctionnalité reste utile avec les agents et les API externes désactivés.
5. **Réversible plutôt qu’astucieux.** Des actions explicites et réversibles, et une technologie éprouvée.
6. **Reprendre, pas punir.** Un jour manqué ramène sur le chemin, jamais à la honte.
7. **Exploitable par d’autres.** Avec la documentation, n’importe qui peut le faire tourner, l’observer et le restaurer.

## Fonctionnalités

### 🧠 Second cerveau et Zettelkasten
- ✅ Un espace de notes : arborescence de dossiers, éditeur Markdown avec aperçu en direct, `[[liens wiki]]` avec suggestions, rétroliens et mentions, étiquettes, encadrés, tableaux, notes quotidiennes, modèles et centre de commandes (Ctrl+K).
- ✅ Dossiers PARA et notes Zettelkasten, import et export Markdown.
- ✅ Synchronisation bidirectionnelle avec le coffre `~/SecondBrain` (Git), qui garde les deux versions en cas de conflit.
- ✅ Raccourcis Vim dans l’éditeur de notes (en option), et raccourcis clavier partout : `?` pour la liste, `g` puis une lettre pour aller à une page.
- ✅ Synchronisation en direct avec votre coffre Markdown : ajoutez, modifiez, renommez ou supprimez un fichier et la note suit en quelques secondes, et inversement ; les modèles du coffre apparaissent dans Axon.
- ✅ Vos notes vivent dans Axon par défaut ; indiquez un serveur et un dossier dans les Préférences et une seule commande l’appaire, le synchronise dans les deux sens et prépare un coffre vide avec ses dossiers et ses modèles (Axon indique s’il est à jour).
- ✅ Des récapitulatifs du jour, de la semaine et du mois par e-mail, Slack, Discord, Microsoft Teams, Telegram, ntfy ou un webhook : ce qu’il y a à faire, ce qui arrive (tâches, jalons, objectifs), ce qu’il faut apprendre et se remémorer, le bilan de la semaine ou du mois, avec une image dessinée pour le jour et une citation choisie pour lui.
- ✅ Répondez au récapitulatif dans Slack, Discord ou Telegram : « fait 2 », « j’ai terminé le passeport », « ajoute appeler la banque demain à 15h », « reporte 3 lundi », « note … », « annule » — Axon le fait aussitôt, en français ou en anglais, sans adresse publique.
- ✅ Connexion avec Google, Microsoft, Apple, Discord, Slack ou votre propre fournisseur OpenID Connect, liés par vous dans Compte → Connexion et sécurité (jamais par l’adresse e-mail) ; les clés d’accès restent le moyen le plus sûr.
- ✅ Une veille technologique : suivez des flux, des sites, des auteurs Medium, des Substack, des chaînes YouTube et des dépôts GitHub (à la main, en CSV, en OPML ou depuis FreshRSS) selon une planification, minuit par défaut ; chaque histoire est gardée une seule fois quel que soit le nombre de sources, étiquetée par du TAL local, lue comme une note de veille et gardée en note quand elle compte ; chaque source affiche sa santé. Un graphe des actualités montre leurs concepts, leurs liens et ce qui monte, à part du graphe des notes, avec un résumé au survol. Toute vue s’exporte en PDF, ODS, CSV, Markdown, notes de veille ou JSON, ou devient un lien en lecture seule à partager. Activez son podcast pour écouter les meilleures actualités lues par votre appareil, et retrouvez-les dans vos récapitulatifs, dont vous choisissez les rubriques. Un agent de veille propose des sources et des sujets que vous approuvez.
- ✅ Un explorateur de connaissances (Notes → Explorer) : filtres par PARA, type Zettelkasten, étiquette, texte, date ou notes non reliées ; graphiques des notes dans le temps, par PARA et par type, des étiquettes et des mots-clés partagés ; et un graphe de connaissances qui ajoute à vos liens des relations NLP locales (entités partagées, mots-clés distinctifs, synonymes anglais et français, formulations proches), chacune expliquant pourquoi elle existe. Aucun fournisseur d’IA n’est utilisé.
- ✅ Analyse des notes de lecture et permanentes (Notes → Analyse) : racinisation Snowball en anglais et en français, expressions clés par TextRank, lisibilité, thèmes issus de formulations proches, un graphe de concepts dont chaque lien s’explique (ensemble plus souvent que le hasard selon la NPMI, ou synonymes d’Open English WordNet), et sur chaque note ses expressions clés, son thème et des idées liées avec leurs raisons. Tout reste local.
- ✅ Tout est une note dans votre coffre : tâches, objectifs, jalons, projets, domaines, ressources, archives, habitudes, films, séries et livres sont des notes Markdown sous `Axon/` (champs dans l’en-tête, texte en sections, une section Notes libre), synchronisées dans les deux sens : modifiez la note ou l’élément, l’autre suit ; une nouvelle note avec un type crée l’élément ; supprimer le fichier le supprime ; le déplacer dans `Archive/` l’archive. Les journaux d’alimentation et d’entraînement aussi, dans un seul sens, si vous l’activez.
- ✅ Des images et des PDF dans les notes : collez, déposez ou choisissez un fichier dans l’éditeur et il est intégré (`![](Attachments/…)`) ; les intégrations Obsidian comme `![[photo.png|300]]` s’affichent aussi. Les photos perdent leurs métadonnées et sont réduites à 2400 px ; les fichiers se synchronisent avec le coffre dans les deux sens et font partie de l’export.
- ✅ Des modèles pour chaque type de note, avec leurs sections affichées avant la création : éphémère, de lecture, permanente ; notes de projet, de domaine, de ressource et d’archive ; tâches, objectifs, jalons, projets, domaines, ressources, habitudes, films, séries, livres ; notes quotidiennes et de réunion ; et les modèles de votre coffre dans `Templates/` (Notes → Modèles).
- ✅ Un graphe de connaissances Zettelkasten sur la page Notes : notes de lecture et permanentes reliées par des indices combinés (expressions clés pondérées par l’IDF, synonymes WordNet, noms partagés, formulations proches, vos propres liens), communautés trouvées par la modularité de Louvain et nommées par TF-IDF de classe, centralité par PageRank, et notes passerelles entre communautés ; sélectionnez une note pour voir pourquoi elle est reliée.

### ✅ Action et exécution
- ✅ Des tâches avec les vues Aujourd’hui, semaine et GTD (boîte de réception, prochaines actions, en attente, un jour peut-être), contextes, récurrence, capture en une touche et bilans quotidiens.
- ✅ Des projets avec tableaux Kanban (utilisables au clavier, limites de travail en cours), epics, tickets, sous-tâches et documentation de projet ; les tâches s’ouvrent sur un tableau GTD, avec un Kanban à un clic ; le calendrier s’ouvre sur une grille mensuelle.
- ✅ Pages Domaines, Ressources et Archives (PARA), et des modèles pour chaque type de note, projet, domaine, ressource, objectif, jalon et habitude, conservés comme fichiers dans votre coffre.
- ✅ Un tableau de bord Aujourd’hui avec des indicateurs définis, un calendrier et des flux iCal.
- ✅ Un tableau de bord sur Aujourd’hui : indicateurs comparés à la période précédente, graphiques d’activité quotidienne et sur 12 mois, tableaux de ce qui demande votre attention, et un réseau de vos compétences techniques et humaines avec leurs prérequis (uniquement des compétences ; livres et cours restent rattachés à chacune).
- ✅ Sessions Pomodoro (25/5 × 4 puis 15 minutes de pause longue par défaut, tout est réglable, autant de sessions que voulu) liées à la tâche, au projet, à la compétence ou à la note travaillée, avec le temps de concentration sur le tableau de bord, une série quotidienne, des sons de concentration générés dans le navigateur (bruit, pluie), une musique composée pendant que vous travaillez et assortie à ce que vous faites (lo-fi pour les tâches et projets, ambiance pour les notes, piano calme pour les compétences et les cartes ; rien n’est diffusé depuis Internet) et vos propres liens de musique par activité.
- ✅ Finances personnelles dans l’esprit de Firefly III : comptes d’actifs et de passifs avec soldes, revenus, dépenses et virements (jamais comptés comme dépenses), catégories avec budgets mensuels, étiquettes, opérations récurrentes, règles de catégorisation, import CSV des relevés bancaires sans doublons, des rapports (flux mensuels, dépenses par catégorie, taux d’épargne, patrimoine net) et des tirelires pour un achat futur, avec un objectif, une date facultative et le montant à mettre de côté chaque mois. Les montants sont des centimes entiers exacts ; les données financières restent hors de la recherche et des agents, et figurent dans votre export.

### 🌱 Habitudes, objectifs et parcours
- ✅ Des habitudes conçues avec *Atomic Habits* de James Clear : créer de bonnes habitudes (évidentes, attrayantes, faciles, satisfaisantes) ou en perdre de mauvaises (invisibles, repoussantes, difficiles, insatisfaisantes), enchaînement (« Après…, je… »), intentions de mise en œuvre (« Je… à… à/dans… »), une identité pour laquelle chaque bilan vote, contrats d’habitude, tableau des habitudes, versions de deux minutes et invitations bienveillantes à reprendre.
- ✅ Objectifs et jalons reliés aux tâches, notes, habitudes, projets et parcours, avec une progression expliquée et une revue hebdomadaire.
- ✅ Des parcours d’apprentissage créés à la main, ou le parcours livré avec Axon : le top 5 % dans 26 domaines de l’ingénierie, 14 phases d’octobre 2026 à février 2030 avec leurs cours, livres et certifications. Le démarrer crée tout le plan, avec son rythme hebdomadaire en blocs d’étude dans l’agenda : l’objectif, un jalon et un projet par phase avec son examen de sortie en tâches et sous-tâches, les projets 42, quatre habitudes, les tâches de la première semaine, des domaines et des ressources. Chaque parcours s’affiche en liste ou en graphe de compétences par niveaux.
- ✅ Une toile d’araignée des compétences : les savoir-faire techniques et comportementaux acquis, et ceux en cours, domaine par domaine.
- ✅ Une évaluation des compétences avant de commencer : votre situation, vos objectifs et quelques questions par domaine donnent un niveau expliqué, et les parcours démarrent à votre niveau.
- ✅ Un agent Tuteur qui vous interroge puis rédige un parcours dont chaque compétence cite sa source, et un agent Ressources qui trouve de quoi apprendre. Les deux attendent votre approbation.
- ✅ Reliez n’importe quel élément à n’importe quel autre, là où il s’affiche.

### 📚 Apprentissage et médias
- ✅ Un journal de pratique pour les problèmes, niveaux de wargames et labs (LeetCode, Codeforces, Advent of Code, OverTheWire, PortSwigger, Kaggle…) : collez un lien, notez chaque tentative résolue seul, avec aide ou non, et refaites-les à intervalles croissants ; résolus par difficulté, sujet et semaine.
- ✅ Un journal de décisions : notez un choix, ce que vous attendez et avec quelle probabilité ; faites-en le bilan le jour venu et voyez si vos prévisions sont bien calibrées (score de Brier, courbe de fiabilité).
- ✅ Café et thé : la caféine face à la limite quotidienne de l'EFSA, ce qu'il en reste au coucher, et le dernier café qui préserve votre sommeil, selon votre heure de coucher et votre métabolisme.
- ✅ Les protocoles santé d'Andrew Huberman dans Habitudes, chacun avec ses études, une note honnête des preuves, et une habitude en un clic.
- ✅ Une collection dans Progression : un trophée par compétence de la feuille de route, une carte par habitude, et des jalons, du bronze à l'argent puis à l'or.
- ✅ Les certifications : prévues, réservées, obtenues ou échouées, avec le compte à rebours de l’examen, le coût face au plan, le certificat et l’expiration avec un rappel de renouvellement ; examens et expirations dans l’agenda. Le parcours prévoit ses neuf examens.
- ✅ Les heures d’étude face au plan : le temps de concentration sur un parcours, cette semaine jour par jour à côté du bloc prévu et sur 12 semaines face à l’objectif hebdomadaire, par compétence et par projet.
- ✅ Des cartes mémoire illustrées à consulter, ajouter, modifier et supprimer, révisées avec FSRS-5 (le planificateur d’Anki, vérifié sur son implémentation de référence), des tests d’entraînement, et un agent Enseignant qui rédige des questions de récupération à partir de vos notes de lecture et permanentes, soumises à votre approbation.
- ✅ Un moteur d’apprentissage et de mémorisation inspiré de *Understanding How We Learn* et de *Moonwalking with Einstein* : Apprendre → Aujourd’hui planifie la journée (cartes à réviser et leur durée, notes à se remémorer, test mixte hebdomadaire, notes éphémères à traiter, cartes difficiles à travailler) ; quelques nouvelles cartes par jour et des paquets mélangés (entrelacement) ; écrire sa réponse avant de la voir ; chaque carte peut porter le pourquoi, un exemple concret, une image mentale frappante, et une image ou un dessin de l’IA (double codage) ; rappel libre de notes entières sur une page blanche, comparé à leurs idées clés et espacé davantage à chaque réussite ; palais de mémoire parcourus lieu par lieu ; cartes tirées de ce qu’une note marque (`question :: réponse`, `==terme==`, `**Terme** : définition`) ; et l’Enseignant rédige des cartes illustrées soumises à votre approbation.
- ✅ Votre mémoire, mesurée : Apprendre compare le rappel prévu par la planification avec ce dont vous vous êtes souvenu, face à votre rétention cible, et dit quoi changer.
- ✅ Couvertures des films, séries et livres, retrouvées aussi pour ce qui est ajouté à la main ou importé, sur une étagère d’affiches.
- ✅ Rappels de tâches à une heure précise : dans la tâche, ou tapez « Appeler la banque vendredi à 15h » dans la saisie rapide.
- ✅ Partager vers Axon : depuis la feuille de partage de toute application une fois installé, ou un favori sur ordinateur ; enregistrer comme note ou tâche, ou garder le texte lisible de la page comme note de lecture.
- ✅ Une corbeille : ce que vous supprimez reste 30 jours avec ce qui l’accompagnait (les jalons d’un objectif, les pointages d’une habitude), Annuler juste après la suppression, et Compte → Corbeille pour restaurer.
- ✅ Actions groupées sur les tâches (statut, contexte, échéance, suppression), et glisser les tâches d’un jour à l’autre de l’agenda ou dessus depuis les prochaines actions sans date.
- ✅ Un Aujourd’hui à votre façon : choisissez et ordonnez ses cartes, dont l’apprentissage, les habitudes, l’agenda du jour et le foyer.
- ✅ Seulement ce qui vous sert : un nouveau compte dit pour quoi il veut Axon, et le menu et Aujourd’hui n’affichent que ces pages ; chaque page se masque ou revient dans les Préférences.
- ✅ Recherche par le sens, si vous le voulez : choisissez un service de plongements (un serveur de modèles local, OpenAI ou Voyage AI) et donnez son consentement propre ; les notes se retrouvent alors par ce qu’elles veulent dire, dans le centre de commandes et sous « Proche par le sens ». Désactivée, les notes liées utilisent des statistiques de mots locales.
- ✅ Partage en foyer : les personnes d’un même Axon rejoignent le foyer avec un code à usage unique et partagent des listes (courses, tâches ménagères, tout), avec des éléments confiés à quelqu’un ; rien de personnel n’est partagé.
- ✅ Imports depuis d’autres applications : tâches Todoist et TickTick, livres Goodreads, films Letterboxd, transactions Firefly III et relevés bancaires OFX/QFX, pages Notion ; aperçu d’abord, et réimporter n’ajoute rien en double.
- ✅ Une revue hebdomadaire guidée façon Getting Things Done : vider sa tête, boîte de réception à zéro, notes éphémères, faits marquants de la semaine, deux semaines à venir, en attente, projets bloqués, objectifs, habitudes et apprentissage, un jour peut-être, et ce qui compte la semaine prochaine ; proposée le week-end au réveil tant qu’elle n’est pas faite.
- ✅ Vos autres agendas dans Axon, en lecture seule : suivez Google, Nextcloud ou tout serveur CalDAV, Outlook ou toute adresse ICS ; événements répétés, fuseaux horaires et exceptions gérés ; affichés dans les vues mois et agenda avec leur couleur.
- ✅ Saisie rapide de tout : appuyez sur n et tapez `Appeler le plombier @téléphone demain`, `livre: Atomic Habits`, `habitude: Lire une page`, `objectif: Courir 10 km d’ici 2027-05-01` ou `carte: question :: réponse #Paquet` ; la saisie montre ce qu’elle va ajouter.
- ✅ Rappels : un résumé le matin (tâches, cartes à réviser et leur durée, notes à se remémorer, habitudes, test hebdomadaire) et un rappel le soir (habitudes encore ouvertes, avec leur version de deux minutes), en notifications sur vos appareils, chiffrées de bout en bout, et par e-mail si votre serveur peut l’envoyer ; une fois par jour, seulement s’il y a quelque chose à dire, désactivés par défaut.
- ✅ Installable sur téléphone et ordinateur ; hors ligne, la saisie rapide garde tâches et notes sur l’appareil et les envoie, une seule fois, au retour de la connexion.
- ✅ Un menu en cinq groupes (Faire, Penser, Grandir, Vie, Bilan) plus Aujourd’hui, et une seule feuille Menu sur téléphone.
- ✅ Les agents là où vous travaillez : chaque page propose les agents dont les compétences lui correspondent (Gestionnaire de tâches sur Tâches et Aujourd’hui, Objectifs, Habitudes, l’Enseignant sur Notes, une note, Apprendre et Cartes, le Tuteur et Ressources sur Parcours, Médias, le Coach sportif et la Nutritionniste sur Santé, Projets annexes et Revue de code sur Projets, Veille sur la veille technologique), avec des demandes toutes prêtes et ce que la page affiche joint à la demande ; les propositions s’approuvent sur place.
- ✅ Paquets Anki (.apkg) par paquet de cartes, images comprises, et synchronisation automatique avec votre propre `anki-sync-server` (par exemple sur `srv-services`), à la demande ou dès que vous approuvez les cartes de l’Enseignant ; les cartes présentes seulement dans Anki ne sont jamais supprimées.
- ✅ Films, séries et livres : à voir ou à lire, en cours, terminés ou abandonnés, avec une note sur 10, la progression en pages, épisodes ou minutes, et une critique. Une fois les recherches autorisées (elles envoient le titre saisi), les livres viennent d’Open Library, et les films et séries de TMDB si le serveur a une clé ; tout peut s’ajouter à la main. Les couvertures sont copiées dans Axon, donc les pages ne chargent jamais d’images ailleurs. « À suivre » classe votre liste avec ses raisons (même créateur ou genre que ce que vous avez noté 8 ou plus, ou ce que vous avez commencé), et un agent Médias propose de nouveaux titres soumis à votre approbation.

### 🥗 Santé et sport
- ✅ L’eau : un verre ou une bouteille en un geste, face à la part des boissons dans votre besoin quotidien en eau, avec la semaine ; des données de santé, sous leur consentement.
- ✅ Un journal quotidien : humeur, énergie et stress sur cinq visages, heures de sommeil, trois choses pour lesquelles vous êtes reconnaissant et une note, avec les 30 derniers jours en graphique ; sur Aujourd’hui en deux gestes si vous voulez. Des données de santé, sous leur consentement.
- ✅ Ce qui vous aide : quelles habitudes vont avec vos meilleurs jours (sommeil, sport, concentration, habitudes, tâches, gratitude face à l’humeur, l’énergie et le stress), avec des statistiques qui écartent le hasard, présentées comme des liens, jamais comme des causes. Calculé sur votre serveur, sans IA.
- ✅ Journaux d’alimentation et d’entraînement, seulement après votre consentement à conserver des données de santé (le retirer les supprime) : repas avec énergie et macronutriments, trouvés dans Open Food Facts (par nom ou code-barres) ou USDA FoodData Central une fois les recherches autorisées, ou saisis ; séances avec séries, minutes et effort. Jamais dans la recherche ni les agents IA. La photo d’un repas estimée par votre propre fournisseur d’IA en aliments, portions et macronutriments que vous vérifiez avant de les noter (la photo n’est pas conservée ; ses informations d’appareil et de lieu sont retirées avant), et la photo d’un code-barres lue sur votre appareil. Des besoins quotidiens pour garder votre poids (énergie, protéines, glucides, lipides, fibres, eau) calculés selon votre âge, sexe, gabarit et activité à partir de références publiées, chacun expliqué, avec un avertissement à vérifier auprès d’un diététicien ; aucun avant 18 ans ni pendant la grossesse, et aucun objectif de perte de poids.
- ✅ Agents Coach sportif et Nutritionniste qui ne lisent vos journaux qu’avec un consentement distinct pour les partager avec votre fournisseur d’IA, proposent des séances ou des tâches de repas simples à approuver, renvoient les sujets d’alerte (douleur thoracique, malaise, troubles alimentaires, grossesse, médicaments…) vers un professionnel sans appeler le modèle, et ne peuvent jamais suggérer moins de 1 200 kcal par jour ni des jeûnes de plusieurs jours.

### ✨ Progrès à taille humaine
- ✅ Récompenses de progression optionnelles : des points pour chaque domaine (tâches, projets, habitudes, concentration, cartes mémoire, notes, parcours, objectifs), chacun traçable dans un registre, des niveaux, des badges, une série quotidienne avec des gels achetés uniquement avec des points gagnés, sans honte ni notifications.
- ✅ Analyses : neuf indicateurs hebdomadaires (tâches terminées et capturées, bilans d’habitudes, jalons, minutes de concentration, notes, cartes révisées, taux de rappel, compétences acquises), chacun avec sa formule, sa source, une tendance en mots simples et les éléments derrière chaque chiffre ; une revue hebdomadaire comparée à la semaine précédente, avec ce qui a changé et les tâches ouvertes à reporter ; export CSV. Tout est calculé sur votre serveur.
- ✅ Conception éthique : pas de dark patterns, pas de honte, pas de consentement caché, des contrôles conformes au RGPD.
- ✅ Une photo de profil, avec une image par défaut générée (vos initiales sur vos couleurs) ; les envois sont réencodés, aucune donnée de localisation ne subsiste.
- ✅ Des espaces thématiques : chaque domaine a son propre décor original, en clair et en sombre : un carnet de Léonard de Vinci pour les notes, une bibliothèque d’Oxford pour l’apprentissage, les ressources et les cartes mémoire, un plan d’atelier pour les projets, un jardin pour les habitudes, une carte topographique pour les objectifs et la progression, une carte du ciel pour le calendrier, un registre pour le budget, un vieil atlas pour les domaines et les archives, et du papier pointillé pour les tâches et la concentration. Les effets sont lents, s’arrêtent avec la réduction des animations et peuvent être désactivés dans les préférences. Journal, Tendances, la revue hebdomadaire et Agents ont aussi le leur (la lumière du soir de Vermeer, un observatoire, un phare, des automates) ; chaque page dit ce que montre son image et ce qu’elle évoque, et chaque type de note a son petit dessin.
- ✅ Un ciel qui suit le soleil : la page d’accueil et Aujourd’hui montrent le ciel tel qu’il est dans votre fuseau horaire, le soleil et la lune (avec sa phase) là où ils sont, les étoiles après le crépuscule, au-dessus du paysage de votre choix : montagnes, forêt, océan, désert, la Terre vue de l’orbite, ou un monument de chaque continent (tour Eiffel, pyramides de Gizeh, Taj Mahal, statue de la Liberté, Christ Rédempteur, opéra de Sydney). Le thème peut le suivre aussi : clair de l’aube au crépuscule.
- ✅ Déplacez tout votre Axon vers un autre serveur : `axon instance export` et `axon instance import` transportent chaque table et chaque image avec un manifeste de sommes de contrôle, vers une installation neuve, y compris depuis une version plus ancienne.
- ✅ Des préférences au même endroit : anglais ou français (toutes les pages, avec les dates dans votre langue), thème, police, taille du texte, icônes, animations, modèles pour les notes, projets et ressources, et votre fournisseur d’IA et budget. Modifiables dans l’application ou dans un fichier YAML (les clés sont en écriture seule).

## Captures d’écran

| Accueil | Aujourd’hui (clair) | Aujourd’hui (sombre, téléphone) |
|---|---|---|
| ![Page d’accueil avec un paysage de montagne nocturne et des sections de fonctionnalités](doc/screenshots/home.png) | ![Vue Aujourd’hui avec un bandeau paysager, des compteurs de tâches et des cartes](doc/screenshots/today.png) | ![Vue Aujourd’hui en mode sombre sur téléphone](doc/screenshots/today-mobile-dark.png) |

| Tableau de bord | Espace de notes | Centre de commandes |
|---|---|---|
| ![Tableau de bord avec indicateurs, graphique d’activité, carte de chaleur, réseau de compétences et tableaux](doc/screenshots/dashboard.png) | ![Espace de notes avec arborescence, éditeur Markdown et aperçu en direct](doc/screenshots/notes-edit.png) | ![Centre de commandes cherchant dans les notes](doc/screenshots/command-center.png) |

Capturées sur un compte de test avec des données d’exemple.

<details><summary>Direction visuelle</summary>

Grande typographie, espaces généreux, mode sombre, verre translucide sur des paysages intégrés (ADR 0010). Le mouvement doit clarifier l’état, comme un instrument bien fait, sans réclamer l’attention. Les préférences de mouvement réduit sont respectées, et chaque action reste utilisable sans animation.

</details>

## Architecture

Axon est d’abord un monolithe modulaire en Go, avec PostgreSQL 16 + pgvector comme source de vérité (le travail de fond tourne dans le processus, coordonné par PostgreSQL), et SvelteKit + TypeScript pour l’application web. Les agents appellent des outils bornés via une couche d’orchestration qui respecte l’approbation.

### Mermaid — contexte du système

```mermaid
flowchart LR
  user(["Person: owns the data, approves every AI change"])
  axon["Axon: self-hosted life OS"]
  ai["AI provider of each person's choice: Anthropic, OpenAI, Mistral, Gemini, DeepSeek, or a local model"]
  cat["Catalogues: Open Library, TMDB, Open Food Facts, USDA"]
  anki["anki-sync-server"]
  vault["Markdown vault, e.g. Obsidian"]
  user -->|"plans, captures, learns, reviews"| axon
  axon -->|"approved prompts and tools, with consent"| ai
  axon -->|"lookups, with consent"| cat
  axon -->|"flashcards"| anki
  axon <-->|"notes and items, both ways"| vault
```

### Mermaid — conteneurs

```mermaid
flowchart TB
  user(["Browser or phone"])
  subgraph host["One host per environment, Docker Compose"]
    caddy["Caddy: HTTPS front, headers"]
    web["Web: SvelteKit + TypeScript"]
    api["API: Go + chi, REST and OpenAPI 3.1"]
    workers["In-process workers: agents, vault mirror, demo reset, media prune"]
    bridge["Anki bridge: official Anki library"]
    db[("PostgreSQL 16 + pgvector: system of record")]
    backup["Backup: encrypted dumps"]
  end
  user --> caddy
  caddy --> web
  caddy -->|"/api"| api
  web -->|"server-side calls"| api
  api --- workers
  api --> db
  workers --> db
  api --> bridge
  backup --> db
```

### Mermaid — déploiement

```mermaid
flowchart LR
  dev["Your machine: make up, make check"] -->|"deploy.sh staging, green CI only"| staging["srv-staging: Axon + PostgreSQL"]
  staging -->|"deploy.sh prod, only what staging serves"| prod["srv-prod: Axon + PostgreSQL"]
  prod -->|"Anki sync, encrypted backups, tailnet"| services["srv-services: anki-sync-server, backups"]
  prod <-->|"notes-sync -watch, tailnet"| brain["srv-secondbrain: Markdown vault"]
```

### Mermaid — modèle de données et liens universels

```mermaid
erDiagram
  USER ||--o{ ENTITY : owns
  ENTITY ||--o{ LINK : "source or target"
  ENTITY ||--o{ EVENT : "history"
  ENTITY ||--o| NOTE : "is a"
  ENTITY ||--o| TASK : "is a"
  ENTITY ||--o| GOAL : "is a"
  GOAL ||--o{ MILESTONE : has
  ENTITY ||--o| PROJECT : "is a"
  ENTITY ||--o| HABIT : "is a"
  ENTITY ||--o| ROADMAP : "is a"
  ROADMAP ||--o{ SKILL : has
  ENTITY ||--o| CARD : "is a"
  CARD ||--o{ CARD_REVIEW : "reviewed in"
  ENTITY ||--o| MEDIA_ITEM : "is a"
  ENTITY ||--o| VAULT_NOTE : "mirrored as"
  ENTITY {
    uuid id PK
    uuid user_id FK
    string type
    string title
  }
  LINK {
    uuid id PK
    uuid source_id FK
    string relation
    uuid target_id FK
  }
  EVENT {
    uuid id PK
    string kind
    string fields
  }
```

### Mermaid — boucle d’apprentissage et de mémorisation

```mermaid
sequenceDiagram
  actor P as Person
  participant N as Notes
  participant T as Teacher agent
  participant R as Review engine, FSRS
  participant L as Learn: today's plan
  participant A as anki-sync-server
  P->>N: captures fleeting notes, rewrites them as literature and permanent notes
  P->>N: marks what to remember: q :: a, ==term==, **Term**: definition
  N-->>R: Make cards: drafts from the marks, added by the person
  P->>T: illustrated cards from these notes
  T->>N: reads only literature and permanent notes
  T-->>P: proposes cards with the why, an example, a vivid image, an SVG drawing
  P->>R: approves, then places cards in a memory palace
  loop Every day
    L-->>P: cards due, a few new ones, decks mixed, and notes due for recall
    P->>R: writes the answer, then sees it with the why, the example, and the image, and rates it
    R->>R: schedules the next review at the chosen retention
    P->>N: recalls a whole note on a blank page
    N-->>L: ideas recalled and missed, next recall further away when recalled well
  end
  L-->>P: weekly mixed test, and leeches to practise and rewrite
  R->>A: syncs approved cards
```

### Mermaid — évaluation et parcours du Tuteur

```mermaid
sequenceDiagram
  actor P as Person
  participant As as Assessment
  participant Tu as Tutor agent
  participant L as Curated library
  participant Ro as Roadmaps
  P->>As: situation, job, goals, current level
  As-->>Ro: places you on roadmaps
  P->>Tu: what to learn, time, format
  Tu->>L: searches the best resources
  L-->>Tu: books, courses, papers, videos
  Tu-->>P: draft roadmap, every skill cites its source
  P->>Ro: approves, edits, or removes
```

### Mermaid — miroir du coffre, dans les deux sens

```mermaid
sequenceDiagram
  participant I as Item, e.g. a task
  participant M as Vault mirror worker
  participant N as Note under Axon/
  participant V as Markdown vault
  I->>M: created, changed, or deleted
  M->>N: writes front matter and sections
  N->>V: notes-sync carries the file
  V->>N: you edit, add, move, or delete the file
  N->>M: change in the event log
  M->>I: updates, creates, archives, or deletes the item
```

### Mermaid — série quotidienne et gels

```mermaid
stateDiagram-v2
  [*] --> Idle
  Idle --> Active: first qualifying action of the day
  Active --> Active: an action on each following day
  Active --> AtRisk: a day passes without an action
  AtRisk --> Active: act today
  AtRisk --> Frozen: a freeze bought with earned points covers the day
  Frozen --> Active: act on the next day
  AtRisk --> Restart: two days without an action
  Restart --> Active: start again, without shame
```

## Pile technique

| Couche | Choix | Pourquoi |
|---|---|---|
| Backend | Go + chi | Peu de mémoire, concurrence explicite, déploiement simple, très bons outils. |
| API | REST + OpenAPI 3.1 ; GraphQL éventuellement plus tard | Cache, observabilité, clients générés ; pas de flexibilité prématurée. |
| Frontend | SvelteKit + TypeScript, jetons de design + CSS limité aux composants ([ADR 0007](doc/adr/0007-styling-without-tailwind.md)) | Une interface petite et rapide, accessible, en amélioration progressive. |
| Données | PostgreSQL 16 + pgvector | Un cœur relationnel durable : texte intégral, vecteurs, contraintes, extensions. |
| Travail de fond | Workers dans le processus, lisant PostgreSQL (curseur du journal d’événements, tables de file) | Aucun service de plus ; la base reste l’unique source de vérité ([ADR 0022](doc/adr/0022-no-valkey-yet.md)). |
| Vue graphe | D3 ou Sigma.js | Visualisation interactive des compétences et des relations, qui passe à l’échelle. |
| Authentification | WebAuthn + TOTP intégrés | Clés d’accès, MFA, contrôle auto-hébergé. |
| Entrée réseau | Caddy ou Traefik | TLS automatique et routage clair. |
| Exploitation | Prometheus, Grafana, Loki, OpenTelemetry | Métriques, tableaux de bord, journaux, traces. |
| Livraison | GitOps ou Compose + Ansible/OpenTofu | Une promotion reproductible adaptée aux petits parcs. |
| Sécurité | SOPS/age, Trivy, SBOM, Renovate, pratiques SLSA | Réduire les risques de chaîne d’approvisionnement et de secrets. |

Là où le tableau laisse un choix, l’[ADR 0002](doc/adr/0002-initial-technology-stack.md) en propose un : WebAuthn + TOTP intégrés, Caddy, et Compose + Ansible.

**Pourquoi Go et pas Rust ?** Rust apporterait des garanties plus fortes à la compilation et un peu moins de mémoire. Mais le travail d’Axon est surtout de l’entrée-sortie (PostgreSQL, HTTP, fournisseurs d’IA), où Go est aussi rapide en pratique. Go compile en quelques secondes, dispose de SDK officiels pour les services utilisés par Axon, et garde le code facile à lire pour un mainteneur seul comme pour des agents IA. Réécrire l’API existante arrêterait les nouvelles fonctionnalités pendant des mois, sans gain visible pour les utilisateurs. Le choix est consigné dans l’ADR 0002, et le [journal des changements](doc/BACKLOG.md#change-log) note qu’une réécriture a été envisagée sans être planifiée.

## Matériel requis

| Hôte | Minimum | Confortable | Charge |
|---|---:|---:|---|
| `srv-staging` | 2 vCPU / 4 Go / 40 Go SSD | 4 / 8 Go / 80 Go | La version avant la production : même pile, migrations répétées. |
| `srv-prod` | 2 vCPU / 4 Go / 60 Go SSD | 4 / 8 Go / 160 Go | Caddy, API et ses workers, web, PostgreSQL + pgvector, passerelle Anki, sauvegardes. |
| `srv-services` | 1 vCPU / 1 Go / 20 Go | 2 / 2 Go / 50 Go | Service de synchronisation Anki existant ; reçoit les sauvegardes chiffrées. |
| `srv-secondbrain` | 1 vCPU / 1 Go / 50 Go | 2 / 2 Go / 100 Go | Le coffre Markdown, synchronisé par `notes-sync -watch`. |

Chaque environnement tient sur un hôte ([ADR 0003](doc/adr/0003-single-host-first.md), [ADR 0004](doc/adr/0004-hosts-and-tailnet-only-access.md)) ; mesurée le 11 octobre 2026, toute la pile utilise environ 300 Mo de mémoire et sert 700 requêtes par seconde ([ADR 0021](doc/adr/0021-single-host-measured.md)). Un hôte de base de données séparé ne se justifie que si une mesure le montre.

> [!NOTE]
> Les petites configurations supposent des workers bornés, un pool de connexions, la pagination et aucune inférence de modèle locale. Les appels aux modèles externes sont limités en débit et en budget.

## Démarrage rapide

> [!IMPORTANT]
> Axon 1.0 est une version candidate : chaque domaine décrit ici fonctionne aujourd’hui. Pour l’utiliser pour de vrai, suivez le **[guide d’installation](doc/install.md)** ; puis lisez le **[guide d’utilisation](doc/user-guide.md)**, comment **[mettre à jour](doc/upgrading.md)** et la **[politique de support](doc/support.md)** (en anglais). Notes de version : [CHANGELOG.md](CHANGELOG.md).

### Prérequis

- Docker Engine et Compose v2 ; Git ; GNU Make pour les vérifications.
- Seulement si vous utilisez les agents : une clé Anthropic, OpenAI, Mistral, Google Gemini ou DeepSeek, ou un serveur de modèles local (Ollama, LM Studio, vLLM). Chaque personne peut choisir le sien dans l’application.
- Facultatif : identifiants TMDB, Open Library, USDA et Open Food Facts, ou leurs points d’accès publics selon leurs conditions.

```bash
git clone git@github.com:thoutmose/axon.git   # dépôt privé
cd axon
cp .env.example .env
docker compose up -d --build --wait
docker compose exec api axon migrate up
```

Ouvrez ensuite <http://localhost:3000>. L’API est servie sous `/api` (par exemple `/api/healthz` et `/api/readyz`). La pile n’écoute que sur `127.0.0.1`. Pour l’essayer avec des données d’exemple, mettez `AXON_DEMO=1` dans `.env`, redémarrez, lancez `docker compose exec api axon demo reset`, puis cliquez sur **Try the demo**.

**Référence de l’API.** Le document OpenAPI 3.1 est à `/api/v1/openapi.yaml` ([source](api/openapi/openapi.yaml)), et une interface Swagger interactive est à `/api-docs/index.html`, servie par Axon lui-même.

### Vérifications

| Commande | Ce qu’elle lance |
|---|---|
| `make check` | `gofmt`, golangci-lint, tests unitaires Go ; `svelte-check`, Prettier, Vitest (Node 24 en conteneur) ; parité des README anglais et français |
| `make test-integration` | Tests de l’API, du contrat OpenAPI, des évaluations d’agents et des migrations sur PostgreSQL 16 + pgvector |
| `make up` / `make down` | Construire et démarrer, ou arrêter, la pile locale |

## Configuration

| Variable | Obligatoire | Exemple | Rôle |
|---|---|---|---|
| `AXON_BASE_URL` | Oui | `https://axon.example` | URL canonique et origine des retours. |
| `DATABASE_URL` | Oui | `postgres://...` | Connexion PostgreSQL ; à stocker comme secret. |
| `AXON_ENCRYPTION_KEY` | Prod | `openssl rand -hex 32` | Chiffre les secrets TOTP au repos ; ne jamais la committer. |
| `AXON_REGISTRATION` | Non | `first-user` | `first-user` : seul le premier compte peut s’inscrire ; `open` : toute personne qui atteint Axon. |
| `AXON_DEMO` | Non | `1` | Propose un compte de démonstration partagé, avec des données d’exemple remises à zéro chaque nuit, et montre la bande-annonce d’une minute sur la page d’accueil. |
| `WEBAUTHN_RP_ID` | Prod | `axon.example` | Identifiant de la partie de confiance pour les clés d’accès. |
| `ANTHROPIC_API_KEY` | Facultatif | secret | Clé Anthropic par défaut du serveur pour les agents (chacun peut aussi choisir son fournisseur). |
| `AXON_ANTHROPIC_OAUTH` | Facultatif | config | `1` pour utiliser un profil `ant auth login` au lieu d’une clé. |
| `TMDB_API_KEY` | Facultatif | secret | Recherche des films et séries (votre propre clé TMDB non commerciale, clé v3 ou jeton de lecture v4). |
| `USDA_API_KEY` | Facultatif | secret | Recherche d’aliments dans USDA FoodData Central (clé data.gov gratuite). |
| `AXON_CALENDAR_PRIVATE` | Facultatif | valeur | `1` autorise des agendas suivis sur votre propre réseau (un Nextcloud sur le réseau local ou le tailnet) ; sinon, seules les adresses HTTPS publiques sont lues. |
| `AXON_SMTP_URL` | Facultatif | secret | E-mails de rappel par votre serveur de messagerie : `smtp://user:pass@host:587` ou `smtps://user:pass@host:465`. |
| `AXON_SMTP_FROM` | Facultatif | valeur | L’adresse d’expédition des e-mails de rappel. |
| `AXON_GOOGLE_CLIENT_ID`, `AXON_GOOGLE_CLIENT_SECRET` | Facultatif | secret | Connexion avec Google ; aussi `AXON_MICROSOFT_*` (avec `AXON_MICROSOFT_TENANT`), `AXON_DISCORD_*`, `AXON_SLACK_*`. Adresse de retour : `AXON_BASE_URL/api/v1/auth/oauth/<fournisseur>/callback`. |
| `AXON_APPLE_CLIENT_ID`, `AXON_APPLE_TEAM_ID`, `AXON_APPLE_KEY_ID`, `AXON_APPLE_PRIVATE_KEY` | Facultatif | secret | Connexion avec Apple : un Services ID et la clé `.p8` (ou `AXON_APPLE_PRIVATE_KEY_FILE`). |
| `AXON_OIDC_ISSUER`, `AXON_OIDC_CLIENT_ID`, `AXON_OIDC_CLIENT_SECRET`, `AXON_OIDC_NAME` | Facultatif | secret | Connexion avec n’importe quel fournisseur OpenID Connect (Authentik, Keycloak, GitLab…). |
| `AXON_CONTACT` | Facultatif | `vous@example.org` | Comment Open Library peut joindre l’opérateur (envoyé dans le User-Agent) ; par défaut `AXON_BASE_URL`. |
| `ANKI_SYNC_URL` | Facultatif | URL privée | Adresse suggérée du serveur anki-sync-server pour le pont Anki (chacun saisit son propre compte). |

Ne mettez jamais de secrets dans du Markdown, des tickets, des prompts, des captures d’écran ou des journaux. Utilisez SOPS/age et un gestionnaire de secrets adapté au déploiement.

## Sécurité et RGPD

Axon suit l’OWASP ASVS comme base de vérification et les recommandations CIS pour les hôtes. Le trafic entre serveurs passe par WireGuard ou Tailscale ; l’entrée publique se limite au proxy inverse. Les clés d’accès WebAuthn et le TOTP protègent les comptes ; les sessions tournent et se révoquent. Les contrôles d’appartenance par utilisateur s’appliquent à chaque frontière de l’API.

Les contrôles de confidentialité comprennent les registres de consentement, la limitation des finalités, les durées de conservation, l’export, les demandes d’effacement, les journaux d’audit, l’inventaire des sous-traitants et le masquage des prompts et des journaux. Les sauvegardes sont chiffrées, avec des délais de suppression documentés. Les données de santé et d’apprentissage sont sensibles ; la visibilité par défaut est privée.

> [!NOTE]
> La conformité au RGPD est un travail d’exploitation continu, pas un badge. Demandez un avis juridique adapté à votre déploiement et à votre juridiction.

## Agents IA

| Agent | Rôle | Garde-fou | Statut |
|---|---|---|---|
| Gestionnaire de tâches | Trier et planifier le travail. | Aperçu avant de changer une échéance. | ✅ |
| Objectifs/Jalons | Découper un résultat en étapes mesurables. | Aucun engagement ni date en silence. | ✅ |
| Habitudes | Suggérer déclencheurs, enchaînements, reprises et revues. | Ni honte ni pression manipulatrice. | ✅ |
| Projets perso | Analyser des dépôts en epics, tickets et tâches. | Lecture seule ; les tickets sont des propositions. | ✅ |
| Revue de code | Relire diffs, tests, sécurité et maintenabilité. | Peut commenter après approbation ; ne peut jamais approuver ni fusionner. | ✅ |
| Tuteur | Interroger, rechercher et rédiger des parcours. | Chaque compétence cite une source de la bibliothèque ou d’un site autorisé ; vous approuvez. | ✅ |
| Ressources | Trouver et classer des ressources citées. | Sites autorisés uniquement, robots.txt respecté, 15 pages par exécution, des liens et pas des copies. | ✅ |
| Enseignant | Produire des exercices de récupération et des cartes mémoire relus. | Ne lit que les notes de lecture et permanentes. | ✅ |
| Médias | Suggérer quoi voir ou lire ensuite, et proposer de nouveaux titres. | Ne voit que votre liste (ni critiques, ni texte de catalogue) ; chaque ajout demande votre approbation. | ✅ |
| Coach sportif | Proposer les prochaines séances depuis votre journal, en tâches à approuver. | Consentement propre (`health_ai`) ; alertes renvoyées sans appel au modèle ; non médical. | ✅ |
| Nutritionniste | Suggestions douces depuis votre journal alimentaire, en tâches à approuver. | Consentement propre ; jamais d’objectif calorique, de jeûne ni de diagnostic (imposé, pas seulement demandé). | ✅ |
| Chef et nutritionniste | Des plans de repas pour les jours choisis, dans votre budget, vos restrictions et allergies, vos contraintes et votre temps de cuisine, au plus près de vos besoins quotidiens, avec recettes, liste de courses et coût estimé ; enregistrés en note une fois approuvés. | Consentement propre ; restrictions strictes, rappel de vérifier les étiquettes ; aucun objectif quotidien sous 1 200 kcal (imposé) ; alertes renvoyées. | ✅ |

Tous les agents partagent une constitution : simple, explicite, petites modifications, lire avant d’écrire, raisonner sur des données, tout tester, ne pas faire le malin. Les compétences sont des fichiers `SKILL.md` versionnés. Chaque tâche se termine par une réflexion, des enseignements, des preuves d’évaluation et, si besoin, une approbation humaine.

## Gamification

Les mécanismes de progression utilisent XP, niveaux, séries, badges, récompenses variables, engagements et gels de série achetés avec des points gagnés. La conception est optionnelle et transparente : on peut masquer les surfaces compétitives, voir comment les points sont calculés et reprendre après un jour manqué sans humiliation. Elle s’inspire de *Hooked*, d’Octalysis et de la théorie de l’autodétermination ; la contrainte éthique passe avant les indicateurs d’engagement.

## Méthode de développement

1. Prendre le prochain ticket prêt dans [BACKLOG.md](doc/BACKLOG.md) ; ne pas travailler sur plusieurs Epics à la fois.
2. Créer une branche ciblée et écrire d’abord les tests ou les exemples d’acceptation.
3. Livrer une petite tranche verticale ; garder les migrations réversibles et les API documentées.
4. Lancer formatage, lint et tests unitaires, d’intégration, de contrat, d’accessibilité et de sécurité.
5. Ouvrir une PR/MR en Conventional Commits avec preuves, risques, retour arrière et documentation.
6. Demander une revue indépendante et une approbation humaine, fusionner, déployer en préproduction et observer avant la promotion.

## Tests et exploitation

- **Tests :** tests unitaires Go, tests d’intégration PostgreSQL, tests de contrat OpenAPI, tests navigateur Playwright, contrôles d’accessibilité, évaluations des agents, répétition des migrations et exercices de restauration des sauvegardes.
- **Observabilité :** métriques Prometheus, tableaux de bord Grafana, journaux Loki, traces OpenTelemetry, identifiants de corrélation, profondeur des files, latence des fournisseurs, coût en jetons et décisions d’agents masquées.
- **Fiabilité :** définir SLO, alertes, RPO/RTO, limites de débit, clés d’idempotence, disjoncteurs et procédures avant de déclarer une fonctionnalité terminée.
- **Chaîne d’approvisionnement :** Renovate, SBOM, analyses Trivy, images épinglées, artefacts signés quand c’est possible, et provenance inspirée de SLSA.

## Feuille de route

| Phase | Résultat | Backlog, dans l’ordre |
|---|---|---|
| Fondations | Déploiement sécurisé, CI, observabilité, authentification, confidentialité. | EP-00 → 01 → 02 → 03 → 04 |
| Graphe personnel | Liens, tâches, notes, objectifs, tableau de bord, sans IA. | EP-05 → 06 → 07 → 09 → 11 → 13 → 08 |
| Croissance guidée | Agents, habitudes, projets, parcours, espace de notes, agents de livraison et Tuteur. | EP-19 → 12 → 22 → 23 → 24 → 25 → 10 → 26 → 14 → 27 |
| Usage quotidien | Démo, docs bilingues, tableau de bord, réglages, raccourcis, avatars, Atomic Habits, Pomodoro, apprentissage et Anki, gamification, coffre en direct, graphe de connaissances, espaces thématiques, finances, portabilité. | EP-28 → 29 → 30 → 41 → 31 → 32 → 33 → 34 → 35 → 16 → 42 → 15 → 36 → 43 → 39 → 38 → 37 → 40 |
| Bêta vie entière | Médias, santé, nutrition, analyses. | EP-17 → 18 → 45 → 46 → 44 → 20 |
| 1.0 | Durcissement, restauration, documentation, chemin de mise à jour. | EP-21 |
| 1.1 | Demandes du propriétaire : graphe Zettelkasten, tout dans le coffre, diagrammes, images dans les notes, moteur d’apprentissage et de mémoire, agents sur chaque page, application installable, rappels, saisie rapide, agendas externes, revue hebdomadaire, imports, foyer, recherche par le sens, Aujourd’hui personnalisable, actions groupées, corbeille et annulation, partager vers Axon, rappels de tâches, code propre. | EP-47 → 48 → … → 69 |

État actuel : **EP-00 à EP-16, EP-19, EP-22 à EP-35, EP-36, EP-41, EP-42, EP-43, EP-39, EP-38, EP-37, EP-40, EP-17, EP-18, EP-45, EP-46, EP-44, EP-20, EP-21 et EP-47 à EP-68 terminés et fusionnés ; EP-69 à EP-87 (code propre, performances, durcissement, toile d’araignée des compétences, musique de concentration, bande-annonce, tout en français), EP-88 à EP-90 (besoins quotidiens, photo d’un repas, le Chef et nutritionniste), et EP-91 à EP-105 (le journal quotidien, ce qui vous aide, les jours du journal dans le coffre, seulement ce qui vous sert, le parcours top 5 %, le ciel qui suit le soleil, le parcours dans l’agenda, chaque page son image, une année lisible, les modèles du coffre, l’eau, des images corrigées, la pratique, les certifications, les heures d’étude) sont en revue ; la 1.0 attend la revue de sécurité externe et une installation observée.** Pour découvrir Axon sans compte, utilisez **Try the demo** sur la page d’accueil : un compte partagé avec des données d’exemple, remis à zéro chaque nuit. Exploitation : [doc/operations.md](doc/operations.md). Détails dans [doc/BACKLOG.md](doc/BACKLOG.md#delivery-order).

## Contribuer

Les contributions sont bienvenues quand elles renforcent l’autonomie des utilisateurs, l’accessibilité, la confidentialité ou la maintenabilité. Lisez [CONTRIBUTING.md](CONTRIBUTING.md) et le [code de conduite](CODE_OF_CONDUCT.md), ouvrez un ticket avec le contexte et les critères d’acceptation, et gardez les modifications petites. Pour une vulnérabilité, suivez [SECURITY.md](SECURITY.md) plutôt que d’ouvrir un ticket public.

## Licence et remerciements

Axon est sous licence **AGPL-3.0-or-later**, pour que les modifications hébergées restent disponibles pour la communauté. Le texte de référence est dans [LICENSE](LICENSE).

Remerciements et lectures : [roadmap.sh](https://roadmap.sh/), [OSSU](https://github.com/ossu/computer-science), [Anki sync server](https://github.com/ankicommunity/anki-sync-server), [API TMDB](https://developer.themoviedb.org/docs/getting-started), [API Open Library](https://openlibrary.org/developers/api), [Open Food Facts](https://world.openfoodfacts.org/data), [OWASP ASVS](https://owasp.org/www-project-application-security-verification-standard/), [Twelve-Factor App](https://12factor.net/), [livre SRE de Google](https://sre.google/sre-book/table-of-contents/), [FSRS](https://github.com/open-spaced-repetition/free-spaced-repetition-scheduler) et la [documentation d’Anthropic](https://docs.anthropic.com/en/docs).

Données de catalogue : les données et couvertures des livres viennent d’[Open Library](https://openlibrary.org/) ; les synonymes anglais d’[Open English WordNet](https://github.com/globalwordnet/english-wordnet) (CC BY 4.0) ; les données alimentaires d’[Open Food Facts](https://world.openfoodfacts.org/) (ODbL) et d’[USDA FoodData Central](https://fdc.nal.usda.gov/) (domaine public). This product uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.

Livres qui façonnent le produit : *Understanding How We Learn* (Weinstein, Sumeracki, Caviglioli), *Aventures au cœur de la mémoire* (*Moonwalking with Einstein*, Foer), *Make It Stick*, *Atomic Habits* (Clear), *Hooked* (Eyal), *The Octalysis Framework* (Chou), *Building a Second Brain* (Forte), *How to Take Smart Notes* (Ahrens) et *Designing Data-Intensive Applications* (Kleppmann).

---

<p align="center"><sub>Axon — construire une vie que l’on comprend, un petit lien à la fois.</sub></p>
