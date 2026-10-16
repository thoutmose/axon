# Votre premier après-midi avec Axon

Un Axon neuf est vide exprès : aucune note, tâche, objectif ou habitude de quelqu’un d’autre. Il fournit en revanche l’échafaudage : des modèles de notes (éphémère, de lecture, permanente, film, série, livre, projet, domaine, ressource, quotidienne, hebdomadaire, les tests de rappel…), une feuille de route de bibliothèque avec ses cours, les protocoles de Huberman, et des agents qui proposent sans jamais agir sans vous. En trois à quatre heures, il devient le vôtre. *English version: [getting-started.md](getting-started.md).*

Chaque agent fonctionne de la même façon : il **propose**, vous **lisez l’aperçu**, vous **approuvez ou refusez**. Rien ne change avant votre accord ([ADR 0008](adr/0008-agents-propose-people-approve.md)).

## Avant de commencer (15 minutes)

1. Installez Axon ([guide d’installation](install.md)) et inscrivez-vous avec une clé d’accès. Gardez les codes de récupération.
2. **Compte → Préférences** : langue, fuseau horaire, thème et icônes.
3. **Compte → Préférences → Fournisseur d’IA** (ou la page **Agents**) : une clé (Anthropic, OpenAI, Mistral, Gemini, DeepSeek) ou un modèle local (Ollama, LM Studio, vLLM). Sans cela, Axon fonctionne entièrement ; seuls les agents attendent.
4. Dites à quoi Axon doit vous servir sur l’écran d’accueil : le menu n’affiche que cela, puis s’enrichit à l’usage.

## Heure 1 — Où vous allez

1. **Grandir → Feuilles de route** : démarrez la feuille de route de la bibliothèque (un large plan d’ingénierie avec ses cours), ou demandez au **Tuteur** d’écrire la vôtre (« Je veux devenir ingénieur backend en 18 mois, 10 heures par semaine »). Passez l’évaluation des compétences pour démarrer à votre niveau.
2. **Objectifs** : demandez à l’agent **Objectifs** de découper le résultat visé en jalons datés que vous pouvez tenir.
3. **Projets** : un projet par résultat avec une ligne d’arrivée, à partir d’un modèle (logiciel, apprentissage, événement, maison, écriture). Avec un dépôt GitHub, l’agent **Projets perso** en tire des epics et des tickets.

## Heure 2 — Ce que vous faites chaque jour

1. **Habitudes** : demandez à l’agent **Habitudes** deux ou trois habitudes greffées sur ce que vous faites déjà, chacune avec sa version de deux minutes. Reliez une habitude de lecture à son livre (« Lire 20 pages » → le livre) : la cocher fait avancer les pages.
2. **Tâches** : appuyez sur `n` n’importe où pour noter ; le **Gestionnaire de tâches** range la boîte de réception en prochaines actions par contexte. Le tableau GTD montre cinq cartes par colonne, le reste à un clic.
3. **Concentration** : réglez vos sessions Pomodoro et la musique.
4. **Santé** (facultatif, avec son propre consentement) : votre profil pour les besoins quotidiens ; le **Chef & Nutritionniste** planifie les repas selon votre budget et vos restrictions. La toile de vie note la journée.

## Heure 3 — Ce que vous apprenez et gardez

1. **Notes** : connectez votre coffre (Obsidian ou tout dossier Markdown), ou gardez-le dans Axon. Les modèles arrivent dans `Templates/` ; modifiez-les comme n’importe quelle note.
2. Écrivez des notes de lecture en lisant ou en regardant : depuis un film, une série ou un livre, **Écrire une note** part de son modèle. Le **Professeur** transforme les notes de lecture et permanentes en exercices de rappel et en cartes (synchronisées avec Anki si vous voulez).
3. **Veille** : suivez quelques sources, des chaînes YouTube ou vos podcasts Audiobookshelf ; le résumé retient le meilleur chaque matin.

## La dernière demi-heure — Rester en contact

1. **Résumé** : choisissez ses heures et où il part (e-mail, Slack, Discord, Teams, Telegram, ntfy), puis envoyez un test.
2. **Budget** (facultatif) : importez un CSV bancaire, ajoutez des règles pour les bénéficiaires de chaque mois ; Rapports montre où est passé l’argent.
3. Mettez la **revue hebdomadaire** dans votre agenda : Axon vous mène d’une tête vide à la semaine suivante.

Ensuite, les agents de chaque page (**Demander à un agent**) sont le moyen le plus rapide de mettre en place quoi que ce soit : décrivez ce que vous voulez, lisez ce qu’ils proposent, gardez ce qui vous convient.
