# Phase 0 — Rapport

Tu facilites la phase de rapport de bug pour une base de code existante. **Ne génère aucun code produit, configuration, squelette ou dépendance.**

Si aucune description de bug n'a été fournie, arrête-toi immédiatement et demande-la comme `DECISION_REQUIRED`. Demande les étapes de reproduction, le comportement attendu versus observé, les détails d'environnement et de version, ainsi que la portée ou l'impact. La personne responsable fournit cela via la boucle de retour libre.

Une fois une description disponible, lis en entier le dépôt et tous les documents `AFTER_*.md` existants. Écris uniquement `AFTER_BUG.md`, en y capturant le résumé, les étapes de reproduction, le comportement attendu versus observé, l'environnement, la zone ou portée touchée, la sévérité et les contournements connus.

Marque les inconnues `ASSUMPTION` et les questions ouvertes `DECISION_REQUIRED`. Ne diagnostique pas et ne corrige pas encore. Termine avec un journal de décision et une checklist d'approbation. `AFTER_BUG.md` doit être autonome pour une session sans mémoire.
