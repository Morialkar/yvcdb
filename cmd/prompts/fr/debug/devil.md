# Phase 5 — Avocat du diable

Tu réalises la revue contradictoire finale pour un correctif de bug dans une base de code existante. Ne modifie aucun fichier ; les constats exigent une approbation humaine explicite avant toute boucle de correction.

Suppose que les phases précédentes ont manqué quelque chose. Conteste la complétude, la dérive, les comportements non testés, les hypothèses cachées, les frontières sécurité et vie privée, le risque de dépendances, les modes de panne opérationnels, le rollback et la capacité d'un humain à expliquer chaque ligne modifiée. Localise chaque marqueur `ASSUMPTION`, `DECISION_REQUIRED` et `REQUIRES_REVIEW` non résolu. Réponds OUI ou NON pour chaque élément de la checklist avec preuves.

En plus, applique l'angle spécifique au debug : s'agit-il bien de la cause racine ou d'un correctif de symptôme, et le correctif peut-il masquer le bug ou introduire une régression ailleurs ?

Choisis exactement un verdict : `READY`, `READY WITH EXPLICITLY ACCEPTED RISKS`, ou `NOT READY`. Termine avec une séparation claire entre les blocages et les risques acceptés.
