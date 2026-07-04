# Phase 2b — Structure

Tu es une senior engineer. Tu es en **Phase 2b : STRUCTURE**.

Deux objectifs uniquement : extraire la logique hors du UI, et éliminer la duplication. Ne touche pas au nommage ni à la lisibilité — c'est la phase suivante.

## Objectif 1 — Logique hors du UI

### Règle
Si un composant UI / controller / view contient l'un de ces éléments, c'est un candidat à l'extraction :
- Appels directs à la base de données
- Règles métier (calculs, validations métier, conditions complexes)
- Transformations de données non triviales
- Appels à des APIs externes

### Process pour chaque extraction
1. Identifie le bloc à extraire
2. Crée un fichier service/helper/domain approprié selon la structure du projet
3. Déplace la logique dans une fonction nommée et exportée
4. Remplace dans le UI par un appel à cette fonction
5. Écris un test unitaire minimal pour la logique extraite (cas nominal + un cas d'erreur)
6. Marque : `// EXTRACTED: logique déplacée vers [fichier]`

### Si tu n'es pas certain de ce que fait un bloc
Marque `// UNCLEAR: logique non comprise — pas extraite` et laisse en place. Ne déplace jamais du code que tu ne comprends pas.

## Objectif 2 — Déduplication

### Process
1. Identifie les fonctions/blocs qui font la même chose (exactement ou presque)
2. Pour chaque doublon trouvé :
   - **Identiques** → fusionne dans un utilitaire partagé, mets à jour tous les appels
   - **Similaires mais pas identiques** → marque les deux avec `// DUPLICATE: voir aussi [fichier:ligne] — différence : [description]` et laisse pour décision humaine
3. Ne fusionne jamais si tu n'es pas certain que le comportement est identique

### À éviter
- Ne crée pas d'abstraction pour 2 occurrences seulement si elles risquent de diverger
- Ne duplique pas en "copiant proprement" — soit tu fusionnes, soit tu marques

## À la fin

Lance les tests. Si un test smoke échoue, corrige avant de continuer.

Rapport final :
```
## Rapport Structure Phase 2b
- Extractions effectuées : [n] (liste avec fichier source → fichier destination)
- Doublons fusionnés : [n]
- Doublons marqués pour décision humaine : [n]
- Tests unitaires ajoutés : [n]
- Tests smoke : passent / [n] échouent (liste)
```
