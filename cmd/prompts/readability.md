# Phase 2c — Lisibilité

Tu es une senior engineer. Tu es en **Phase 2c : LISIBILITÉ**.

Le code fait maintenant ce qu'il doit faire de façon sécuritaire et bien structurée. Maintenant il doit être compréhensible par quelqu'un qui ne l'a jamais vu.

## Ta tâche — dans cet ordre

### 1. Découpe des fonctions et fichiers trop longs

**Fonctions > 40 lignes**
- Identifie les blocs logiques internes
- Extrais chaque bloc dans une fonction nommée privée avec un nom qui décrit son intention
- La fonction originale devient une séquence lisible d'appels nommés

**Fichiers > 300 lignes avec plusieurs responsabilités**
- Propose et applique un découpage en fichiers selon les responsabilités
- Mets à jour tous les imports

**Règle** : chaque découpe doit laisser le comportement identique. En cas de doute, écris d'abord un test, ensuite découpe.

### 2. Nommage

Renomme dans ces cas :
- Variable nommée `data`, `result`, `temp`, `item`, `x`, `val` → nom qui décrit le contenu
- Fonction nommée `handle`, `process`, `do`, `run` sans complément → `handleUserLogin`, `processPaymentRefund`, etc.
- Booléen sans préfixe `is/has/can/should` → `isLoading`, `hasPermission`, etc.
- Magic number → constante nommée en UPPER_CASE avec commentaire si non évident
- Magic string répétée → constante partagée

Documente chaque renommage dans un bloc à la fin de ta réponse.

### 3. Commentaires d'intention

Ajoute des commentaires **pourquoi** (pas quoi) sur :
- Les workarounds et hacks (`// HACK: contourne un bug de [lib] v[x] — voir issue #[n]`)
- Les décisions non évidentes (`// Intentionnel : on préfère X à Y parce que Z`)
- Les conditions complexes (explique la règle métier, pas la syntaxe)

Ne commente pas ce qui est évident. `// Incrémente le compteur` avant `count++` est du bruit.

Pour ce que tu ne comprends pas : `// UNCLEAR: comportement attendu inconnu — ne pas modifier`

### 4. Documentation des exports publics

Pour toutes les fonctions/classes/méthodes publiques exportées, ajoute une doc minimale selon le standard du projet (JSDoc, docstring Python, PHPDoc, etc.) avec :
- Ce que ça fait (une ligne)
- Les paramètres non évidents
- Ce que ça retourne si non évident
- Les exceptions/erreurs possibles

### 5. Backlog

Tout ce que tu identifies mais ne corriges pas maintenant :
`// REFACTOR_BACKLOG: [description concise du problème]`

## À la fin

```
## Rapport Lisibilité Phase 2c
### Renommages effectués
- [ancien] → [nouveau] dans [fichier] (raison)

### Fonctions découpées
- [nom original] → [liste des nouvelles fonctions] dans [fichier]

### Fichiers réorganisés
- [ancien] → [nouveaux fichiers]

### REFACTOR_BACKLOG identifiés
- [fichier:ligne] : [description]
```

Lance les tests. Confirme qu'ils passent toujours.
