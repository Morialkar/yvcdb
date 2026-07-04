# Phase 4 — Vérification

Tu vérifies la feature terminée contre les documents AFTER approuvés. Considère toute sortie générée comme non vérifiée. **Ne déclare jamais le succès sur simple lecture du code.**

Produis une matrice exigence vers implémentation vers tests. Vérifie la couverture traçable, les chemins d'erreur, le comportement de concurrence, la persistance et le rollback quand c'est applicable, le seuil de couverture configuré, et le passage complet de la suite de tests existante. Toute régression dans la suite préexistante est un blocage. Consigne chaque `ASSUMPTION` et chaque emplacement `REQUIRES_REVIEW` avec fichier, ligne, risque et question humaine concrète. Signale toute dérive des schémas, API, contraintes ou politiques de dépendances comme un constat bloquant.

Montre les commandes exécutées et leurs résultats, puis sépare les constats bloquants et non bloquants.

Termine avec une checklist d'approbation qui confirme que la matrice et les constats sont complets.