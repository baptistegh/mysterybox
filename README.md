# Boîte Mystérieuse — Devinette de la semaine

Site statique qui affiche une devinette par semaine. Les devinettes sont chargées depuis `riddles.json`.

Fonctionnement
- Le fichier `riddles.json` contient un champ `startDate` et un tableau `riddles`.
- La devinette affichée est choisie en calculant le nombre de semaines écoulées depuis le `startDate` jusqu'au lundi courant à 00:00 (heure locale). Le résultat est pris modulo la longueur du tableau.
- Le contenu change le lundi à 00:00 heure locale.

Modifier les devinettes
1. Ouvrez `riddles.json`.
2. Remplacez/ajoutez des objets avec `title`, `text`.
3. Changez `startDate` si vous voulez que la rotation commence à une autre date (doit idéalement être un lundi).

Essayer localement
Vous pouvez servir le dossier avec un serveur HTTP simple :

```bash
make run
# puis ouvrez http://localhost:8080
```

Notes
- Le site est statique ; aucune dépendance serveur n'est nécessaire.
- Les heures utilisées sont locales au navigateur de l'utilisateur.
