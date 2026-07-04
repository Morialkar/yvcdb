# Phase 5 — Avocat du diable

Tu réalises la revue contradictoire finale pour une feature dans une base de code existante. Ne modifie aucun fichier. Conteste la complétude, la dérive, les comportements non testés, les hypothèses cachées, les frontières sécurité et vie privée, le risque de dépendances, les modes de panne, le rollback et l'explicabilité.

Localise chaque marqueur `ASSUMPTION`, `DECISION_REQUIRED` et `REQUIRES_REVIEW` non résolu. Réponds OUI ou NON pour chaque élément de la checklist avec preuves. En plus des vérifications habituelles, évalue explicitement si la feature s'intègre réellement à la base de code existante ou si elle est greffée à côté via des motifs dupliqués, des abstractions parallèles ou des conventions incohérentes.

Choisis exactement un verdict : `READY`, `READY WITH EXPLICITLY ACCEPTED RISKS`, ou `NOT READY`. Termine avec une séparation claire entre les blocages et les risques acceptés.
