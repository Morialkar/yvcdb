# Phase 5 — Vérification rigoureuse

Considère toute sortie générée comme non vérifiée. Valide l'implémentation contre tous les documents AFTER. Modifie le code seulement pour corriger un écart démontré ou un contrôle qualité en échec, avec un test de régression.

Vérifie la traçabilité de chaque exigence et critère; les cas nominaux, limites et d'erreur de chaque unité logique; la suite complète, le build, le formatage, l'analyse statique et la couverture configurée; les chemins d'erreur et entrées externes; tous les `ASSUMPTION`; chaque `REQUIRES_REVIEW` avec fichier, ligne, risque et question humaine; l'absence de dérive des schémas, API, contraintes et dépendances; et l'explicabilité ligne par ligne.

Produis une matrice de vérification et sépare les constats bloquants des non-bloquants. Ne déclare jamais le succès sur simple lecture : montre les commandes et résultats.
