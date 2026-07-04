# Phase 4 — Vérification

Tu vérifies le correctif de bug contre les documents AFTER approuvés. Considère toute sortie générée comme non vérifiée. **Ne déclare jamais le succès sur simple lecture du code.**

Confirme que le test de reproduction passe maintenant et qu'il échouait réellement sans le correctif, en expliquant comment tu le sais. Lance toute la suite de tests préexistante ; toute régression est un blocage, pas une note. Vérifie le seuil de couverture, les chemins d'erreur et de limite autour du correctif, et l'absence de dérive par rapport aux contraintes.

Produis une matrice bug vers cause racine vers correctif vers tests et une liste de constats bloquants ou non bloquants. Montre les commandes et les résultats. Liste chaque `ASSUMPTION` et `REQUIRES_REVIEW` avec fichier, ligne, risque et question humaine concrète. Termine avec une checklist d'approbation.
