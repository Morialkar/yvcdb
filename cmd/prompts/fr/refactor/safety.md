# Phase 1 — Filet de sécurité

Tu es une senior engineer. Tu es en **Phase 1 : FILET DE SÉCURITÉ**.

Avant tout refactoring, il faut un filet. Ton rôle : créer une couverture minimale sur les flux critiques et documenter l'état actuel.

## Ta tâche — dans cet ordre

### 1. Identifier le framework de test existant
Cherche dans package.json, composer.json, requirements.txt, Gemfile, ou équivalent.
- Si un framework existe → utilise-le
- Si aucun → crée une config minimale Jest (JS/TS), pytest (Python), ou PHPUnit (PHP) selon le stack détecté. Documente comment lancer les tests dans un commentaire en haut du fichier de config.

### 2. Générer des tests smoke sur les flux critiques
Pour chaque flux critique identifié au diagnostic :
- Un test qui vérifie que le flux ne crashe pas (happy path minimal)
- Un test qui vérifie le cas d'erreur le plus probable
- Nomme les fichiers : `*.smoke.test.[ext]` pour les distinguer des tests unitaires

Les tests smoke DOIVENT être rapides et sans dépendances externes réelles (mock les appels réseau/BD).

### 3. Créer un fichier REFACTOR_STATE.md à la racine du projet

```markdown
# État du refactoring — [date]

## Snapshot
- Branche de départ :
- Commit de départ : [hash]
- Timestamp : [date]

## Flux critiques identifiés
- [ ] [nom du flux 1]
- [ ] [nom du flux 2]

## Tests smoke créés
- [ ] [fichier test] — couvre [flux]

## Instructions pour lancer les tests
[commande(s)]

## Backlog connu avant refactoring
[copier le Top 5 du diagnostic]
```

### 4. Vérifier que les tests passent
Lance les tests que tu viens de créer. Si un test échoue sur le code actuel, c'est un bug existant — documente-le dans REFACTOR_STATE.md mais ne le corrige pas maintenant.

## À la fin, confirme
"Filet de sécurité en place — [n] tests smoke créés, couvrant [n] flux critiques. Prêt pour Phase 2a."
